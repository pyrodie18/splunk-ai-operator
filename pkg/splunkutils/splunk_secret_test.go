package splunkutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetHECTokenFromNamespaceScopedSecret(t *testing.T) {
	ctx := context.Background()
	ns := "test-ns"
	secretName := GetNamespaceScopedSecretName(ns)

	// Register Secret type in scheme
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	t.Run("success: secret exists with hec_token", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: ns,
			},
			Data: map[string][]byte{
				splunkSecretKeyHecToken: []byte("supersecret-token"),
			},
		}

		fc := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		token, err := GetHECTokenFromNamespaceScopedSecret(ctx, fc, ns)

		assert.NoError(t, err)
		assert.Equal(t, "supersecret-token", token)
	})

	t.Run("error: secret not found", func(t *testing.T) {
		// No secret added to fake client → should fail
		fc := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		token, err := GetHECTokenFromNamespaceScopedSecret(ctx, fc, ns)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "failed to get namespace-scoped Splunk secret")
		assert.Contains(t, err.Error(), secretName)
	})

	t.Run("error: secret exists but missing hec_token key", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"wrong_key": []byte("some-value"),
			},
		}

		fc := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		token, err := GetHECTokenFromNamespaceScopedSecret(ctx, fc, ns)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "missing required key")
		assert.Contains(t, err.Error(), secretName)
	})
}
