package features

import (
	"github.com/splunk/splunk-ai-operator/pkg/ai/features/saia"
	"github.com/splunk/splunk-ai-operator/pkg/ai/features/seca"
)

var FeatureHandlers = map[string]FeatureHandler{
	"saia": &saia.SaiaHandler{},
	"seca": &seca.SecaHandler{},
}
