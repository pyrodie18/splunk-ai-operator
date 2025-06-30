# Splunk AI Platform Helm Installation

## Splunk AI Helm Chart Repository

Add the Splunk AI Platform Helm repository and update:

```bash
helm repo add splunk-ai https://splunk.github.io/splunk-ai-operator/
helm repo update
```

This repository includes the following charts:

* `splunk-ai/splunk-ai-operator`: Deploys the Splunk AI Operator (controller for CRDs like `AIPlatform`)
* `splunk-ai/splunk-ai-platform`: Deploys the full AI platform stack via an `AIPlatform` custom resource

> **Note:** Helm does not manage CRD upgrades. To upgrade CRDs, run:

```bash
git clone https://github.com/splunk/splunk-ai-operator.git
cd splunk-ai-operator
git checkout release/0.1.0
make install
```

---

## Install the Splunk AI Operator

To install the controller that manages `AIPlatform` resources:

```bash
helm install splunk-ai-operator splunk-ai/splunk-ai-operator \
  -n splunk-ai-operator --create-namespace \
  --set installCRDs=true
```

You can inspect all configurable values using:

```bash
helm show values splunk-ai/splunk-ai-operator
```

---

## Deploy the Splunk AI Platform

To deploy the full AI Platform stack using the `splunk-ai-platform` chart, you only need to define a few core fields in your `values.yaml` file.

### ✨ Example: `ai-platform-values.yaml`

```yaml
aiPlatform:
  enabled: true
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
helm install splunk-ai-platform splunk-ai/splunk-ai-platform \
  -n ai-stack --create-namespace \
  -f ai-platform-values.yaml \
  --set installCRDs=true
```

To upgrade:

```bash
helm upgrade splunk-ai-platform splunk-ai/splunk-ai-platform \
  -n ai-stack -f ai-platform-values.yaml
```

To uninstall:

```bash
helm uninstall splunk-ai-platform -n ai-stack
```

---

## View Running Resources

Once installed, confirm the AI platform resources are running:

```bash
kubectl get aiplatform -n ai-stack
kubectl get pods -n ai-stack
```

---

## Learn More

* [Helm Documentation](https://helm.sh/docs/)
* [Splunk AI Operator GitHub](https://github.com/splunk/splunk-ai-operator)
* View all default values:

```bash
helm show values splunk-ai/splunk-ai-platform
```
