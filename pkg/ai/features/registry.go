package features

import (
	"github.com/splunk/splunk-ai-operator/pkg/ai/features/common"
	"github.com/splunk/splunk-ai-operator/pkg/ai/features/saia"
	"github.com/splunk/splunk-ai-operator/pkg/ai/features/seca"
)

var FeatureFactories = map[string]common.FeatureFactory{
	"saia": &saia.SaiaFactory{},
	"seca": &seca.SecaFactory{},
}
