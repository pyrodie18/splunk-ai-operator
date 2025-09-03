# Reference Architecture

To set up the Splunk AI Operator, follow the steps in this document to verify everything in your setup exists as expected.

- [Reference Architecture](#reference-architecture)
  - [AWS EKS Setup](#aws-eks-setup)
    - [Create a Cluster Config](#create-a-cluster-config)
    - [Deploy the Cluster Config](#deploy-the-cluster-config)
    - [Ensure OIDC Provider](#ensure-oidc-provider)
    - [Install Cluster Add Ons](#install-cluster-add-ons) 

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

1. Create the policy file. Update the `__REGION__`, `__COLON__`, and `__ACCOUNT_ID__` fields with the information for your cluster.
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
        "StringLike":   { "aws:SourceArn": "arn:aws:eks:__REGION____COLON____ACCOUNT_ID__:podidentityassociation/*" }
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
aws eks create-pod-identity-association --cluster-name "cluster-name" --namespace "kube-system" --service-account "ebs-csi-controller-sa" --role-arn "arn:aws:iam::${ACCOUNT_ID}:role/EBSCSIDriverPodIdentityRole-cluster-name"
```

