package telemetry

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	labelNamespace     = "namespace"
	labelName          = "name"
	labelKind          = "kind"
	labelFeature       = "feature"
	labelPhase         = "phase"
	labelStage         = "stage"
	labelResult        = "result"
	labelErrorType     = "error_type"
	labelMethodName    = "api"
	labelModuleName    = "module"
	labelOutcome       = "outcome"
	labelConditionType = "condition"
	labelCondStatus    = "status"
)

var (
	reconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "splunk_ai_assistant_reconcile_duration_seconds",
			Help:    "Duration of reconcile stages",
			Buckets: prometheus.DefBuckets,
		},
		[]string{labelKind, labelStage},
	)

	reconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "splunk_ai_assistant_reconcile_total",
			Help: "Total number of reconciliations by result",
		},
		[]string{labelKind, labelResult},
	)

	reconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "splunk_ai_assistant_reconcile_errors_total",
			Help: "Total number of reconcile errors by type",
		},
		[]string{labelKind, labelErrorType},
	)

	observedGeneration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "splunk_ai_assistant_observed_generation",
			Help: "Observed .status.observedGeneration for the CR",
		},
		[]string{labelNamespace, labelName, labelKind},
	)

	conditionStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "splunk_ai_assistant_condition_status",
			Help: "Condition status per condition type (1=True, 0=False/Unknown)",
		},
		[]string{labelNamespace, labelName, labelKind, labelConditionType, labelCondStatus},
	)

	desiredReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "splunk_ai_assistant_spec_desired_replicas",
			Help: "Desired replicas from spec",
		},
		[]string{labelNamespace, labelName, labelKind, labelFeature},
	)

	readyReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "splunk_ai_assistant_status_ready_replicas",
			Help: "Ready replicas",
		},
		[]string{labelNamespace, labelName, labelKind, labelFeature},
	)

	apiRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "splunk_ai_assistant_api_requests_total",
			Help: "Number of external or module API calls",
		},
		[]string{labelNamespace, labelName, labelKind, labelFeature, labelModuleName, labelMethodName, labelOutcome},
	)

	apiLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "splunk_ai_assistant_api_latency_seconds",
			Help:    "Latency for external or module API calls",
			Buckets: prometheus.DefBuckets,
		},
		[]string{labelNamespace, labelName, labelKind, labelFeature, labelModuleName, labelMethodName},
	)

	resourceGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "splunk_ai_assistant_resources",
			Help: "Inventory of CRs by kind/feature/phase",
		},
		[]string{labelKind, labelFeature, labelPhase},
	)
)

func init() {
	metrics.Registry.MustRegister(
		reconcileDuration,
		reconcileTotal,
		reconcileErrors,
		observedGeneration,
		conditionStatus,
		desiredReplicas,
		readyReplicas,
		apiRequests,
		apiLatency,
		resourceGauge,
	)
}

// ---------- Helpers that accept Scope (perfect for pkg/ai) ----------

func ObserveReconcileStageS(s Scope, stage string, start time.Time) {
	reconcileDuration.With(prometheus.Labels{
		labelKind:  s.Kind,
		labelStage: stage,
	}).Observe(time.Since(start).Seconds())
}

func ObserveReconcileResultS(s Scope, result string) {
	reconcileTotal.With(prometheus.Labels{
		labelKind:   s.Kind,
		labelResult: result,
	}).Inc()
}

func ObserveReconcileErrorS(s Scope, errorType string) {
	if errorType == "" {
		errorType = "unknown"
	}
	reconcileErrors.With(prometheus.Labels{
		labelKind:      s.Kind,
		labelErrorType: errorType,
	}).Inc()
}

func SetObservedGenerationS(s Scope, gen int64) {
	observedGeneration.With(prometheus.Labels{
		labelNamespace: s.Namespace,
		labelName:      s.Name,
		labelKind:      s.Kind,
	}).Set(float64(gen))
}

func SetConditionS(s Scope, condType, status string) {
	v := 0.0
	switch status {
	case "True":
		v = 1.0
	case "False", "Unknown":
	default:
		status = "Unknown"
	}
	conditionStatus.With(prometheus.Labels{
		labelNamespace:     s.Namespace,
		labelName:          s.Name,
		labelKind:          s.Kind,
		labelConditionType: condType,
		labelCondStatus:    status,
	}).Set(v)
}

func SetDesiredReplicasS(s Scope, desired int32) {
	feat := s.Feature
	if feat == "" {
		feat = "platform"
	}
	desiredReplicas.With(prometheus.Labels{
		labelNamespace: s.Namespace,
		labelName:      s.Name,
		labelKind:      s.Kind,
		labelFeature:   feat,
	}).Set(float64(desired))
}

func SetReadyReplicasS(s Scope, ready int32) {
	feat := s.Feature
	if feat == "" {
		feat = "platform"
	}
	readyReplicas.With(prometheus.Labels{
		labelNamespace: s.Namespace,
		labelName:      s.Name,
		labelKind:      s.Kind,
		labelFeature:   feat,
	}).Set(float64(ready))
}

func IncAPIRequestS(s Scope, module, api, outcome string) {
	if outcome == "" {
		outcome = "ok"
	}
	feat := s.Feature
	if feat == "" {
		feat = "platform"
	}
	apiRequests.With(prometheus.Labels{
		labelNamespace:  s.Namespace,
		labelName:       s.Name,
		labelKind:       s.Kind,
		labelFeature:    feat,
		labelModuleName: module,
		labelMethodName: api,
		labelOutcome:    outcome,
	}).Inc()
}

func ObserveAPILatencyS(s Scope, module, api string, start time.Time) {
	feat := s.Feature
	if feat == "" {
		feat = "platform"
	}
	apiLatency.With(prometheus.Labels{
		labelNamespace:  s.Namespace,
		labelName:       s.Name,
		labelKind:       s.Kind,
		labelFeature:    feat,
		labelModuleName: module,
		labelMethodName: api,
	}).Observe(time.Since(start).Seconds())
}

func SetResourceCount(kind, feature, phase string, count int) {
	if feature == "" {
		feature = "platform"
	}
	resourceGauge.With(prometheus.Labels{
		labelKind:    kind,
		labelFeature: feature,
		labelPhase:   phase,
	}).Set(float64(count))
}

// ---------- Context-friendly wrappers (for deep stacks) ----------

func ScopeFrom(ctx context.Context) Scope {
	if s, ok := FromContext(ctx); ok {
		return s
	}
	return Scope{} // empty scope (no labels). You may prefer to no-op instead.
}

func ObserveReconcileStage(ctx context.Context, stage string, start time.Time) {
	ObserveReconcileStageS(ScopeFrom(ctx), stage, start)
}
func ObserveReconcileResult(ctx context.Context, result string) {
	ObserveReconcileResultS(ScopeFrom(ctx), result)
}
func ObserveReconcileError(ctx context.Context, errorType string) {
	ObserveReconcileErrorS(ScopeFrom(ctx), errorType)
}
func SetObservedGeneration(ctx context.Context, gen int64) {
	SetObservedGenerationS(ScopeFrom(ctx), gen)
}
func SetCondition(ctx context.Context, condType, status string) {
	SetConditionS(ScopeFrom(ctx), condType, status)
}
func SetDesiredReplicas(ctx context.Context, desired int32) {
	SetDesiredReplicasS(ScopeFrom(ctx), desired)
}
func SetReadyReplicas(ctx context.Context, ready int32) {
	SetReadyReplicasS(ScopeFrom(ctx), ready)
}
func IncAPIRequest(ctx context.Context, module, api, outcome string) {
	IncAPIRequestS(ScopeFrom(ctx), module, api, outcome)
}
func ObserveAPILatency(ctx context.Context, module, api string, start time.Time) {
	ObserveAPILatencyS(ScopeFrom(ctx), module, api, start)
}
