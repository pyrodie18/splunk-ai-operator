# Reference Architecture

To set up the Splunk AI Operator, follow the steps in this document to verify everything in your setup exists as expected.

- [Reference Architecture](#reference-architecture)
  - [AWS EKS Setup](#aws-eks-setup)
    - [Create a Cluster Config](#create-a-cluster-config)
    - [Deploy the Cluster Config](#deploy-the-cluster-config)
    - [Ensure OIDC Provider](#ensure-oidc-provider)
    - [Install Cluster Add Ons](#install-cluster-add-ons)
    - [EBS Pod Identity Role and Association](#ebs-pod-identity-role-and-association)
    - [Create gp3 Storage Class](#create-gp3-storage-class)
  - [Prerequisite App Installation](#prerequisite-app-installation)
    - [Cluster Autoscaler](#cluster-autoscaler)
    - [NVIDIA Device Plugin](#nvidia-device-plugin)
    - [Uncordon Ready Nodes](#uncordon-ready-nodes)
    - [Kube Prometheus Stack](#kube-prometheus-stack)
    - [Cert Manager](#cert-manager)
    - [OpenTelemetry Operator](#opentelemtry-operator)
    - [Ray Operator](#ray-operator)
  - [Splunk Setup](#splunk-setup)
    - [Splunk Operator Installation](#splunk-operator-installation)
    - [Splunk AI Operator Installation](#splunk-ai-operator-installation)
    - [S3 Bucket Setup](#s3-bucket-setup)
      - [IAM Policy for S3 Bucket](#iam-policy-for-s3-bucket)
      - [IRSA for Service Accounts](#irsa-for-service-accounts)
    - [Splunk Standalone Installation](#splunk-standalone-installation)
    - [Splunk AI Platform CR Installation](#splunk-ai-platform-cr-installation)

## AWS EKS Setup
The first step is creating a Kubernetes cluster that the Splunk AI operator and Splunk AI Operator CRs will run on. For now, the supported insfrastructure is AWS EKS clusters.

### Create a Cluster Config
The cluster config should include the following:
 - name
 - region
 - service account for the ebs csi controller
 - vpcs
 - managed node groups

The cluster config should be saved to a file. In the following examples, the file name is `eks-cluster-config.yaml`. An example of a cluster config is:
```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: cluster-name
  region: us-west-2

iam:
  withOIDC: true
  serviceAccounts:
    - metadata:
        name: ebs-csi-controller-sa
        namespace: kube-system
      attachPolicyARNs:
        - arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy
      roleName: AmazonEKS_EBS_CSI_DriverRole
      wellKnownPolicies:
        ebsCSIController: true

vpc:
  subnets:
    private:
      ...
    public:
      ...

managedNodeGroups:

  - name: cpu-nodes
    instanceType: m5.xlarge
    desiredCapacity: 4
    minSize: 2
    maxSize: 8
    volumeSize: 500
    volumeType: gp3
    tags:
      Name: cluster-name-cpu
      Environment: prod
      kubernetes.io/cluster/cluster-name: owned
      k8s.io/cluster-autoscaler/enabled: "true"
      k8s.io/cluster-autoscaler/cluster-name: owned
  - name: gpu-nodes
    instanceType: g6e.24xlarge
    desiredCapacity: 1
    minSize: 0
    maxSize: 3
    volumeSize: 1000
    volumeType: gp3
    tags:
      Name: cluster-name-gpu
      Environment: prod
      kubernetes.io/cluster/cluster-name: owned
      k8s.io/cluster-autoscaler/enabled: "true"
      k8s.io/cluster-autoscaler/cluster-name: owned
    taints:
      - key: "dedicated"
        value: "gpu"
        effect: "NoSchedule"
```

### Deploy the Cluster Config
Now that the cluster config is created, next is to deploy the cluster config using the following command:
```bash
eksctl create cluster -f eks-cluster-config.yaml
```

The cluster creation will take a few minutes. When the command completes, verify that the kubeconfig has been updated to point to the newly created cluster to continue with the deployments.

### Ensure OIDC Provider
An OIDC Provider is required to create pvcs and other storage requirements during dpeloyment. Verify the OIDC provider is active with the following command:
```bash
aws eks describe-cluster --name "cluster-name" --query 'cluster.identity.oidc.issuer' --output text
```

If there is no output, or the output is None, then run the following command to associate the oidc provider with the cluster:
```bash
eksctl utils associate-iam-oidc-provider --region "us-west-2" --cluster "cluster-name" --approve
```

### Install Cluster Add Ons
The eks-pod-identity-agent and aws-ebs-csi-driver add ons are required for the cluster. Create them with the following commands:
```bash
eksctl create addon --cluster "cluster-name" --name eks-pod-identity-agent --force
eksctl create addon --cluster "cluster-name" --name aws-ebs-csi-driver --force 
```

### EBS Pod Identity Role and Association
For the eks-pod-identity-agent and aws-ebs-csi-driver add ons to work, they need roles and associations created.

1. Create the policy file. Update the `__REGION__` and `__ACCOUNT_ID__` fields with the information for your cluster.
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EKSPodIdentityTrust",
      "Effect": "Allow",
      "Principal": { "Service": "pods.eks.amazonaws.com" },
      "Action": [ "sts:AssumeRole", "sts:TagSession" ],
      "Condition": {
        "StringEquals": { "aws:SourceAccount": "__ACCOUNT_ID__" },
        "StringLike":   { "aws:SourceArn": "arn:aws:eks:__REGION__:__ACCOUNT_ID__:podidentityassociation/*" }
      }
    }
  ]
}
```
2. Create the pod identity role with the following command:
```bash
aws iam create-role --role-name "role-name" --assume-role-policy-document "path/to/policy/file"
```
3. Attach the AmazonEBSCSIDriverPolicy with the following command:
```bash
aws iam attach-role-policy --role-name "role-name" --policy-arn "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
```
4. Create a pod identity association for the service account for the ebs csi controller with the following command:
```bash
aws eks create-pod-identity-association --cluster-name "cluster-name" --namespace "kube-system" --service-account "ebs-csi-controller-sa" --role-arn "arn:aws:iam::${ACCOUNT_ID}:role/role-name"
```

### Create gp3 Storage Class
Create the storage class file to apply. In the following examples, the file name is `storageclass.yaml`.
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  fsType: ext4
reclaimPolicy: Retain
volumeBindingMode: WaitForFirstConsumer
```

Apply the storage class with the following command:
```bash
kubectl apply -f storageclass.yaml
```

## Prerequisite App Installation
There are a few deployments that have to be available in order for the Splunk AI Operator to work correctly. Install the following to continue with the setup.

### Cluster Autoscaler
The cluster autoscaler requires an iamserviceaccount to be created. Start by running the following command:
```bash
eksctl create iamserviceaccount  --cluster "cluster-name" \
    --name "cluster-autoscaler" \
    --namespace "kube-system" \
    --role-name "ClusterAutoscalerRole-cluster-name" \
    --attach-policy-arn arn:aws:iam::aws:policy/AutoScalingFullAccess \
    --approve \
    --override-existing-serviceaccounts
```

Next, verify the helm chart is up to date.
```bash
helm repo add autoscaler https://kubernetes.github.io/autoscaler
helm repo update
```

Finally, install the cluster-autoscaler helm chart with the following command:
```bash
helm_retry 5 upgrade --install "cluster-autoscaler" autoscaler/cluster-autoscaler \
    --namespace "kube-system" \
    --set autoDiscovery.clusterName="cluster-name" \
    --set awsRegion="us-west-2" \
    --set rbac.serviceAccount.create=false \
    --set rbac.serviceAccount.name="cluster-autoscaler" \
    --set image.repository=registry.k8s.io/autoscaling/cluster-autoscaler \
    --set image.tag="v1.31.2" \
    --set extraArgs.balance-similar-node-groups=true \
    --set extraArgs.skip-nodes-with-system-pods=false \
    --set extraArgs.expander=least-waste \
    --wait --timeout 15m
```

### NVIDIA Device Plugin
The NVIDIA device plugin allows for managing the GPUs on the cluster. Install it with the following commands:
```bash
kubectl apply -n kube-system -f "https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.17.3/deployments/static/nvidia-device-plugin.yml"
kubectl -n kube-system rollout status ds/nvidia-device-plugin-daemonset --timeout=10m
```

### Uncordon Ready Nodes
Some of the processes can leave nodes on the cluster unschedulable. Set them back to a good state with the following steps.
1. Get the list of nodes that are marked as SchedulingDisabled
```bash
kubectl get nodes --no-headers | awk '/SchedulingDisabled/ {print $1}'
```
2. For each of the nodes in the output from Step 1, check if they are in the Ready state
```bash
kubectl get node "<node-name>" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
```
3. For each node in the Ready state, uncordon the node
```bash
kubectl uncordon "<node-name>"
```

### Kube Prometheus Stack
Set up Kubernetes cluster monitoring with the kube prometheus stack deployment.

First, verify the helm chart is up to date.
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
```

Then, install the kube-prometheus-stack helm chart with the following command:
```bash
helm_retry 5 upgrade --install kube-prometheus prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace --wait --timeout 15m
```

### Cert Manager
Cert manager is required to create and manage TLS certificates on the cluster.

First, verify the helm chart is up to date.
```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
```

Then, install the cert-manager helm chart with the following command:
```bash
helm_retry 5 upgrade --install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true --wait --timeout 15m
```

### OpenTelemtry Operator
OpenTelemetry facilitates the generation, export, and collection of telemetry data.

First, verify the helm chart is up to date.
```bash
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```

Then, install the ope helm chart with the following command:
```bash
helm_retry 5 upgrade --install otel-operator open-telemetry/opentelemetry-operator --namespace observability --create-namespace --set admissionWebhooks.certManager.enabled=true --wait --timeout 15m
```

Installing the OpenTelemetry Collector depends on the apiversion of the OTel api version. In the following two examples, the config file should be named otel_collector_config.yaml.
If the OTel api version is v1beta1, use:
```yaml
apiVersion: ${apiversion}
kind: OpenTelemetryCollector
metadata:
  name: otel-collector
  namespace: observability
spec:
  image: ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:latest
  mode: deployment
  replicas: 1
  config:
    receivers:
      otlp:
        protocols: { grpc: {}, http: {} }
    processors: { batch: {} }
    exporters: { debug: {} }
    service:
      pipelines:
        traces:  { receivers: [otlp], processors: [batch], exporters: [debug] }
        metrics: { receivers: [otlp], processors: [batch], exporters: [debug] }
        logs:    { receivers: [otlp], processors: [batch], exporters: [debug] }
```

Otherwise, use:
```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otel-collector
  namespace: observability
spec:
  image: ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:latest
  mode: deployment
  replicas: 1
  config: |
    receivers:
      otlp:
        protocols:
          grpc: {}
          http: {}
    processors:
      batch: {}
    exporters:
      debug: {}
    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [debug]
        metrics:
          receivers: [otlp]
          processors: [batch]
          exporters: [debug]
        logs:
          receivers: [otlp]
          processors: [batch]
          exporters: [debug]
```

Install the OpenTelemetry Collector with the following command:
```bash
kubectl apply --server-side --force-conflicts -f otel_collector_config.yaml
```

### Ray Operator
The Ray Operator aides in managing Ray services for scaling the AI application.

Install the Ray Operator with the following command:
```bash
kubectl apply -k "github.com/ray-project/kuberay/ray-operator/config/default?ref=v1.2.2" --server-side --force-conflicts
```

## Splunk Setup

### Splunk Operator Installation
The Splunk Operator creates and manages Splunk custom resources. A Splunk instance is requried to run the Splunk AI Assitant app.

Install the Splunk Operator with the following command:
```bash
kubectl apply -f https://github.com/splunk/splunk-operator/releases/download/3.0.0/splunk-operator-cluster.yaml --server-side --force-conflicts
```

Verify that the Splunk Operator and Splunk Enterprise versions used support the Splunk AI Assistant app.

### Splunk AI Operator Installation
The Splunk AI Operator handles the Ray Services, and AI Platform and AI Service custom resources to install the Splunk AI Assistant app on the deployed splunk instance.

First, download the artifacts.yaml file for the Splunk AI Operator. 

Next, create the namespace if it does not exist yet with the following command:
```bash
kubectl create ns splunk-ai-operator-system
```

Install the Splunk AI Operator with the following command:
```bash
kubectl apply -f artifacts.yaml --server-side --force-conflicts
```

### S3 Bucket Setup
The AI Platform expects the S3 bucket to have specific prefixes for the folders, and apps uploaded.

Create an S3 bucket with a unique name that will be used in the CRs. In the bucket, create three folders, with the exact names `artifacts/`, `apps/`, and `tasks/`. Upload the Splunk_AI_Assistant_Cloud.tgz app into the `apps/` folder.

Next, create the namespace where the Splunk and Splunk IA Platform deployment will be created with the following command:
```bash
kubectl create ns ai-platform
```

#### IAM Policy for S3 Bucket
Create an IAM policy for the S3 bucket by first creating the following policy file:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    { "Sid":"ListBucket","Effect":"Allow","Action":["s3:ListBucket"],"Resource":"arn:aws:s3:::${bucket}" },
    { "Sid":"ObjectRW","Effect":"Allow","Action":["s3:GetObject","s3:PutObject","s3:DeleteObject","s3:AbortMultipartUpload","s3:ListMultipartUploadParts","s3:ListBucketMultipartUploads"],"Resource":"arn:aws:s3:::${bucket-name}/*" }
  ]
}
```

Then, create the policy with the following command:
```bash
aws iam create-policy --policy-name S3Access-cluster-name-ai-platform --policy-document "file://policy_document.json" --query 'Policy.Arn' --output text
```

Save the output policy arn for the following IRSA for Service Accounts steps.

#### IRSA for Service Accounts
Create an IRSA role for the Ray Head Service Account with the following command:
```bash
eksctl create iamserviceaccount \
    --cluster cluster-name \
    --namespace ai-platform \
    --name ray-head-sa \
    --role-name IRSA-cluster-name-ray-head-sa \
    --attach-policy-arn <policy arn from s3 bucket policy> \
    --approve \
    --override-existing-serviceaccounts
```

Create an IRSA role for the Ray Worker Service Account with the following command:
```bash
eksctl create iamserviceaccount \
    --cluster cluster-name \
    --namespace ai-platform \
    --name ray-worker-sa \
    --role-name IRSA-cluster-name-ray-worker-sa \
    --attach-policy-arn <policy arn from s3 bucket policy> \
    --approve \
    --override-existing-serviceaccounts
```

Create an IRSA role for the SAIA Service Account with the following command:
```bash
eksctl create iamserviceaccount \
    --cluster cluster-name \
    --namespace ai-platform \
    --name saia-service-sa \
    --role-name IRSA-cluster-name-saia-service-sa \
    --attach-policy-arn <policy arn from s3 bucket policy> \
    --approve \
    --override-existing-serviceaccounts
```

### Splunk Standalone Installation
A Splunk Standalone instance is needed to install and use the Splunk AI Assistant app. 

First, create an s3 secret to connect to the s3 bucket with the following command:
```bash
kubectl -n ai-platform create secret generic s3-secret --from-literal=s3_access_key="$AWS_ACCESS_KEY_ID" --from-literal=s3_secret_key="$AWS_SECRET_ACCESS_KEY"
```

Next, create a configmap for the Splunk defaults:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: splunk-defaults
data:
  default.yml: |
    splunk:
      conf:
        - key: authentication
          value:
            directory: /opt/splunk/etc/system/local
            content:
              oauth2_settings:
                issuer_uri: https://splunk-splunk-standalone-standalone-service:8089
                certFile: $SPLUNK_HOME/etc/auth/server.pem
                sslPassword: password
```
```bash
kubectl -n ai-platform apply -f configmap.yaml
```

Then, create a standalone instance with appRepo sources pointing to the s3 bucket.
```yaml
apiVersion: enterprise.splunk.com/v4
kind: Standalone
metadata:
  name: splunk-standalone
  namespace: ai-platform
spec:
  serviceAccount: saia-service-sa
  etcVolumeStorageConfig:
    storageClassName: gp3
  varVolumeStorageConfig:
    storageClassName: gp3
  volumes:
    - name: defaults
      configMap:
        name: splunk-defaults
  defaultsUrl: /mnt/defaults/default.yml
  appRepo:
    appInstallPeriodSeconds: 90
    appSources:
      - name: apps
        scope: local
        location: apps
    appsRepoPollIntervalSeconds: 60
    defaults:
      scope: local
      volumeName: volume_app_repo
    installMaxRetries: 2
    volumes:
      - name: volume_app_repo
        provider: aws
        storageType: s3
        endpoint: https://s3.amazonaws.com
        region: us-west-2
        path: bucket-name
        secretRef: s3-secret
```
```bash
kubectl apply -f standalone.yaml --server-side --force-conflicts
```

### Splunk AI Platform CR Installation
Start by finding the latest Splunk standlone secret. Run the following command, and choose the version with the highest number:
```bash
kubectl get secrets -n ai-platform
```
The correct secret is the secret with the name `splunk-splunk-standalone-standalone-secret-v1`, or that of the highest version.

Apply the cert-manager CR with the following spec:
```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned-issuer
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: platform-issuer
spec:
  isCA: true
  commonName: my-selfsigned-ca
  secretName: root-secret
  privateKey: { algorithm: ECDSA, size: 256 }
  issuerRef: { name: selfsigned-issuer, kind: Issuer, group: cert-manager.io }
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: my-ca-issuer
spec:
  ca: { secretName: root-secret }
```
```bash
kubectl -n ai-platform apply --server-side --force-conflicts -f cert_manager.yaml
```

Apply the AI Platform CR with the following spec:
```yaml
apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: splunk-ai-stack
spec:
  objectStorage:
    path: s3://bucket-name
    region: us-west-2
  serviceAccountName: ray-head-sa
  defaultAcceleratorType: L40S
  features:
    - name: saia
      version: "1.1.0"
      serviceAccountName: saia-service-sa
  storage:
    vectorDB:
      size: 50Gi
      storageClassName: gp3
  workerGroupSpec:
    serviceAccountName: ray-worker-sa
    gpuConfigs:
      - tier: g6e.12xlarge-0-gpu
        minReplicas: 0
        maxReplicas: 10
        gpusPerPod: 0
        resources:
          limits: { cpu: "16", memory: "32Gi", ephemeral-storage: "10Gi", nvidia.com/gpu: "0" }
          requests: { cpu: "4" }
      - tier: g6e.12xlarge-1-gpu
        minReplicas: 0
        maxReplicas: 10
        gpusPerPod: 1
        resources:
          requests: { cpu: "4" }
          limits: { cpu: "16", memory: "16Gi", ephemeral-storage: "50Gi", nvidia.com/gpu: "1" }
      - tier: g6e.12xlarge-2-gpu
        minReplicas: 0
        maxReplicas: 10
        gpusPerPod: 2
        resources:
          requests: { cpu: "1" }
          limits: { cpu: "2", memory: "48Gi", ephemeral-storage: "100Gi", nvidia.com/gpu: "2" }
      - tier: g6e.12xlarge-4-gpu
        minReplicas: 0
        maxReplicas: 10
        gpusPerPod: 4
        resources:
          requests: { cpu: "1" }
          limits: { cpu: "4", memory: "64Gi", ephemeral-storage: "200Gi", nvidia.com/gpu: "4" }
  cpuScheduler: {}
  gpuScheduler:
    tolerations:
      - key: "nvidia.com/gpu"
        operator: "Equal"
        value: "true"
        effect: "NoSchedule"
  ingress:
    className: nginx
    hosts:
      - host: ai.example.com
        paths: [ { path: "/", pathType: Prefix } ]
    tls:
      - hosts: [ ai.example.com ]
        secretName: ai-platform-tls
  splunkConfiguration:
    endpoint: splunk-standalone-standalone-service
    secretRef: { name: ${secret_name} }
  certificateRef: platform-issuer
```
```bash
kubectl -n ai-platform apply --server-side --force-conflicts -f ai_platform.yaml
```

Verify that the Splunk AI Assistant app is deployed on the standalone instance. Run the following command and see that the deploy status is complete:
```bash
kubectl get standalone splunk-standalone -n ai-platform -o yaml
```

Finally, edit the splunkaiassistant.conf file on the standalone pod to set the configurations.
Exec into the pod using the following command:
```bash
kubectl exec -it splunk-splunk-standalone-standalone-0 -n ai-platform -- bash
```

Find the splunkaiassistant.conf file on the pod.
```bash
cd /opt/splunk/etc/apps/Splunk_AI_Assistant_Cloud/default
cat splunkaiassistant.conf
```
If the file does not exist, create it.

Edit the contents of splunkaiassistant.conf to be the following:
```
[splunk_ai_assistant]
feedback_enabled=true

[cloud_connected_configurations]

[cloud_connected_configurations:proxy_settings]

[saia_sok_configurations]
saia_sok_enabled=true
saia_sok_url=http://splunk-ai-stack-saia-saia-service:8080
```

Restart the Splunk instance with the following command:
```bash
/opt/bin/splunk restart
```

Wait for the pod to come up, connect to it, and start using the Splunk AI Assistant app!