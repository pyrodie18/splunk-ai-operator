package splunkutils

import (
	"context"
	"testing"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubernetesSecretResolver_GetHECToken(t *testing.T) {
	ns := "test-ns"
	secretName := GetNamespaceScopedSecretName(ns)
	ctx := context.Background()

	// Register scheme
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name        string
		objects     []client.Object
		expectedVal string
		expectedErr string
	}{
		{
			name: "Secret exists with hec_token",
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: ns,
					},
					Data: map[string][]byte{
						"hec_token": []byte("supersecret-token"),
					},
				},
			},
			expectedVal: "supersecret-token",
			expectedErr: "",
		},
		{
			name:        "Secret not found",
			objects:     []client.Object{}, // no secret in fake client
			expectedVal: "",
			expectedErr: "failed to get namespace-scoped Splunk secret",
		},
		{
			name: "Secret exists but missing hec_token key",
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: ns,
					},
					Data: map[string][]byte{
						"wrong_key": []byte("value"),
					},
				},
			},
			expectedVal: "",
			expectedErr: "missing hec_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build fake client with test-specific objects
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			resolver := &KubernetesSecretResolver{Client: fakeClient}

			cfg := &aiApi.SplunkConfigurationSpec{}
			got, err := resolver.GetHECToken(ctx, ns, cfg)

			if tt.expectedErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVal, got)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}
