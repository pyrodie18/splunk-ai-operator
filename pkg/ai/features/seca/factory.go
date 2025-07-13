package seca

import (
	"context"

	aiv1 "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/splunk/splunk-ai-operator/pkg/ai/features/common"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SecaFactory struct{}

func (f *SecaFactory) New(ctx context.Context, c client.Client, scheme *runtime.Scheme, ai *aiv1.AIService) (common.FeatureHandler, error) {
	return &SecaReconciler{
		Client: c,
		Scheme: scheme,
	}, nil
}
