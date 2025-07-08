package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	labelNamespace       = "namespace"
	labelName            = "name"
	labelKind            = "kind"
	labelErrorType       = "error_type"
	labelMethodName      = "api"
	labelModuleName      = "module"
	labelResourceVersion = "resource_version"
)

var (
	reconcileHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "splunk_ai_assistant_reconcile_duration_seconds",
			Help: "Duration of reconcile stages",
		},
		[]string{"stage"},
	)
)

func getPrometheusLabels(request reconcile.Request, kind string) prometheus.Labels {
	return prometheus.Labels{
		labelNamespace: request.Namespace,
		labelName:      request.Name,
		labelKind:      kind,
	}
}

func init() {
	metrics.Registry.MustRegister(reconcileHistogram)
}
