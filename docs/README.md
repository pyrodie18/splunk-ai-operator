# Splunk AI Operator Documentation

Welcome to the Splunk AI Operator documentation!

## Getting Started

1. **[Install](installation.md)** - Install the operator in your Kubernetes cluster
2. **[Custom Resources](api-reference.md)** - Configure your AI Platform
3. **[Helm](helm-deployment.md)** - Deploy using Helm charts

## Configuration Guides

### Core Configuration
- **[Custom Resources](api-reference.md)** - Complete AIPlatform spec reference
- **[Service Artifacts Storage](storage-artifacts.md)** - Configure S3/GCS/Azure storage for AI models

### Storage & Access
- **[Storage Configuration](storage-configuration.md)** - Set up persistent storage for Weaviate vector database
- **[Ingress Usage](ingress-configuration.md)** - Expose your AI services externally with custom domains

### Monitoring & Troubleshooting
- **[Error Handling and Events](troubleshooting.md)** - Understand status conditions, events, and troubleshoot issues

## Architecture
- **[Reference Architecture](deployment-aws-eks.md)** - Understand how the system works

## Quick Reference

### Check if Platform is Ready
```bash
kubectl get aiplatform <name> -n <namespace>
```

### View Status Details
```bash
kubectl get aiplatform <name> -n <namespace> -o jsonpath='{.status.conditions}'
```

### Watch Events
```bash
kubectl get events -n <namespace> --watch --field-selector involvedObject.name=<name>
```

### Common Tasks

**Configure persistent storage:**
```yaml
spec:
  storage:
    vectorDB:
      size: "100Gi"
      storageClassName: "gp3"
```

**Enable external access:**
```yaml
spec:
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: ai.example.com
        paths:
          - path: /
            pathType: Prefix
```

**Check what's failing:**
```bash
kubectl get aiplatform <name> -o jsonpath='{.status.conditions}' | jq '.[] | select(.status=="False")'
```

## Need Help?

1. Check [Error Handling and Events](troubleshooting.md) for troubleshooting guides
2. View operator logs: `kubectl logs -n splunk-ai-operator-system deployment/splunk-ai-operator-controller-manager`
3. Report issues with diagnostic info (see troubleshooting guide)

## Documentation Organization

- **Getting Started** - Installation and basic setup
- **Configuration Guides** - Detailed configuration for specific features
- **Monitoring** - Understanding status and troubleshooting
- **Architecture** - System design and components
