package telemetry

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestReconcileCounter(t *testing.T) {
	// Reset metrics before test
	registry := prometheus.NewRegistry()
	reconcileCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_reconcile_total",
			Help: "Total number of reconciliations",
		},
		[]string{"controller", "result"},
	)
	registry.MustRegister(reconcileCounter)

	// Increment counter
	reconcileCounter.WithLabelValues("aiplatform", "success").Inc()
	reconcileCounter.WithLabelValues("aiplatform", "success").Inc()
	reconcileCounter.WithLabelValues("aiplatform", "error").Inc()

	// Verify counts
	successCount := testutil.ToFloat64(reconcileCounter.WithLabelValues("aiplatform", "success"))
	assert.Equal(t, float64(2), successCount)

	errorCount := testutil.ToFloat64(reconcileCounter.WithLabelValues("aiplatform", "error"))
	assert.Equal(t, float64(1), errorCount)
}

func TestReconcileDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	reconcileDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_reconcile_duration_seconds",
			Help:    "Duration of reconciliation operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller"},
	)
	registry.MustRegister(reconcileDuration)

	// Record some durations
	reconcileDuration.WithLabelValues("aiplatform").Observe(0.1)
	reconcileDuration.WithLabelValues("aiplatform").Observe(0.5)
	reconcileDuration.WithLabelValues("aiservice").Observe(0.2)

	// Verify observations were recorded - test that the histogram is registered and working
	metrics, err := registry.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)

	// Find our histogram metric
	var found bool
	for _, mf := range metrics {
		if mf.GetName() == "test_reconcile_duration_seconds" {
			found = true
			assert.Equal(t, 2, len(mf.GetMetric())) // 2 label combinations
		}
	}
	assert.True(t, found, "Expected to find histogram metric")
}

func TestReplicaGauge(t *testing.T) {
	registry := prometheus.NewRegistry()
	replicaGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_replicas",
			Help: "Number of replicas",
		},
		[]string{"namespace", "name"},
	)
	registry.MustRegister(replicaGauge)

	// Set gauge values
	replicaGauge.WithLabelValues("default", "service1").Set(3)
	replicaGauge.WithLabelValues("default", "service2").Set(5)

	// Verify values
	service1Replicas := testutil.ToFloat64(replicaGauge.WithLabelValues("default", "service1"))
	assert.Equal(t, float64(3), service1Replicas)

	service2Replicas := testutil.ToFloat64(replicaGauge.WithLabelValues("default", "service2"))
	assert.Equal(t, float64(5), service2Replicas)

	// Update value
	replicaGauge.WithLabelValues("default", "service1").Set(10)
	updatedReplicas := testutil.ToFloat64(replicaGauge.WithLabelValues("default", "service1"))
	assert.Equal(t, float64(10), updatedReplicas)
}

func TestAPILatency(t *testing.T) {
	registry := prometheus.NewRegistry()
	apiLatency := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "test_api_latency_seconds",
			Help:       "API request latency",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"method", "endpoint"},
	)
	registry.MustRegister(apiLatency)

	// Record some latencies
	apiLatency.WithLabelValues("GET", "/api/v1/platforms").Observe(0.1)
	apiLatency.WithLabelValues("GET", "/api/v1/platforms").Observe(0.15)
	apiLatency.WithLabelValues("POST", "/api/v1/services").Observe(0.3)

	// Verify observations were recorded - test that the summary is registered and working
	metrics, err := registry.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)

	// Find our summary metric
	var found bool
	for _, mf := range metrics {
		if mf.GetName() == "test_api_latency_seconds" {
			found = true
			assert.Equal(t, 2, len(mf.GetMetric())) // 2 label combinations
		}
	}
	assert.True(t, found, "Expected to find summary metric")
}

func TestConditionStatus(t *testing.T) {
	registry := prometheus.NewRegistry()
	conditionGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_condition_status",
			Help: "Status of resource conditions",
		},
		[]string{"namespace", "name", "condition", "status"},
	)
	registry.MustRegister(conditionGauge)

	// Set condition statuses
	conditionGauge.WithLabelValues("default", "platform1", "Ready", "True").Set(1)
	conditionGauge.WithLabelValues("default", "platform1", "Ready", "False").Set(0)
	conditionGauge.WithLabelValues("default", "platform2", "Ready", "Unknown").Set(0)

	// Verify values
	readyTrue := testutil.ToFloat64(conditionGauge.WithLabelValues("default", "platform1", "Ready", "True"))
	assert.Equal(t, float64(1), readyTrue)

	readyFalse := testutil.ToFloat64(conditionGauge.WithLabelValues("default", "platform1", "Ready", "False"))
	assert.Equal(t, float64(0), readyFalse)
}

func TestTimerHelper(t *testing.T) {
	registry := prometheus.NewRegistry()
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_operation_duration_seconds",
			Help:    "Duration of operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
	registry.MustRegister(histogram)

	// Simulate timed operation
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	duration := time.Since(start).Seconds()

	histogram.WithLabelValues("test_op").Observe(duration)

	// Verify duration was recorded - test that the histogram is registered and working
	metrics, err := registry.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)

	// Find our histogram metric
	var found bool
	for _, mf := range metrics {
		if mf.GetName() == "test_operation_duration_seconds" {
			found = true
			assert.Equal(t, 1, len(mf.GetMetric())) // 1 label combination
			// Verify that a sample was recorded (histogram has observations)
			if len(mf.GetMetric()) > 0 {
				assert.Greater(t, mf.GetMetric()[0].GetHistogram().GetSampleCount(), uint64(0))
			}
		}
	}
	assert.True(t, found, "Expected to find histogram metric")
}

func TestErrorCounter(t *testing.T) {
	registry := prometheus.NewRegistry()
	errorCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_errors_total",
			Help: "Total number of errors",
		},
		[]string{"controller", "error_type"},
	)
	registry.MustRegister(errorCounter)

	// Record errors
	errorCounter.WithLabelValues("aiplatform", "validation").Inc()
	errorCounter.WithLabelValues("aiplatform", "storage").Inc()
	errorCounter.WithLabelValues("aiplatform", "storage").Inc()
	errorCounter.WithLabelValues("aiservice", "deployment").Inc()

	// Verify counts
	validationErrors := testutil.ToFloat64(errorCounter.WithLabelValues("aiplatform", "validation"))
	assert.Equal(t, float64(1), validationErrors)

	storageErrors := testutil.ToFloat64(errorCounter.WithLabelValues("aiplatform", "storage"))
	assert.Equal(t, float64(2), storageErrors)

	deploymentErrors := testutil.ToFloat64(errorCounter.WithLabelValues("aiservice", "deployment"))
	assert.Equal(t, float64(1), deploymentErrors)
}

func TestResourceGauge(t *testing.T) {
	registry := prometheus.NewRegistry()
	resourceGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_resource_count",
			Help: "Number of resources",
		},
		[]string{"type", "namespace"},
	)
	registry.MustRegister(resourceGauge)

	// Set resource counts
	resourceGauge.WithLabelValues("aiplatform", "default").Set(5)
	resourceGauge.WithLabelValues("aiplatform", "prod").Set(10)
	resourceGauge.WithLabelValues("aiservice", "default").Set(15)

	// Verify counts
	defaultPlatforms := testutil.ToFloat64(resourceGauge.WithLabelValues("aiplatform", "default"))
	assert.Equal(t, float64(5), defaultPlatforms)

	prodPlatforms := testutil.ToFloat64(resourceGauge.WithLabelValues("aiplatform", "prod"))
	assert.Equal(t, float64(10), prodPlatforms)

	// Delete resource
	resourceGauge.DeleteLabelValues("aiplatform", "default")

	// Verify deletion
	deletedCount := testutil.ToFloat64(resourceGauge.WithLabelValues("aiplatform", "default"))
	assert.Equal(t, float64(0), deletedCount)
}
