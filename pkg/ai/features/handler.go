package features

import (
	"context"

	aiv1 "github.com/splunk/splunk-ai-operator/api/v1"
)

type FeatureHandler interface {
	Reconcile(ctx context.Context, aiservice *aiv1.AIService) error
}
