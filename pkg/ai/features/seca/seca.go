package seca

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SecaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (h *SecaReconciler) BuildResources(namespace, name string) []client.Object {
	return []client.Object{
		ConfigMap(namespace, name),
		Secret(namespace, name),
		Deployment(namespace, name),
		Service(namespace, name),
	}
}
