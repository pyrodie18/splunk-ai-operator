# Troubleshooting with Events and Status

This guide helps you understand what's happening with your AI Platform deployments using Kubernetes events and status conditions.

## Quick Start

### Is My Platform Ready?

```bash
# Check overall status
kubectl get aiplatform <name> -n <namespace>

# Get detailed readiness
kubectl get aiplatform <name> -n <namespace> -o jsonpath='{.status.conditions[?(@.type=="Ready")]}'
```

If `status: "True"` - your platform is ready!
If `status: "False"` - check the message field for what's wrong.

### What's Happening Right Now?

```bash
# Watch events in real-time
kubectl get events -n <namespace> --watch --field-selector involvedObject.name=<name>

# See recent events
kubectl describe aiplatform <name> -n <namespace> | tail -30
```

## Understanding Status Conditions

Your AI Platform tracks several health indicators:

### Platform Components

| Condition | What It Means | When It's False |
|-----------|---------------|-----------------|
| `Ready` | Everything is working | One or more components have issues |
| `RayServiceReady` | Ray cluster is operational | Ray is starting, upgrading, or failed |
| `RayClusterReady` | Ray pods are running | Pods are pending, failing, or not enough replicas |
| `RayServeRouteReady` | AI inference API is available | Applications failed to deploy or endpoints not ready |
| `WeaviateDatabaseReady` | Vector database is running | Weaviate pods are not ready |
| `IngressReady` | External access is configured | Ingress hasn't received an address yet |

### Check Specific Components

```bash
# Check if Ray is ready
kubectl get aiplatform <name> -n <namespace> \
  -o jsonpath='{.status.conditions[?(@.type=="RayServiceReady")]}'

# Check if Weaviate is ready
kubectl get aiplatform <name> -n <namespace> \
  -o jsonpath='{.status.conditions[?(@.type=="WeaviateDatabaseReady")]}'

# Check if external access is ready
kubectl get aiplatform <name> -n <namespace> \
  -o jsonpath='{.status.conditions[?(@.type=="IngressReady")]}'
```

## Understanding Events

Events tell you what's happening as your platform deploys and runs.

### Normal Events (Good News)

These indicate successful operations:

| Event | Meaning |
|-------|---------|
| `RayServiceCreated` | Ray cluster was created successfully |
| `RayServiceReady` | Ray cluster is now operational |
| `RayClusterReady` | All Ray pods are running |
| `RayServeReady` | AI inference endpoints are available |
| `WeaviateCreated` | Vector database was created |
| `WeaviateReady` | Vector database is operational |
| `IngressCreated` | External access was configured |
| `IngressReady` | External URL is now available |
| `PlatformReady` | Everything is working! |

### Warning Events (Needs Attention)

These indicate problems that need investigation:

| Event | What's Wrong | What To Do |
|-------|--------------|------------|
| `PlatformDegraded` | One or more components failing | Check the message to see which components |
| `RayServiceNotReady` | Ray cluster is unhealthy | Check Ray pods and logs |
| `RayApplicationErrors` | AI models failed to load | Check application logs and model paths |
| `RayClusterNotReady` | Ray pods are failing | Check pod status and events |
| `WeaviateNotReady` | Vector database is failing | Check Weaviate pod status |
| `IngressNotReady` | External access lost | Check Ingress controller |

## Common Troubleshooting Scenarios

### Scenario 1: Platform Stuck in "Not Ready"

**Check what's failing:**
```bash
kubectl get aiplatform <name> -n <namespace> -o jsonpath='{.status.conditions}' | jq '.[] | select(.status=="False")'
```

This shows all components that aren't ready yet.

**Check recent events:**
```bash
kubectl get events -n <namespace> --field-selector involvedObject.name=<name> --sort-by='.lastTimestamp' | tail -20
```

### Scenario 2: AI Models Won't Load

**Symptoms:**
- Events show `RayApplicationErrors`
- Status condition `RayServeRouteReady` is False

**Check which models are failing:**
```bash
# View detailed error messages
kubectl get aiplatform <name> -n <namespace> \
  -o jsonpath='{.status.conditions[?(@.type=="RayServeRouteReady")].message}'

# Check Ray Serve logs
kubectl logs -l ray.io/cluster=<name> -n <namespace> | grep -i error
```

**Common causes:**
- Model files not in S3/object storage
- Wrong S3 bucket path in `objectStorage.path`
- IAM permissions issues (IRSA not configured correctly)
- Model files are corrupted or wrong format

### Scenario 3: Weaviate Database Issues

**Symptoms:**
- Events show `WeaviateNotReady`
- Status condition `WeaviateDatabaseReady` is False

**Check Weaviate status:**
```bash
# Check StatefulSet
kubectl get statefulset <name>-weaviate -n <namespace>

# Check pods
kubectl get pods -l app=<name>-weaviate -n <namespace>

# Check logs
kubectl logs <name>-weaviate-0 -n <namespace>
```

**Common causes:**
- Persistent volume not provisioned (check PVC)
- Resource limits too low
- Storage class not available

### Scenario 4: Can't Access from Outside

**Symptoms:**
- Ingress is enabled but can't access the URL
- Status condition `IngressReady` is False

**Check Ingress status:**
```bash
# View Ingress resource
kubectl get ingress <name> -n <namespace>

# Check if address is assigned
kubectl describe ingress <name> -n <namespace>

# Check Ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller
```

**Common causes:**
- Ingress controller not installed
- DNS not pointing to LoadBalancer IP
- Wrong Ingress class name
- Certificate not issued (if using TLS)

## View Detailed Errors

### Ray Application Errors

When AI models fail to load, you'll see detailed errors:

```bash
# View application errors
kubectl get events -n <namespace> \
  --field-selector involvedObject.name=<name>,reason=RayApplicationErrors

# Check specific application logs
kubectl logs -l ray.io/node-type=worker -n <namespace> | grep <app-name>
```

**Example error messages:**
- `FileNotFoundError: model_artifacts/my-model/model.bin` → Check S3 path
- `CUDA_VISIBLE_DEVICES is set to empty string` → GPU configuration issue
- `RuntimeError: CUDA out of memory` → Increase GPU resources

### Weaviate Errors

```bash
# View Weaviate errors
kubectl get events -n <namespace> \
  --field-selector involvedObject.name=<name>,reason=WeaviateNotReady

# Check Weaviate logs
kubectl logs <name>-weaviate-0 -n <namespace>
```

### Pod-Level Errors

Sometimes individual pods fail:

```bash
# List all pods
kubectl get pods -n <namespace> -l ai.splunk.com/platform=<name>

# Check failing pods
kubectl describe pod <pod-name> -n <namespace>

# View pod logs
kubectl logs <pod-name> -n <namespace>
```

## Event Timeline

During deployment, you'll typically see events in this order:

1. **Creation Phase** (1-2 minutes)
   - `RayServiceCreating`
   - `RayServiceCreated`
   - `WeaviateCreating`
   - `WeaviateCreated`
   - `IngressCreating` (if enabled)
   - `IngressCreated` (if enabled)

2. **Startup Phase** (2-5 minutes)
   - `RayClusterReady` - Pods are running
   - `WeaviateReady` - Database is running
   - `RayServiceReady` - Ray is operational

3. **Application Loading** (5-15 minutes depending on model sizes)
   - Model artifacts downloading from S3
   - Models loading into GPU memory
   - `RayServeReady` - AI inference ready

4. **Ready!**
   - `IngressReady` (if enabled) - External access available
   - `PlatformReady` - Everything is operational

## Monitoring in Production

### Set Up Alerts

Monitor Warning events to catch problems early:

```bash
# Count Warning events
kubectl get events -n <namespace> --field-selector type=Warning

# Watch for specific problems
kubectl get events -n <namespace> --watch --field-selector reason=PlatformDegraded
```

### Integration with Monitoring Systems

Export events to your monitoring system:

**Prometheus:**
```yaml
# Example PromQL query
rate(kube_event_count{type="Warning",involved_object_kind="AIPlatform"}[5m]) > 0
```

**Splunk:**
Configure the Splunk operator to forward events to your Splunk instance.

## Getting Help

If you're still stuck:

1. **Collect diagnostics:**
```bash
# Save all relevant information
kubectl get aiplatform <name> -n <namespace> -o yaml > aiplatform.yaml
kubectl get events -n <namespace> > events.txt
kubectl get pods -n <namespace> > pods.txt
kubectl logs <pod-name> -n <namespace> > pod-logs.txt
```

2. **Check operator logs:**
```bash
kubectl logs -n splunk-ai-operator-system \
  deployment/splunk-ai-operator-controller-manager
```

3. **Report an issue:** Include the diagnostics files when reporting issues

## Summary

**Use Status Conditions** to understand current state:
```bash
kubectl get aiplatform <name> -o jsonpath='{.status.conditions}'
```

**Use Events** to understand what happened:
```bash
kubectl get events --field-selector involvedObject.name=<name>
```

**Use Logs** for detailed debugging:
```bash
kubectl logs <pod-name>
```

For more technical details about the event system, see [Event Coverage](EVENT_COVERAGE.md) and [Event Strategy](EVENT_STRATEGY.md).
