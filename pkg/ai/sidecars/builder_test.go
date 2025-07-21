package sidecars

import (
	"context"
	"testing"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupFakeScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = aiApi.AddToScheme(s)
	return s
}

func TestReconcileFluentBitConfig(t *testing.T) {
	ctx := context.Background()
	scheme := setupFakeScheme()
	ns := "test-ns"
	name := "test-ai"

	t.Run("FluentBit disabled -> should return nil and do nothing", func(t *testing.T) {
		fc := fake.NewClientBuilder().WithScheme(scheme).Build()

		ai := &aiApi.AIPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: aiApi.AIPlatformSpec{
				Sidecars: aiApi.SidecarSpec{
					FluentBit: false,
				},
			},
		}

		builder := &Builder{
			Client: fc,
			Scheme: scheme,
			ai:     ai,
		}

		err := builder.reconcileFluentBitConfig(ctx, ai)
		assert.NoError(t, err)

		// ConfigMap should NOT exist
		cm := &corev1.ConfigMap{}
		cmName := name + "-fluentbit-config"
		err = fc.Get(ctx, clientKey(ns, cmName), cm)
		assert.Error(t, err)
	})

	t.Run("FluentBit enabled but Secret missing -> should return error", func(t *testing.T) {
		fc := fake.NewClientBuilder().WithScheme(scheme).Build()

		ai := &aiApi.AIPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: aiApi.AIPlatformSpec{
				Sidecars: aiApi.SidecarSpec{
					FluentBit: true,
				},
				SplunkConfiguration: aiApi.SplunkConfigurationSpec{
					SecretRef: corev1.SecretReference{
						Name: "missing-secret",
					},
					Endpoint: "https://splunk-endpoint",
				},
			},
		}

		builder := &Builder{
			Client: fc,
			Scheme: scheme,
			ai:     ai,
		}

		err := builder.reconcileFluentBitConfig(ctx, ai)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve secret")
	})

	t.Run("FluentBit enabled but Secret missing hec_token -> should return error", func(t *testing.T) {
		// Secret exists but without hec_token key
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "splunk-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{},
		}

		fc := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		ai := &aiApi.AIPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: aiApi.AIPlatformSpec{
				Sidecars: aiApi.SidecarSpec{
					FluentBit: true,
				},
				SplunkConfiguration: aiApi.SplunkConfigurationSpec{
					SecretRef: corev1.SecretReference{
						Name: "splunk-secret",
					},
					Endpoint: "https://splunk-endpoint",
				},
			},
		}

		builder := &Builder{
			Client: fc,
			Scheme: scheme,
			ai:     ai,
		}

		err := builder.reconcileFluentBitConfig(ctx, ai)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hec_token not found")
	})

	t.Run("FluentBit enabled with valid secret -> should create ConfigMap", func(t *testing.T) {
		// Secret exists with hec_token
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "splunk-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"hec_token": []byte("my-token"),
			},
		}

		fc := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		ai := &aiApi.AIPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: aiApi.AIPlatformSpec{
				Sidecars: aiApi.SidecarSpec{
					FluentBit: true,
				},
				SplunkConfiguration: aiApi.SplunkConfigurationSpec{
					SecretRef: corev1.SecretReference{
						Name: "splunk-secret",
					},
					Endpoint: "https://splunk-endpoint",
				},
			},
		}

		builder := &Builder{
			Client: fc,
			Scheme: scheme,
			ai:     ai,
		}

		err := builder.reconcileFluentBitConfig(ctx, ai)
		assert.NoError(t, err)

		// ✅ Verify ConfigMap created
		cm := &corev1.ConfigMap{}
		cmName := name + "-fluentbit-config"
		err = fc.Get(ctx, clientKey(ns, cmName), cm)
		assert.NoError(t, err)
		assert.Contains(t, cm.Data["fluent-bit.conf"], "https://splunk-endpoint")
		assert.Contains(t, cm.Data["fluent-bit.conf"], "my-token")
	})

	t.Run("FluentBit enabled and ConfigMap exists but needs update -> should update", func(t *testing.T) {
		// Secret exists with valid token
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "splunk-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"hec_token": []byte("updated-token"),
			},
		}

		// Existing ConfigMap with old data
		oldCm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name + "-fluentbit-config",
				Namespace: ns,
			},
			Data: map[string]string{
				"fluent-bit.conf": "old-data",
			},
		}

		fc := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret, oldCm).
			Build()

		ai := &aiApi.AIPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: aiApi.AIPlatformSpec{
				Sidecars: aiApi.SidecarSpec{
					FluentBit: true,
				},
				SplunkConfiguration: aiApi.SplunkConfigurationSpec{
					SecretRef: corev1.SecretReference{
						Name: "splunk-secret",
					},
					Endpoint: "https://splunk-endpoint",
				},
			},
		}

		builder := &Builder{
			Client: fc,
			Scheme: scheme,
			ai:     ai,
		}

		err := builder.reconcileFluentBitConfig(ctx, ai)
		assert.NoError(t, err)

		// ✅ Verify ConfigMap got updated
		updated := &corev1.ConfigMap{}
		err = fc.Get(ctx, clientKey(ns, name+"-fluentbit-config"), updated)
		assert.NoError(t, err)
		assert.Contains(t, updated.Data["fluent-bit.conf"], "updated-token")
	})
}

// helper for namespaced names
func clientKey(ns, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
}
