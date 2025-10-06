#!/bin/bash
set -x

curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
chmod +x ./kind
mv ./kind /usr/local/bin/kind

curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/linux/amd64
chmod +x kubebuilder
mv kubebuilder /usr/local/bin/

KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl"
chmod +x kubectl
mv kubectl /usr/local/bin/kubectl

docker network create -d=bridge --subnet=172.19.0.0/24 kind

kind version
kubebuilder version
docker --version
go version
kubectl version --client

wget -O /usr/local/bin/okta-artifactory-login https://repo.splunkdev.net/artifactory/generic/okta-cli/0.7.20250127-212405.2e86b53/linux/okta-artifactory-login
chmod +x /usr/local/bin/okta-artifactory-login
wget -O /usr/local/bin/okta-kube-token https://repo.splunkdev.net/artifactory/generic/okta-cli/0.7.20250127-212405.2e86b53/linux/okta-kube-token
chmod +x /usr/local/bin/okta-kube-token
wget -O /usr/local/bin/okta-aws-login https://repo.splunkdev.net/artifactory/generic/okta-cli/0.7.20250127-212405.2e86b53/linux/okta-aws-login
chmod +x /usr/local/bin/okta-aws-login