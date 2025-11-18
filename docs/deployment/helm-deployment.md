# Splunk AI Platform Helm Installation

Helm charts for the Splunk AI Operator are distributed via **OCI Registry (GHCR)** and **GitHub Releases**. The OCI registry is the recommended method for Helm 3.8+.

## Installation Methods

### Method 1: Helm OCI Registry (Recommended)

Install directly from the OCI registry using Helm 3.8+:

```bash
# Latest version: v0.1.0
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 0.1.0 \
  --namespace splunk-ai-operator \
  --create-namespace
```

**Pros:**
- ✅ Modern OCI-based distribution
- ✅ No .tgz files in git repository
- ✅ Automatic image verification
- ✅ Native container registry integration

**Requirements:**
- Helm 3.8+ (for OCI support)
- Internet access to ghcr.io

### Method 2: Direct Install from GitHub Release

Install directly from a specific release URL (for Helm < 3.8 compatibility):

```bash
# Latest version: v0.1.0
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  --namespace splunk-ai-operator \
  --create-namespace
```

**Pros:**
- ✅ Compatible with older Helm versions (3.0+)
- ✅ Simple one-command installation
- ✅ Explicit version control

**Available Charts:**
* `splunk-ai-operator`: Deploys the Splunk AI Operator (controller for CRDs like `AIPlatform`)
* `splunk-ai-platform`: Deploys the full AI platform stack via an `AIPlatform` custom resource

---

## Finding Available Versions

### OCI Registry (Recommended)

View available versions in the GitHub Container Registry:

**GHCR Package:** https://github.com/splunk/splunk-ai-operator/pkgs/container/charts%2Fsplunk-ai-operator

```bash
# Show chart information
helm show chart oci://ghcr.io/splunk/charts/splunk-ai-operator --version 0.1.0

# List all versions (using crane CLI)
crane ls ghcr.io/splunk/charts/splunk-ai-operator
```

### GitHub Releases

**Latest Releases:** https://github.com/splunk/splunk-ai-operator/releases

Or use the GitHub API:

```bash
curl -s https://api.github.com/repos/splunk/splunk-ai-operator/releases | jq -r '.[].tag_name'
```

---

## CRD Management

> **Note:** Helm does not manage CRD upgrades. To install or upgrade CRDs manually:

```bash
# Install CRDs from a specific version
kubectl apply -f https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/aiplatform-crd.yaml

# Or clone and install
git clone https://github.com/splunk/splunk-ai-operator.git
cd splunk-ai-operator
git checkout v0.1.0
make install
```

---

## Install the Splunk AI Operator

To install the controller that manages `AIPlatform` resources:

```bash
# OCI Registry (recommended - Helm 3.8+)
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 0.1.0 \
  --namespace splunk-ai-operator \
  --create-namespace

# Or from GitHub Release (for Helm < 3.8)
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  --namespace splunk-ai-operator \
  --create-namespace
```

**View available configuration options:**

```bash
# From OCI registry
helm show values oci://ghcr.io/splunk/charts/splunk-ai-operator --version 0.1.0

# Or download and inspect from GitHub Release
curl -sL https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz | tar -xzO splunk-ai-operator/values.yaml
```

---

## Container Images Configuration

### Overview

All container images used by the Splunk AI Platform can be configured via Helm values. This allows you to:

- ✅ Use private container registries (ECR, GCR, ACR, Harbor)
- ✅ Mix public (Docker Hub) and private images
- ✅ Pin specific image versions for reproducibility
- ✅ Use custom-built images for development/testing

### Configurable Images

The following images can be customized in the Helm chart:

| Image | Values Key | Default | Purpose |
|-------|-----------|---------|---------|
| **Operator** | `image.repository` | `docker.io/splunk/splunk-ai-operator:0.1.0` | Main operator controller |
| **Splunk Enterprise** | `splunkEnterpriseImage` | `docker.io/splunk/splunk:9.4.1` | Splunk instance for observability |
| **Ray Head** | `rayHeadImage` | `YOUR_REGISTRY/...` | Ray cluster head node |
| **Ray Worker** | `rayWorkerImage` | `YOUR_REGISTRY/...` | Ray worker nodes (GPU) |
| **Weaviate** | `weaviateImage` | `docker.io/semitechnologies/weaviate:...` | Vector database |
| **SAIA API** | `saiaApiImage` | `YOUR_REGISTRY/...` | AI Assistant API service |
| **SAIA Schema** | `saiaSchemaImage` | `YOUR_REGISTRY/...` | AI Assistant data loader |

### Example: Using Private ECR Registry

Create a `custom-images.yaml` file:

```yaml
# Use your AWS ECR registry
image:
  repository: "123456789012.dkr.ecr.us-west-2.amazonaws.com/splunk-ai-operator:0.1.0"

# Ray images from ECR
rayHeadImage: "123456789012.dkr.ecr.us-west-2.amazonaws.com/ml-platform/ray/ray-head:v1.0"
rayWorkerImage: "123456789012.dkr.ecr.us-west-2.amazonaws.com/ml-platform/ray/ray-worker-gpu:v1.0"

# SAIA images from ECR
saiaApiImage: "123456789012.dkr.ecr.us-west-2.amazonaws.com/ml-platform/saia/saia-api:v1.0"
saiaSchemaImage: "123456789012.dkr.ecr.us-west-2.amazonaws.com/ml-platform/saia/ai-helm-post-hook:v1.0"

# Keep Splunk and Weaviate from Docker Hub
splunkEnterpriseImage: "docker.io/splunk/splunk:9.4.1"
weaviateImage: "docker.io/semitechnologies/weaviate:stable-v1.28-007846a"
```

**Install with custom images:**

```bash
# OCI Registry (recommended)
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 0.1.0 \
  --namespace splunk-ai-operator \
  --create-namespace \
  --values custom-images.yaml

# Or from GitHub Release
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  --namespace splunk-ai-operator \
  --create-namespace \
  --values custom-images.yaml
```

### Example: Using Docker Hub Only

```yaml
# All images from Docker Hub
image:
  repository: "docker.io/myorg/splunk-ai-operator:0.1.0"

rayHeadImage: "docker.io/myorg/ray-head:v1.0"
rayWorkerImage: "docker.io/myorg/ray-worker-gpu:v1.0"
weaviateImage: "docker.io/semitechnologies/weaviate:stable-v1.28-007846a"
saiaApiImage: "docker.io/myorg/saia-api:v1.0"
saiaSchemaImage: "docker.io/myorg/ai-helm-post-hook:v1.0"
splunkEnterpriseImage: "docker.io/splunk/splunk:9.4.1"
```

### Image Pull Secrets

If using private registries, configure image pull secrets:

```yaml
imagePullSecrets:
  - name: ecr-registry-secret
```

**Create the secret first:**

```bash
# For AWS ECR
kubectl create secret docker-registry ecr-registry-secret \
  --docker-server=123456789012.dkr.ecr.us-west-2.amazonaws.com \
  --docker-username=AWS \
  --docker-password=$(aws ecr get-login-password --region us-west-2) \
  -n splunk-ai-operator

# For Docker Hub
kubectl create secret docker-registry dockerhub-secret \
  --docker-server=docker.io \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  -n splunk-ai-operator
```

### Verifying Images Before Installation

Before installing, verify all images are accessible:

```bash
# Test pulling an image manually
docker pull 123456789012.dkr.ecr.us-west-2.amazonaws.com/ml-platform/ray/ray-head:v1.0

# Or use crane (faster, no Docker daemon needed)
crane manifest 123456789012.dkr.ecr.us-west-2.amazonaws.com/ml-platform/ray/ray-head:v1.0

# For ECR, ensure you're logged in
aws ecr get-login-password --region us-west-2 | \
  docker login --username AWS --password-stdin \
  123456789012.dkr.ecr.us-west-2.amazonaws.com
```

### Complete Custom Values Example

```yaml
# Helm values file: my-values.yaml

# Operator image
image:
  repository: "123456789012.dkr.ecr.us-west-2.amazonaws.com/splunk-ai-operator:0.1.0"
  pullPolicy: IfNotPresent

# Image pull secrets for private registry
imagePullSecrets:
  - name: ecr-registry-secret

# Container images
splunkEnterpriseImage: "docker.io/splunk/splunk:10.2.0"
rayHeadImage: "123456789012.dkr.ecr.us-west-2.amazonaws.com/ray/ray-head:v2.44.0"
rayWorkerImage: "123456789012.dkr.ecr.us-west-2.amazonaws.com/ray/ray-worker-gpu:v2.44.0"
weaviateImage: "docker.io/semitechnologies/weaviate:stable-v1.28-007846a"
saiaApiImage: "123456789012.dkr.ecr.us-west-2.amazonaws.com/saia/api:v1.1.0"
saiaSchemaImage: "123456789012.dkr.ecr.us-west-2.amazonaws.com/saia/schema:v1.1.0"

# Resource limits
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 10m
    memory: 64Mi
```

**Install:**

```bash
# OCI Registry (recommended)
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 0.1.0 \
  --namespace splunk-ai-operator \
  --create-namespace \
  --values my-values.yaml

# Or from GitHub Release
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  --namespace splunk-ai-operator \
  --create-namespace \
  --values my-values.yaml
```

---

## Deploy the Splunk AI Platform

To deploy the full AI Platform stack using the `splunk-ai-platform` chart, you only need to define a few core fields in your `values.yaml` file.

### ✨ Example: `ai-platform-values.yaml`

```yaml
name: my-ai-platform
namespace: ai-stack

serviceAccountName: "ai-platform-sa"

volume:
  path: "s3://my-bucket/prefix"
  region: "us-west-2"
  secretRef: "s3-secret"

splunkConfiguration:
  crName: "splunk-observability"
  crNamespace: "splunk"
  secretRef:
    name: "splunk-token-secret"
    namespace: "splunk"
```

> All other settings like Ray/Weaviate images, sidecars, GPU/CPU scheduling, and storage can be customized as needed via the chart’s default `values.yaml`.

---

## Install with the Simplified Config

```bash
# OCI Registry (recommended - Helm 3.8+)
helm install splunk-ai-platform \
  oci://ghcr.io/splunk/charts/splunk-ai-platform \
  --version 0.1.0 \
  --namespace ai-stack \
  --create-namespace \
  --values ai-platform-values.yaml

# Or from GitHub Release (for Helm < 3.8)
helm install splunk-ai-platform \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-platform-0.1.0.tgz \
  --namespace ai-stack \
  --create-namespace \
  --values ai-platform-values.yaml
```

**Upgrade:**

```bash
# OCI Registry
helm upgrade splunk-ai-platform \
  oci://ghcr.io/splunk/charts/splunk-ai-platform \
  --version 0.1.0 \
  --namespace ai-stack \
  --values ai-platform-values.yaml

# Or from GitHub Release
helm upgrade splunk-ai-platform \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-platform-0.1.0.tgz \
  --namespace ai-stack \
  --values ai-platform-values.yaml
```

**Uninstall:**

```bash
helm uninstall splunk-ai-platform -n ai-stack
```

**View configurable values:**

```bash
# From OCI registry
helm show values oci://ghcr.io/splunk/charts/splunk-ai-platform --version 0.1.0

# Or download and inspect from GitHub Release
curl -sL https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-platform-0.1.0.tgz | tar -xzO splunk-ai-platform/values.yaml
```

---

## View Running Resources

Once installed, confirm the AI platform resources are running:

```bash
kubectl get aiplatform -n ai-stack
kubectl get pods -n ai-stack
```

---

## Building and Packaging Helm Charts

For developers and maintainers who need to build Helm charts from source:

### Prerequisites

- `helm` CLI installed (v3.8+)
- `make` available in PATH
- Git repository cloned

### Available Make Targets

The Makefile provides several targets for Helm chart operations:

```bash
# View all available Helm targets
make help | grep helm

# Common targets:
make helm-lint        # Lint both charts
make helm-package     # Package charts into .tgz files
make helm-index       # Generate repository index.yaml
make helm-all         # Lint, package, and index (full build)
make helm-template    # Render templates locally (for testing)
make helm-clean       # Clean build artifacts
```

### Building Helm Charts

**1. Lint charts to check for issues:**

```bash
make helm-lint
```

**Output:**
```
Linting Helm charts...
==> Linting helm-chart/splunk-ai-operator
[INFO] Chart.yaml: icon is recommended
1 chart(s) linted, 0 chart(s) failed

==> Linting helm-chart/splunk-ai-platform
[INFO] Chart.yaml: icon is recommended
1 chart(s) linted, 0 chart(s) failed

✓ Helm charts linting complete
```

**2. Package charts into tgz archives:**

```bash
make helm-package
```

**Output:**
```
Packaging Helm charts...
Successfully packaged chart and saved it to: dist/helm/splunk-ai-operator-0.1.0.tgz
Successfully packaged chart and saved it to: dist/helm/splunk-ai-platform-0.1.0.tgz
✓ Helm charts packaged:
-rw-r--r-- 1 user staff 12K Nov 14 10:00 dist/helm/splunk-ai-operator-0.1.0.tgz
-rw-r--r-- 1 user staff 8.5K Nov 14 10:00 dist/helm/splunk-ai-platform-0.1.0.tgz
```

**3. Generate Helm repository index:**

```bash
make helm-index
```

This creates `dist/helm/index.yaml` with metadata for both charts.

**4. Complete build (lint + package + index):**

```bash
make helm-all
```

### Customizing Chart Version

Set custom version when building:

```bash
# Build charts with specific version
make helm-package VERSION=0.2.0 HELM_CHART_VERSION=0.2.0

# Or set environment variable
export VERSION=0.2.0
export HELM_CHART_VERSION=0.2.0
make helm-all
```

### Testing Charts Locally

**Render templates without installing:**

```bash
make helm-template

# Or manually:
helm template test-operator helm-chart/splunk-ai-operator --debug
helm template test-platform helm-chart/splunk-ai-platform --debug
```

**Install from local chart directory:**

```bash
# Install operator from source
make helm-install-operator

# Or manually:
helm install splunk-ai-operator ./helm-chart/splunk-ai-operator \
  -n splunk-ai-operator --create-namespace \
  -f my-custom-values.yaml
```

**Uninstall:**

```bash
make helm-uninstall

# Or manually:
helm uninstall splunk-ai-operator -n splunk-ai-operator
```

### Publishing Charts to GitHub Releases

**1. Build and package charts:**

```bash
make helm-all VERSION=0.1.0
```

**2. Upload artifacts to GitHub release:**

Upload these files from `dist/helm/` to your GitHub release:
- `splunk-ai-operator-0.1.0.tgz`
- `splunk-ai-platform-0.1.0.tgz`
- `index.yaml` (optional, for Helm repository)

**3. Users can install directly from release URL:**

```bash
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  -n splunk-ai-operator --create-namespace
```

### Chart Directory Structure

```
helm-chart/
├── splunk-ai-operator/
│   ├── Chart.yaml           # Chart metadata
│   ├── values.yaml          # Default values
│   ├── templates/           # Kubernetes manifests
│   │   ├── deployment.yaml
│   │   ├── serviceaccount.yaml
│   │   └── ...
│   └── crds/                # Custom Resource Definitions
│       └── aiplatform_crd.yaml
└── splunk-ai-platform/
    ├── Chart.yaml
    ├── values.yaml
    └── templates/
        └── aiplatform.yaml
```

### Generating Chart Documentation

If you have `helm-docs` installed:

```bash
make helm-docs

# This generates/updates README.md files in each chart directory
```

Install helm-docs: https://github.com/norwoodj/helm-docs

---

## Learn More

* [Helm Documentation](https://helm.sh/docs/)
* [Splunk AI Operator GitHub](https://github.com/splunk/splunk-ai-operator)
* [Helm Chart Best Practices](https://helm.sh/docs/chart_best_practices/)
