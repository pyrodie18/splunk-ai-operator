package features

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FeatureHandler interface {
	BuildResources(namespace, name string) []client.Object
}
