package saia

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SaiaHandler struct{}

func (h *SaiaHandler) BuildResources(namespace, name string) []client.Object {
	return []client.Object{
		ConfigMap(namespace, name),
		Secret(namespace, name),
		Deployment(namespace, name),
		Service(namespace, name),
	}
}
