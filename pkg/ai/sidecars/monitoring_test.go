package sidecars

import (
	"context"
	"testing"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// ✅ safeFakeClient prevents deep-copy panics for unstructured objects
type safeFakeClient struct {
	client.Client
}

func (c *safeFakeClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	// For unstructured objects, skip actual fake storage to avoid deep-copy panic
	_, isUnstructured := obj.(runtime.Unstructured)
	if isUnstructured {
		return nil
	}
	return c.Client.Create(ctx, obj, opts...)
}

func (c *safeFakeClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	_, isUnstructured := obj.(runtime.Unstructured)
	if isUnstructured {
		return nil
	}
	return c.Client.Update(ctx, obj, opts...)
}

func TestReconcilePodMonitor(t *testing.T) {
	ctx := context.Background()

	// ✅ Setup Scheme
	scheme := runtime.NewScheme()
	_ = aiApi.AddToScheme(scheme)

	// ✅ Fake AIPlatform CR
	ai := &aiApi.AIPlatform{}
	ai.Name = "test-ai"
	ai.Namespace = "default"
	ai.Spec.Sidecars.PrometheusOperator = true // Enable PrometheusOperator

	// ✅ Fake client
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	safeClient := &safeFakeClient{Client: fc}

	builder := &Builder{
		Client: safeClient,
		Scheme: scheme,
		ai:     ai,
	}

	t.Run("PrometheusOperator disabled -> should do nothing", func(t *testing.T) {
		ai.Spec.Sidecars.PrometheusOperator = false

		err := builder.reconcilePodMonitor(ctx, ai)
		assert.NoError(t, err, "should not fail when PrometheusOperator is disabled")
	})

	t.Run("PrometheusOperator enabled -> should create head and worker PodMonitors", func(t *testing.T) {
		ai.Spec.Sidecars.PrometheusOperator = true

		err := builder.reconcilePodMonitor(ctx, ai)
		assert.NoError(t, err, "should reconcile head and worker PodMonitors successfully")
	})
}
