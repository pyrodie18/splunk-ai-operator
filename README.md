[![Go Report Card](https://goreportcard.com/badge/github.com/splunk/splunk-ai-operator)](https://goreportcard.com/report/github.com/splunk/splunk-ai-operator)
[![Coverage Status](https://coveralls.io/repos/github/splunk/splunk-ai-operator/badge.svg?branch=main)](https://coveralls.io/github/splunk/splunk-ai-operator?branch=main)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fsplunk%2Fsplunk-ai-operator.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fsplunk%2Fsplunk-ai-operator?ref=badge_shield)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/splunk/splunk-ai-operator)

# splunk-ai-operator
The Splunk AI Operator is a Kubernetes operator that enables customers to manage AI workloads using standardized CRDs, Helm charts, and Kubernetes primitives without reliance on any specific cloud provider’s tooling or rigid infrastructure. This repo includes the Splunk AI Operator, and multiple CRDs to manage the Splunk AI Platform and Splunk AI Services.

## Getting Started

### Quick Install with Helm (Recommended)

```bash
# Install the operator from GitHub Release
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  -n splunk-ai-operator --create-namespace

# Deploy the AI Platform
kubectl apply -f config/samples/ai_v1_aiplatform.yaml
```

See [Helm Deployment Guide](docs/helm-deployment.md) for detailed installation options.

### Prerequisites
- Kubernetes v1.11.3+ cluster
- kubectl v1.11.3+
- Helm v3.8+ (for Helm installation)
- go v1.23.0+ (for development)
- docker 17.03+ (for development)

### Installation Options

**Option 1: Helm (Recommended for Production)**
```bash
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  -n splunk-ai-operator --create-namespace
```

**Option 2: YAML Manifests**
```bash
kubectl apply -f https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-cluster.yaml
```

**Option 3: From Source (Development)**
```bash
# Install CRDs
make install

# Build and deploy
make docker-build docker-push IMG=<registry>/splunk-ai-operator:tag
make deploy IMG=<registry>/splunk-ai-operator:tag
```

### Deploy AI Platform

```bash
# Create sample AI Platform
kubectl apply -k config/samples/
```

### Uninstall

**Helm:**
```bash
helm uninstall splunk-ai-operator -n splunk-ai-operator
```

**From Source:**
```bash
kubectl delete -k config/samples/
make undeploy
make uninstall
```

### Documentation

- **[Installation Guide](docs/installation.md)** - Detailed installation instructions
- **[Helm Deployment](docs/helm-deployment.md)** - Helm chart installation
- **[API Reference](docs/api-reference.md)** - Complete CRD specification
- **[AWS EKS Deployment](docs/deployment-aws-eks.md)** - Production deployment on AWS

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

