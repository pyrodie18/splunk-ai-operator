# Splunk AI Operator Installation Guide

## Download Installation YAML for Customization

To customize the installation of the **Splunk AI Operator**, start by downloading the installation YAML to your local system and open it in your preferred editor.

```bash
wget -O splunk-ai-operator-cluster.yaml https://github.com/splunk/splunk-ai-operator/releases/download/0.1.0/splunk-ai-operator-cluster.yaml
```

## Default Installation (Cluster-Scoped)

By default, the `splunk-ai-operator` is installed in the `splunk-ai-operator` namespace and is configured to watch **all namespaces** in the cluster for Splunk AI custom resources.

```bash
wget -O splunk-ai-operator-cluster.yaml https://github.com/splunk/splunk-ai-operator/releases/download/0.1.0/splunk-ai-operator-cluster.yaml
kubectl apply -f splunk-ai-operator-cluster.yaml --server-side
```

If you want to change the namespace or scope, edit the namespace and RBAC fields in the YAML before applying.

## Namespace-Scoped Installation with Restricted Permissions

To run the operator in **single namespace mode** with limited permissions, use the namespace-scoped YAML manifest:

```bash
wget -O splunk-ai-operator-namespace.yaml https://github.com/splunk/splunk-ai-operator/releases/download/0.1.0/splunk-ai-operator-namespace.yaml
kubectl apply -f splunk-ai-operator-namespace.yaml --server-side
```

This creates only Role/RoleBinding for the namespace in which the operator is deployed.

## Private Registry Support

If you're using a private container registry, update the `manager` image field and any `RELATED_IMAGE_*` environment variables in the deployment:

```yaml
image: <your-private-registry>/splunk-ai-operator:0.1.0
```

Example for related images (e.g., if using AI Platform components like Ray, Weaviate):

```yaml
env:
- name: RELATED_IMAGE_RAY_OPERATOR
  value: <your-private-registry>/ray-operator:latest
- name: RELATED_IMAGE_WEAVIATE
  value: <your-private-registry>/weaviate:latest
```

## Distroless Image Support

Splunk AI Operator supports **distroless images** to improve security by reducing the attack surface.

### Using the Distroless Image

Use the `-distroless` image tag:

```yaml
image: splunk/splunk-ai-operator:0.1.0-distroless
```

### Debugging Distroless Images

To debug, add a **sidecar container** with shell utilities:

```yaml
containers:
- name: manager
  image: splunk/splunk-ai-operator:0.1.0-distroless
- name: debug-sidecar
  image: ubuntu:20.04
  command: ["/bin/bash", "-c", "tail -f /dev/null"]
  volumeMounts:
    - name: ai-data
      mountPath: /data
volumes:
- name: ai-data
  emptyDir: {}
```

Access the sidecar:

```bash
kubectl exec -it <ai-operator-pod> -c debug-sidecar -- /bin/bash
```

## Custom Cluster Domain

If your cluster uses a custom domain (not `cluster.local`), set the `CLUSTER_DOMAIN` environment variable in the operator's deployment:

```yaml
env:
- name: CLUSTER_DOMAIN
  value: "internal.mycluster"
```

## Deploy the Splunk AI Platform

After the operator is installed, it can manage the CRDs for the Splunk AI Platform. The Splunk AI Platform CR will create the necessary Splunk AI Service CRs, based on the `features` listed in the manifest.

See [Custom Resources Documentation](CustomResources.md) for more information on configuring the Splunk AI Platform on your cluster.