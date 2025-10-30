# k0s Cluster Setup for Splunk AI Platform

This script (`k0s_cluster_with_stack.sh`) deploys the Splunk AI Platform on a k0s Kubernetes cluster, supporting both on-premises/baremetal deployments and AWS EC2 testing environments.

## Overview

The script mirrors the functionality of `eks_cluster_with_stack.sh` but uses k0s instead of EKS, making it suitable for:
- **On-premises data centers** with existing hardware
- **Bare metal servers** with customer-managed infrastructure
- **AWS EC2 instances** for testing and simulation
- **Air-gapped environments** (with MinIO instead of S3)

## Features

### Deployed Components

The script installs the complete AI Platform stack:

1. **k0s Kubernetes Cluster** - Lightweight, certified Kubernetes distribution
2. **MinIO** - S3-compatible object storage (replaces AWS S3)
3. **Cert-Manager** - Certificate management for TLS
4. **Kube-Prometheus Stack** - Monitoring and metrics
5. **OpenTelemetry Operator** - Distributed tracing and telemetry
6. **NVIDIA GPU Operator** - GPU support for AI workloads (if GPU workers present)
7. **KubeRay Operator** - Ray cluster management for distributed AI
8. **Splunk Operator** - Splunk Enterprise management
9. **Splunk AI Platform Operator** - Main AI platform orchestration
10. **AI Platform CR** - Complete AI platform deployment with all services

### Two Deployment Modes

#### Mode 1: On-Premises/Baremetal
- Customer provides list of IP addresses
- Requires passwordless SSH with sudo access
- Suitable for production on-prem deployments

#### Mode 2: AWS EC2 (Testing/Simulation)
- Automatically creates EC2 instances
- Simulates on-prem environment
- Useful for testing before on-prem deployment

## Prerequisites

### For Both Modes
- `kubectl` - Kubernetes CLI
- `helm` - Kubernetes package manager
- `git` - For cloning repositories
- `jq` - JSON processing
- `ssh` - SSH client
- `yq` - YAML processing (optional, fallback parsing available)

### For On-Prem Mode
- Ubuntu 22.04 or similar Linux distribution on all nodes
- Passwordless SSH access to all nodes
- Sudo privileges on all nodes
- Open ports between nodes:
  - 6443 (Kubernetes API)
  - 2380 (etcd)
  - 10250 (Kubelet)
  - 30000-32767 (NodePort services)

### For EC2 Mode
- `aws` CLI configured with appropriate credentials
- AWS account with EC2 permissions
- Existing VPC and SSH key pair
- Sufficient EC2 quotas for instance types

## Configuration

### Step 1: Copy and Edit Configuration File

```bash
cd tools/cluster_setup
cp k0s-cluster-config.yaml my-cluster-config.yaml
vi my-cluster-config.yaml
```

### Step 2: Configure for Your Environment

#### On-Prem Configuration Example:

```yaml
cluster:
  name: prod-ai-cluster
  sshUser: ubuntu
  sshKeyPath: ~/.ssh/prod-key

nodes:
  controllers: 1
  cpuWorkers: 0  # Not used when providing IPs
  gpuWorkers: 0  # Not used when providing IPs

  existingIPs:
    controllers:
      - 192.168.1.10  # Your controller node
    workers:
      - 192.168.1.20  # CPU worker 1
      - 192.168.1.21  # CPU worker 2
      - 192.168.1.22  # GPU worker

minio:
  accessKey: admin
  secretKey: SuperSecretPassword123
  bucket: ai-platform-prod

kubernetes:
  namespace: ai-platform
```

#### EC2 Configuration Example:

```yaml
cluster:
  name: test-ai-cluster
  region: us-west-2
  sshUser: ubuntu
  sshKeyPath: ~/.ssh/my-key.pem

nodes:
  controllers: 1
  cpuWorkers: 2
  gpuWorkers: 1

  existingIPs:
    controllers: []  # Empty = create instances
    workers: []

ec2:
  vpcId: vpc-0123456789abcdef0
  subnetId: subnet-0123456789abcdef0  # Optional
  keyName: my-ec2-key

instanceTypes:
  controller: t3.xlarge
  cpuWorker: m5.4xlarge
  gpuWorker: g5.2xlarge

minio:
  accessKey: minioadmin
  secretKey: minioadmin123
  bucket: ai-platform-test
```

## Usage

### Install

```bash
# On-prem deployment
CONFIG_FILE=./my-on-prem-config.yaml ./k0s_cluster_with_stack.sh install

# EC2 testing deployment
CONFIG_FILE=./my-ec2-config.yaml ./k0s_cluster_with_stack.sh install
```

### Delete

```bash
# Delete cluster and resources
CONFIG_FILE=./my-config.yaml ./k0s_cluster_with_stack.sh delete
```

## Post-Installation

### Access the Cluster

```bash
# Set kubeconfig
export KUBECONFIG=~/.kube/k0s-<cluster-name>

# Verify cluster
kubectl get nodes
kubectl get pods --all-namespaces
```

### Access MinIO Console

```bash
# Port forward MinIO console
kubectl port-forward svc/minio -n minio-system 9001:9001

# Open browser
# http://localhost:9001
# Login with credentials from config file
```

### Check AI Platform

```bash
# Check AI Platform status
kubectl get aiplatform -n ai-platform

# Check all AI components
kubectl get all -n ai-platform

# Check Splunk
kubectl get standalone -n ai-platform
```

### Access Splunk

```bash
# Get Splunk password
kubectl get secret splunk-standalone-secret -n ai-platform -o jsonpath='{.data.password}' | base64 -d

# Port forward Splunk
kubectl port-forward svc/splunk-standalone-standalone-service -n ai-platform 8000:8000

# Access at http://localhost:8000
```

## Architecture

### Network Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     k0s Controller Node(s)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ API Server   │  │    etcd      │  │  Scheduler   │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼────────┐  ┌───────▼────────┐  ┌──────▼─────────┐
│  CPU Worker 1  │  │  CPU Worker 2  │  │  GPU Worker    │
│                │  │                │  │                │
│ Ray Head       │  │ Ray Workers    │  │ Ray GPU Workers│
│ Weaviate       │  │ Ray Workers    │  │                │
│ MinIO          │  │                │  │                │
└────────────────┘  └────────────────┘  └────────────────┘
```

### Storage Architecture

```
┌──────────────────────────────────────────────────────┐
│                    MinIO                             │
│  (S3-Compatible Object Storage)                      │
│                                                      │
│  Buckets:                                            │
│  ├─ ai-platform-data/                               │
│  │  ├─ models/      (ML models)                     │
│  │  ├─ datasets/    (Training data)                 │
│  │  ├─ artifacts/   (Build artifacts)               │
│  │  └─ tasks/       (Task outputs)                  │
│  │                                                   │
│  └─ splunk-index/   (Splunk SmartStore)            │
└──────────────────────────────────────────────────────┘
```

## Node Labels and Scheduling

The script automatically labels all nodes for proper workload scheduling:

### Node Labels

#### Controller Nodes:
```yaml
splunk.ai/node-role: controller
splunk.ai/workload-type: control-plane
node.kubernetes.io/role: controller
```

#### CPU Worker Nodes:
```yaml
splunk.ai/node-role: worker
splunk.ai/workload-type: cpu
node.kubernetes.io/workload: ai-cpu
splunk.ai/instance-type: cpu-worker
```

#### GPU Worker Nodes:
```yaml
splunk.ai/node-role: worker
splunk.ai/workload-type: gpu
node.kubernetes.io/workload: ai-gpu
splunk.ai/instance-type: gpu-worker
nvidia.com/gpu: "true"
```

### Taints

GPU nodes are automatically tainted to prevent non-GPU workloads:
```yaml
nvidia.com/gpu=true:NoSchedule
```

### Using Labels in AIPlatform

The AIPlatform CR automatically uses these labels for scheduling:

```yaml
apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: my-platform
spec:
  # CPU workloads scheduled on CPU workers
  cpuSchedulingSpec:
    nodeSelector:
      splunk.ai/workload-type: cpu
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
          - matchExpressions:
            - key: splunk.ai/workload-type
              operator: In
              values:
              - cpu

  # GPU workloads scheduled on GPU workers with tolerations
  gpuSchedulingSpec:
    enabled: true
    nodeSelector:
      splunk.ai/workload-type: gpu
      nvidia.com/gpu: "true"
    tolerations:
    - key: nvidia.com/gpu
      operator: Equal
      value: "true"
      effect: NoSchedule
```

### Viewing Node Labels

```bash
# View all labels
kubectl get nodes --show-labels

# View specific labels
kubectl get nodes -L splunk.ai/workload-type,splunk.ai/node-role

# View GPU nodes
kubectl get nodes -l splunk.ai/workload-type=gpu

# View CPU nodes
kubectl get nodes -l splunk.ai/workload-type=cpu
```

## Differences from EKS Deployment

| Feature | EKS | k0s |
|---------|-----|-----|
| Kubernetes Distribution | AWS EKS | k0s (CNCF certified) |
| Object Storage | AWS S3 | MinIO (on-cluster) |
| Authentication | IAM/IRSA | MinIO credentials |
| Node Provisioning | AWS Auto Scaling | Manual/EC2 script |
| Node Labels | Auto (node groups) | Script-managed |
| Load Balancer | AWS ELB | NodePort/MetalLB |
| Storage | EBS CSI | Local/Longhorn |
| GPU Support | EKS managed | NVIDIA Operator |

## Troubleshooting

### k0s Installation Issues

```bash
# Check k0s status on controller
ssh user@controller-ip
sudo k0s status

# Check k0s logs
sudo journalctl -u k0scontroller -f

# Reset k0s if needed
sudo k0s reset
```

### Worker Join Issues

```bash
# Regenerate worker token
ssh user@controller-ip
sudo k0s token create --role=worker

# Manually install on worker
ssh user@worker-ip
sudo k0s install worker --token-file=<(echo 'TOKEN_HERE')
sudo k0s start
```

### MinIO Connection Issues

```bash
# Check MinIO pods
kubectl get pods -n minio-system

# Check MinIO logs
kubectl logs -n minio-system deployment/minio

# Test MinIO connectivity
kubectl run -it --rm test-minio --image=minio/mc --restart=Never -- sh
mc alias set test http://minio.minio-system.svc.cluster.local:9000 <access-key> <secret-key>
mc ls test
```

### GPU Not Detected

```bash
# Check GPU operator
kubectl get pods -n gpu-operator

# Check node labels
kubectl get nodes -o json | jq '.items[].status.capacity'

# Manually label GPU nodes if needed
kubectl label nodes <gpu-node> nvidia.com/gpu=true
```

## Security Considerations

### For Production On-Prem Deployments

1. **Change MinIO Credentials**: Use strong passwords
2. **Enable TLS**: Configure cert-manager for HTTPS
3. **Network Policies**: Restrict pod-to-pod communication
4. **SSH Keys**: Use unique SSH keys per environment
5. **Firewall Rules**: Lock down node access
6. **Backup**: Regular backups of MinIO data
7. **Monitoring**: Enable Prometheus alerts
8. **Audit Logging**: Enable Kubernetes audit logs

### Example Security Hardening

```bash
# Change MinIO credentials after installation
kubectl create secret generic minio-creds \
  --namespace=minio-system \
  --from-literal=accesskey='<strong-access-key>' \
  --from-literal=secretkey='<strong-secret-key>' \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart MinIO
kubectl rollout restart deployment/minio -n minio-system
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/splunk/splunk-ai-operator/issues
- Documentation: https://docs.splunk.com (when available)

## License

See the main repository LICENSE file.
