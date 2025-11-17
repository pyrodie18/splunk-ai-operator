# Splunk AI Operator

<!-- Build & Test Status -->
[![Build and Test](https://github.com/splunk/splunk-ai-operator/actions/workflows/main.yml/badge.svg)](https://github.com/splunk/splunk-ai-operator/actions/workflows/main.yml)
[![Helm Lint and Test](https://github.com/splunk/splunk-ai-operator/actions/workflows/helm-lint-test.yml/badge.svg)](https://github.com/splunk/splunk-ai-operator/actions/workflows/helm-lint-test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/splunk/splunk-ai-operator)](https://goreportcard.com/report/github.com/splunk/splunk-ai-operator)
[![Coverage Status](https://coveralls.io/repos/github/splunk/splunk-ai-operator/badge.svg?branch=main)](https://coveralls.io/github/splunk/splunk-ai-operator?branch=main)

<!-- Release & Version -->
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/splunk/splunk-ai-operator)](https://github.com/splunk/splunk-ai-operator/releases)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/splunk/splunk-ai-operator?sort=semver)](https://github.com/splunk/splunk-ai-operator/tags)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/splunk-ai-operator)](https://artifacthub.io/packages/helm/splunk-ai-operator/splunk-ai-operator)

<!-- Container Registry -->
[![Container Image](https://img.shields.io/badge/container-ghcr.io-blue)](https://github.com/splunk/splunk-ai-operator/pkgs/container/splunk-ai-operator)
[![Docker Pulls](https://img.shields.io/docker/pulls/splunk/splunk-ai-operator)](https://hub.docker.com/r/splunk/splunk-ai-operator)

<!-- License & Legal -->
[![License](https://img.shields.io/github/license/splunk/splunk-ai-operator)](LICENSE)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fsplunk%2Fsplunk-ai-operator.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fsplunk%2Fsplunk-ai-operator?ref=badge_shield)

<!-- Community & Support -->
[![GitHub issues](https://img.shields.io/github/issues/splunk/splunk-ai-operator)](https://github.com/splunk/splunk-ai-operator/issues)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/splunk/splunk-ai-operator)](https://github.com/splunk/splunk-ai-operator/pulls)
[![GitHub stars](https://img.shields.io/github/stars/splunk/splunk-ai-operator?style=social)](https://github.com/splunk/splunk-ai-operator/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/splunk/splunk-ai-operator?style=social)](https://github.com/splunk/splunk-ai-operator/network/members)
[![GitHub contributors](https://img.shields.io/github/contributors/splunk/splunk-ai-operator)](https://github.com/splunk/splunk-ai-operator/graphs/contributors)

<!-- Code Quality & Security -->
[![CodeQL](https://github.com/splunk/splunk-ai-operator/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/splunk/splunk-ai-operator/actions/workflows/codeql-analysis.yml)
[![OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/projects/XXXXX/badge)](https://bestpractices.coreinfrastructure.org/projects/XXXXX)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/splunk/splunk-ai-operator/badge)](https://api.securityscorecards.dev/projects/github.com/splunk/splunk-ai-operator)

<!-- Documentation -->
[![Documentation](https://img.shields.io/badge/docs-latest-blue)](https://github.com/splunk/splunk-ai-operator/tree/main/docs)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/splunk/splunk-ai-operator)

<!-- Languages & Versions -->
[![Go Version](https://img.shields.io/github/go-mod/go-version/splunk/splunk-ai-operator)](go.mod)
[![Kubernetes Version](https://img.shields.io/badge/kubernetes-v1.31+-blue.svg)](https://kubernetes.io/)

---
The Splunk AI Operator is a Kubernetes operator that enables customers to manage AI workloads using standardized CRDs, Helm charts, and Kubernetes primitives without reliance on any specific cloud provider’s tooling or rigid infrastructure. This repo includes the Splunk AI Operator, and multiple CRDs to manage the Splunk AI Platform and Splunk AI Services.

## Getting Started

### Quick Install with Helm (Recommended)

```bash
# Install the operator from GitHub Release
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  -n splunk-ai-operator-system --create-namespace

# Deploy the AI Platform
kubectl apply -f config/samples/ai_v1_aiplatform.yaml
```

Images are hosted on GitHub Container Registry (ghcr.io) and Docker Hub.

See [Helm Deployment Guide](docs/deployment/helm-deployment.md) for detailed installation options.

### Prerequisites
- Kubernetes v1.11.3+ cluster
- kubectl v1.11.3+
- Helm v3.8+ (for Helm installation)
- go v1.23.0+ (for development)
- docker 17.03+ (for development)

### Installation Options

**Option 1: Helm (Recommended for Production)**
```bash
# Install from GitHub Release
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-0.1.0.tgz \
  -n splunk-ai-operator-system --create-namespace

# Or add Helm repository
helm repo add splunk-ai https://splunk.github.io/splunk-ai-operator/
helm repo update
helm install splunk-ai-operator splunk-ai/splunk-ai-operator \
  -n splunk-ai-operator-system --create-namespace
```

**Option 2: YAML Manifests**
```bash
kubectl apply -f https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/splunk-ai-operator-cluster.yaml
```

**Option 3: From Source (Development)**
```bash
# Install CRDs
make install

# Build and deploy (uses ghcr.io by default)
make docker-build docker-push IMG=ghcr.io/splunk/splunk-ai-operator:tag
make deploy IMG=ghcr.io/splunk/splunk-ai-operator:tag
```

### Container Images

The operator is published to multiple registries:

- **GitHub Container Registry (GHCR)**: `ghcr.io/splunk/splunk-ai-operator:latest` (recommended)
- **Docker Hub**: `docker.io/splunk/splunk-ai-operator:latest`

```bash
# Pull from GHCR
docker pull ghcr.io/splunk/splunk-ai-operator:v0.1.0

# Pull from Docker Hub
docker pull docker.io/splunk/splunk-ai-operator:v0.1.0
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
- **[Helm Deployment](docs/deployment/helm-deployment.md)** - Helm chart installation
- **[API Reference](docs/api-reference.md)** - Complete CRD specification
- **[AWS EKS Deployment](docs/deployment/deployment-aws-eks.md)** - Production deployment on AWS
- **[Configuration Guides](docs/configuration/)** - Storage, ingress, and webhook configuration
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues and solutions

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

