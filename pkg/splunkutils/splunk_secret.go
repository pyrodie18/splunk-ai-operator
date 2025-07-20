package splunkutils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespaceScopedSecretNameTemplateStr = "splunk-%s-secret"
	splunkSecretKeyHecToken              = "hec_token"
)

// GetNamespaceScopedSecretName returns the shared Splunk secret name for the namespace.
func GetNamespaceScopedSecretName(namespace string) string {
	return fmt.Sprintf(namespaceScopedSecretNameTemplateStr, namespace)
}

// GetHECTokenFromNamespaceScopedSecret fetches the HEC token from the namespace-level Splunk secret.
func GetHECTokenFromNamespaceScopedSecret(ctx context.Context, c client.Client, namespace string) (string, error) {
	secretName := GetNamespaceScopedSecretName(namespace)

	var secret corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, &secret); err != nil {
		return "", fmt.Errorf("failed to get namespace-scoped Splunk secret %q: %w", secretName, err)
	}

	hecTokenBytes, ok := secret.Data[splunkSecretKeyHecToken]
	if !ok {
		return "", fmt.Errorf("secret %q missing required key %q", secretName, splunkSecretKeyHecToken)
	}

	return string(hecTokenBytes), nil
}
