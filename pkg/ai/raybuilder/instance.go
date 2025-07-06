package raybuilder

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ReadInstanceMapFromConfigMap(ctx context.Context, cl client.Client, name, namespace string) (InstanceMap, error) {
	cm := &corev1.ConfigMap{}
	err := cl.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cm)
	if err != nil {
		return nil, err
	}
	val, ok := cm.Data["instance.yaml"]
	if !ok {
		return nil, fmt.Errorf("instance.yaml not found in ConfigMap %s/%s", namespace, name)
	}

	var instanceMap InstanceMap
	if err := yaml.Unmarshal([]byte(val), &instanceMap); err != nil {
		return nil, err
	}
	return instanceMap, nil
}

func (b *Builder) ReconcileInstancesConfigMap(ctx context.Context, p *aiApi.AIPlatform) error {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      p.Name + "-instances",
		Namespace: p.Namespace,
	}}
	_, err := controllerutil.CreateOrUpdate(ctx, b.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		if _, exists := cm.Data["instance.yaml"]; !exists {
			home := os.Getenv("HOME")                                         // Get the home directory from environment variable
			home = "/Users/vivekr/Projects/splunk-ai-operator/config/configs" // For testing, use a fixed path FIXME TODO: remove this once we have a better way to handle multiple paths
			content, err := os.ReadFile(path.Join(home, "instance.yaml"))
			if err != nil {
				return err
			}
			cm.Data["instance.yaml"] = string(content)
		}
		return controllerutil.SetOwnerReference(p, cm, b.Scheme)
	})
	return err
}

func detectProvider(k8sClient client.Client, ctx context.Context) (string, error) {
	nodes := &corev1.NodeList{}
	if err := k8sClient.List(ctx, nodes); err != nil {
		return "", err
	}
	if provider, err := detectClusterProviderFromNodeLabels(k8sClient, ctx); err == nil {
		return provider, nil
	}
	return "", fmt.Errorf("could not detect cloud provider from nodes")
}

func detectClusterProviderFromNodeLabels(k8sClient client.Client, ctx context.Context) (string, error) {
	nodes := &corev1.NodeList{}
	if err := k8sClient.List(ctx, nodes); err != nil {
		return "", err
	}
	for _, node := range nodes.Items {
		labels := node.Labels
		switch {
		case labels["eks.amazonaws.com/nodegroup"] != "":
			return "aws", nil
		case labels["cloud.google.com/gke-nodepool"] != "":
			return "gcp", nil
		case labels["kubernetes.azure.com/cluster"] != "":
			return "azure", nil
		case strings.Contains(node.Name, "oke") || strings.Contains(labels["oke.oraclecloud.com/name"], "oke"):
			return "oracle", nil
		default:
			// Fallthrough case: on-prem, RKE, k3s, etc.
			return "generic", nil
		}
	}
	return "", fmt.Errorf("could not detect cluster provider from node labels")
}
