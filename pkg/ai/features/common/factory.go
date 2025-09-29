package common

import (
	"context"

	aiv1 "github.com/splunk/splunk-ai-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FeatureHandler is implemented by all feature reconcilers
type FeatureHandler interface {
	Reconcile(ctx context.Context, ai *aiv1.AIService) error
}

// FeatureFactory creates a FeatureHandler
type FeatureFactory interface {
	New(ctx context.Context, c client.Client, scheme *runtime.Scheme, ai *aiv1.AIService, Recorder record.EventRecorder) (FeatureHandler, error)
}
