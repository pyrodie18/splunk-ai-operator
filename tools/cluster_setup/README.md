# EKS Cluster Setup for Splunk AI Platform

This directory contains scripts to set up a complete end-to-end Splunk AI Platform deployment on AWS EKS.

## Overview

The `eks_cluster_with_stack.sh` script provides idempotent resource creation for:
- EKS cluster with CPU and GPU node groups
- Storage (EBS CSI, S3 bucket)
- Cluster autoscaler
- Monitoring (Prometheus)
- OpenTelemetry Operator
- Ray Operator
- Splunk Enterprise Operator
- Splunk AI Operator
- Splunk Standalone instance
- AIPlatform custom resource

## Prerequisites

### Required Tools
- `aws` CLI (configured with credentials)
- `eksctl`
- `kubectl`
- `helm`
- `git`
- `jq`
- `yq` (optional, recommended for robust YAML parsing)

### AWS Prerequisites
- Valid AWS credentials (via environment variables or AWS SSO profile)
- VPC with private and public subnets
- Appropriate IAM permissions to create EKS clusters, IAM roles, S3 buckets, etc.

### Required Files
Place these files in the `tools/cluster_setup` directory:
- `splunk-operator-cluster.yaml` - Splunk Enterprise Operator manifest
- `artifacts.yaml` - Splunk AI Operator manifest
- (Optional) `Splunk_AI_Assistant_Cloud.tgz` - Splunk AI Assistant app

## Configuration

All configuration is centralized in `cluster-config.yaml`. Edit this file to customize your deployment.

### Key Configuration Sections

#### 1. Cluster Configuration
```yaml
cluster:
  name: "test-new-ai"           # EKS cluster name (DNS-1123 compliant)
  region: "us-west-2"             # AWS region
  k8sVersion: "1.31"              # Kubernetes version
  subnets:
    private:
      - id: "subnet-xxxxx"        # Private subnet IDs
        az: "us-west-2c"
    public:
      - id: "subnet-yyyyy"        # Public subnet IDs
        az: "us-west-2b"
```

#### 2. Node Groups
```yaml
nodeGroups:
  cpu:
    enabled: true
    instanceType: "m5.xlarge"
    desiredCapacity: 4
    minSize: 2
    maxSize: 8
    volumeSize: 500
    volumeType: "gp3"

  gpu:
    enabled: true
    instanceType: "g6e.12xlarge"
    desiredCapacity: 2
    minSize: 2
    maxSize: 4
    volumeSize: 1000
    volumeType: "gp3"
```

#### 3. Storage
```yaml
storage:
  s3Bucket: "ai-platform-dev-vivekr"  # Must be globally unique
  storageClass: "gp3"
  vectorDbSize: "50Gi"
```

#### 4. AI Platform
```yaml
aiPlatform:
  namespace: "ai-platform"
  name: "splunk-ai-stack"
  serviceAccounts:
    rayHead: "ray-head-sa"
    rayWorker: "ray-worker-sa"
    saiaService: "saia-service-sa"
  defaultAcceleratorType: "L40S"
  workerGroupConfig:
    serviceAccountName: "ray-worker-sa"
    imageRegistry: ""
  ingress:
    enabled: true
    className: "nginx"
    host: "ai.example.com"
    tlsSecretName: "ai-platform-tls"
```

#### 5. Splunk Standalone
```yaml
splunkStandalone:
  name: "splunk-standalone"
  serviceAccount: "saia-service-sa"
  localAppPath: ""  # Path to Splunk AI Assistant app (optional)
```

## Usage

### Install Complete Stack
```bash
cd tools/cluster_setup

# Optionally set custom config file location
export CONFIG_FILE="./cluster-config.yaml"

# Run installation
./eks_cluster_with_stack.sh install
```

The script will:
1. Load configuration from `cluster-config.yaml`
2. Run preflight checks
3. Create or update the EKS cluster
4. Install all required operators and components
5. Deploy the AIPlatform custom resource

### Delete Cluster (Clean)
Deletes the cluster and all associated AWS resources (IAM roles, policies, OIDC providers):
```bash
./eks_cluster_with_stack.sh delete
```

### Delete Everything (Full Cleanup)
Uninstalls all Kubernetes resources first, then performs comprehensive AWS cleanup:
```bash
./eks_cluster_with_stack.sh delete-full
```

## Idempotency

The script is designed to be idempotent - you can run it multiple times safely:
- Existing resources are updated rather than recreated
- Failed installations can be resumed by re-running the script
- Configuration changes in `cluster-config.yaml` will be applied on subsequent runs

## AWS Credentials

The script supports multiple credential sources:

### Environment Variables
```bash
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_SESSION_TOKEN="..."  # Optional
```

### AWS SSO Profile
```bash
export AWS_PROFILE="my-profile"
aws sso login --profile my-profile
./eks_cluster_with_stack.sh install
```

The script will automatically export temporary credentials from your SSO profile.

## Customization

### Service Account Names
All service accounts are configurable via `cluster-config.yaml`. The script creates IRSA (IAM Roles for Service Accounts) for:
- Ray Head (`ray-head-sa`)
- Ray Worker (`ray-worker-sa`)
- SAIA Service (`saia-service-sa`)
- Cluster Autoscaler (`cluster-autoscaler`)

### Operator Versions
Configure operator versions in `cluster-config.yaml`:
```yaml
operators:
  splunk:
    image: "splunk/splunk:10.2.0"
  ray:
    version: "v1.2.2"
  nvidia:
    devicePluginVersion: "v0.17.3"
```

### Node Group Configuration
Enable/disable node groups and adjust capacity:
```yaml
nodeGroups:
  cpu:
    enabled: true  # Set to false to skip CPU nodes
  gpu:
    enabled: true  # Set to false to skip GPU nodes
```

## Preflight Checks

The script performs comprehensive preflight checks before installation:
- Configuration file validation
- Required tools verification
- AWS credentials validation
- VPC subnet verification
- Kubernetes API connectivity (for existing clusters)
- Cluster DNS configuration
- Proxy settings validation

## Troubleshooting

### Preflight Failures
If preflight checks fail:
1. Review the error messages
2. Fix the issues (missing tools, invalid config, etc.)
3. Re-run the script

### Installation Failures
The script uses robust error handling:
- Each component installation is idempotent
- Rollout status checks ensure components are healthy
- Helm operations include automatic retry logic

To retry after a failure:
```bash
./eks_cluster_with_stack.sh install
```

### Cleanup Issues
If resource deletion fails:
1. Check AWS CloudFormation console for stuck stacks
2. Manually delete stuck resources
3. Re-run the delete command

### Log Output
The script provides color-coded logging:
- 🟢 **[INFO]** - Normal operation
- 🟡 **[WARN]** - Warning (operation continues)
- 🔴 **[ERROR]** - Fatal error (script exits)

## Advanced Configuration

### Custom Config File Location
```bash
CONFIG_FILE=/path/to/my-config.yaml ./eks_cluster_with_stack.sh install
```

### Skip Splunk App Upload
Leave `localAppPath` empty in config:
```yaml
splunkStandalone:
  localAppPath: ""
```

### Custom Tolerations and Node Selectors
The AIPlatform CR uses `cpuScheduler` and `gpuScheduler` fields (note: the YAML field names use the JSON tag names from the API, not the Go struct names). These are automatically set by the script with default values for GPU nodes that include the nvidia.com/gpu taint.

## Architecture

The deployment creates:
- **EKS Control Plane** - Managed Kubernetes control plane
- **Node Groups**
  - CPU nodes (m5.xlarge) for control plane workloads
  - GPU nodes (g6e.12xlarge) for AI workloads
- **Storage Layer**
  - S3 bucket for artifacts, apps, tasks
  - EBS volumes (gp3) for persistent storage
  - VectorDB (Weaviate) on PVC
- **Operators**
  - KubeRay for Ray cluster management
  - Splunk Operator for Splunk Enterprise
  - Splunk AI Operator for AI platform orchestration
  - Cert-Manager for TLS certificates
  - OpenTelemetry for observability
- **AI Platform**
  - Ray Head and Worker pods
  - SAIA service
  - Splunk Standalone instance
  - Ingress for external access

## Support

For issues or questions:
1. Check preflight output for configuration errors
2. Review AWS CloudFormation events for infrastructure issues
3. Check Kubernetes events: `kubectl get events -A`
4. Review operator logs: `kubectl logs -n <namespace> <pod-name>`
