# Custom Resource Guide

The Splunk AI Operator provides a collection of
[custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
you can use to manage Splunk AI Platform deployments in your Kubernetes cluster.

- [Custom Resource Guide](#custom-resource-guide)
  - [Metadata Parameters](#metadata-parameters)
  - [AI Platform Spec Parameters](#ai-platform-spec-parameters)
  - [AI Service Spec Parameters](#ai-service-spec-parameters)
  - [Examples of Guaranteed and Burstable QoS](#examples-of-guaranteed-and-burstable-qos)
    - [A Guaranteed QoS Class example:](#a-guaranteed-qos-class-example)
    - [A Burstable QoS Class example:](#a-burstable-qos-class-example)
    - [A BestEffort QoS Class example:](#a-besteffort-qos-class-example)
    - [Pod Resources Management](#pod-resources-management)
  - [Troubleshooting](#troubleshooting)
    - [CR Status Message](#cr-status-message)

For examples on how to use these custom resources, please see
[Configuring Splunk Enterprise Deployments](Examples.md).


## Metadata Parameters
All resources in Kubernetes include a `metadata` section. You can use this
to define a name for a specific instance of the resource, and which namespace
you would like the resource to reside within:

| Key       | Type   | Description                                                                                                 |
| --------- | ------ | ----------------------------------------------------------------------------------------------------------- |
| name      | string | Each instance of your resource is distinguished using this name.                                            |
| namespace | string | Your instance will be created within this namespace. You must ensure that this namespace exists beforehand. |

If you do not provide a `namespace`, you current context will be used.

```yaml
apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: example
  namespace: test
```

## AI Platform Spec Parameters

```yaml
apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: example
  labels:
    app.kubernetes.io/name: splunk-ai-platform-example
    app.kubernetes.io/instance: example
    app.kubernetes.io/version: 0.1.0
spec:
  objectStorage:
    path: "s3://bucketname/<path prefix>"
    region: "us-west-2"
    secretRef: s3-secret
  serviceAccountName: "controller-manager"
  features:
    - name: "saia"
      serviceAccountName: "saia-sa"
      version: "0.1.0"
  headGroupSpec:
    serviceAccountName: "head-group-sa"
    imageRegistry: "667741767953.dkr.ecr.us-west-2.amazonaws.com/ml-platform/ray/ray-head"
    nodeSelector: {}
    affinity: {}
    tolerations: []
  workerGroupSpec:
    serviceAccountName: "worker-sa"
    imageRegistry: "667741767953.dkr.ecr.us-west-2.amazonaws.com/ml-platform/ray/ray-worker-gpu"
    nodeSelector: {}
    affinity: {}
    tolerations: []
    gpuConfigs:
      tier: ""
      minReplicas: 0
      maxReplicas: 0
      gpusPerPod: 0
      resources:
        requests:
          memory: "12Gi"
          cpu: "24"
        limits:
          memory: "12Gi"
          cpu: "24"  
  sidecars:
    envoy: true
    fluentBit: true
    otel: true
    prometheusOperator: true
  certificateRef: "platform-issuer"
  clusterDomain: "cluster.local"
  images:
    saiaImage: "splunkai/saia:latest"
    weaviateImage: "docker.io/weaviate:latest"
    rayHeadGroupImage: "rayproject/ray-head:latest"
    rayWorkerGroupImage: "rayproject/ray-worker:latest"
  defaultAcceleratorType: "L40S"
  splunkConfiguration:
    crName: "splunk-standalone"
    crNamespace: "default"
    secretRef:
        name: "splunk-secret"
        namespace: "default"
    endpoint: "https://splunk.default.svc.cluster.local:8089"
    # Optional, if not using secretRef
    # token: "splunk-token"
  storage:
    vectorDB:
      pvcName: "pvc-vector-db"
      size: "100Gi"
      storageClassName: "gp2"
  gpuScheduler:
    nodeSelector: {}
    affinity: {}
    tolerations: []
  cpuScheduler:
    nodeSelector: {}
    affinity: {}
    tolerations: []
  ingress:
    enabled: false
```

The `AIPlatform` resource provides the following `Spec` configuration parameters:

| Key        | Type    | Description                                       |
| ---------- | ------- | ------------------------------------------------- |
| objectStorage   | object | Information for the related s3 bucket that holds the AIPlatform artifacts, tasks, and models. See [Service Artifacts Storage](ServiceArtifactsStorage.md) |
| serviceAccountName   | string | The name of the [Service Account](https://kubernetes.io/docs/concepts/security/service-accounts/) for the project |
| features   | array | List of features to be installed by the AI Platform |
| headGroupSpec   | object | Information for the Ray head group configuration |
| workerGroupSpec   | array | Information for the Ray worker group configuration |
| sidecars   | object | Boolean values for which sidecars to deploy |
| certificatRef   | string | cert-manager Certificate for mTLS |
| clusterDomain   | string | DNS suffix for in-cluster services |
| images   | object | List of image registries to use for Ray |
| defaultAcceleratorType   | string | Default accelerator type |
| splunkConfiguration   | object | Splunk Configuration instance reference |
| storage   | object | Storage configuration for the vectorDB |
| gpuScheduler   | object | Scheduling configuration for GPU nodes |
| cpuScheduler   | object | Scheduling configuration for CPU nodes |
| ingress   | object | Configuration for ingress to be created if enabled |

## AI Service Spec Parameters

The AIService CR is created by the AIPlatform CR, so there are no additional spec values to deploy an AIService CR on its own.

## Examples of Guaranteed and Burstable QoS

You can change the CPU and memory resources, and assign different Quality of Services (QoS) classes to your pods using the Kubernetes Quality of Service section. Here are some examples:

**TODO: Make sure the examples have the correct resources** 
  
### A Guaranteed QoS Class example:
Set equal ```requests``` and ```limits``` values for CPU and memory to establish a QoS class of Guaranteed. 

*Note: A pod will not start on a node that cannot meet the CPU and memory ```requests``` values.*

```yaml
apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: example
spec:
  imagePullPolicy: Always
  resources:
    requests:
      memory: "12Gi"
      cpu: "24"
    limits:
      memory: "12Gi"
      cpu: "24"  
```

### A Burstable QoS Class example:
Set the ```requests``` value for CPU and memory lower than the ```limits``` value to establish a QoS class of Burstable. 

Example: This Standalone AI Platform instance should start with minimal capacity, but will be allowed to scale up if Kubernetes is able to allocate additional CPU and Memory up to the ```limits``` values.

```yaml
apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: example
spec:
  imagePullPolicy: Always
  resources:
    requests:
      memory: "2Gi"
      cpu: "4"
    limits:
      memory: "12Gi"
      cpu: "24"  
```

### A BestEffort QoS Class example:
With no requests or limits values set for CPU and memory, the QoS class is set to BestEffort. The BestEffort QoS is not recommended for use with Splunk AI Operator.

### Pod Resources Management

__CPU Throttling__

Kubernetes starts throttling CPUs if a pod's demand for CPU exceeds the value set in the ```limits``` parameter. If your nodes have extra CPU resources available, leaving the ```limits``` value unset will allow the pods to utilize more CPUs.

## Troubleshooting

### CR Status Message
The Splunk AI Platform CRDs with the Splunk AI Operator have a field `cr.Status.message` which provides a detailed view of the CR's current status.

Here is an example of a Splunk AI Platform with a message indicating an invalid CR config:

**TODO: example**
