package splunkutils

import (
	"context"
	"fmt"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesSecretResolver struct {
	Client client.Client
}

func (r *KubernetesSecretResolver) GetHECToken(ctx context.Context, namespace string, cfg *aiApi.SplunkConfigurationSpec) (string, error) {
	// use existing namespace-scoped secret logic
	secretName := GetNamespaceScopedSecretName(namespace)
	var secret corev1.Secret
	if err := r.Client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, &secret); err != nil {
		return "", fmt.Errorf("failed to get namespace-scoped Splunk secret %q: %w", secretName, err)
	}
	hecToken, ok := secret.Data["hec_token"]
	if !ok {
		return "", fmt.Errorf("secret %q missing hec_token", secretName)
	}
	return string(hecToken), nil
}
