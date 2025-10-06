#!/usr/bin/env sh

echo "updating operator image in manifest file using kustomize"
echo "cd config/manager ; kustomize edit set image controller=${SKAFFOLD_IMAGE}"
#sed "s/namespace: splunk-operator/namespace: splunk-operator/g"  config/default/kustomization.yaml 
#sed "s/value: WATCH_NAMESPACE_VALUE/value: \"\"/g"  config/default/kustomization.yaml 
#sed "s|SPLUNK_ENTERPRISE_IMAGE|splunk/splunk:9.3.0|g"  config/default/kustomization.yaml
cd config/manager ; kustomize edit set image controller=${SKAFFOLD_IMAGE}
