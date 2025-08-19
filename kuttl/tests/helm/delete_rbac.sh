#!/usr/bin/env bash

echo "removing cluster-wide rbac roles"
kubectl delete $(kubectl get clusterrole -o name | grep splunk-ai-operator) 
kubectl delete $(kubectl get clusterrolebinding -o name | grep splunk-ai-operator)