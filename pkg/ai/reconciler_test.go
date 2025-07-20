package ai_platform

import (
	"context"
	"testing"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/stretchr/testify/assert"
	//corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	//"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

/*
func buildTestScheme(t *testing.T) *runtime.Scheme {
	s := runtime.NewScheme()
	err := aiApi.AddToScheme(s)
	assert.NoError(t, err)
	err = corev1.AddToScheme(s)
	assert.NoError(t, err)
	return s
} */

func TestBuildAIService_PopulatesExpectedFields(t *testing.T) {
	scheme := buildTestScheme(t)

	platform := &aiApi.AIPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ai",
			Namespace: "default",
		},
		Spec: aiApi.AIPlatformSpec{
			ObjectStorage: aiApi.ObjectStorageSpec{Path: "/data"},
			SplunkConfiguration: aiApi.SplunkConfiguration{
				Endpoint: "splunk-endpoint",
			},
			MTLS: aiApi.MTLSConfig{Enabled: true, Termination: "operator"},
		},
		Status: aiApi.AIPlatformStatus{
			VectorDbServiceName: "weaviate-db",
		},
	}

	feature := aiApi.FeatureSpec{
		Name:               "feature1",
		Version:            "v1",
		ServiceAccountName: "svc-account",
	}

	r := &AIPlatformReconciler{Scheme: scheme}

	service := r.buildAIService(context.Background(), platform, feature, "my-ai-feature1")

	assert.Equal(t, "my-ai-feature1", service.Name)
	assert.Equal(t, "default", service.Namespace)
	assert.Equal(t, "feature1", service.Spec.Feature.Name)
	assert.Equal(t, "svc-account", service.Spec.ServiceAccountName)
	assert.Equal(t, "weaviate-db", service.Spec.VectorDbUrl)
	assert.Equal(t, int32(1), service.Spec.Replicas)
	assert.True(t, service.Spec.Metrics.Enabled)
	assert.Equal(t, "/metrics", service.Spec.Metrics.Path)

	// Labels should include platform and feature
	assert.Equal(t, "my-ai", service.Labels["aiplatform"])
	assert.Equal(t, "feature1", service.Labels["feature"])
}

func TestReconcileFeatures_CreatesNewAIService(t *testing.T) {
	ctx := context.Background()
	scheme := buildTestScheme(t)

	platform := &aiApi.AIPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ai",
			Namespace: "default",
		},
		Spec: aiApi.AIPlatformSpec{
			Features: []aiApi.FeatureSpec{
				{Name: "feature1", Version: "v1", ServiceAccountName: "svc-account"},
			},
			ObjectStorage: aiApi.ObjectStorageSpec{Path: "/data"},
		},
		Status: aiApi.AIPlatformStatus{
			VectorDbServiceName: "weaviate-db",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform).
		Build()

	reconciler := &AIPlatformReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Act → should create the AIService because it doesn't exist
	err := reconciler.ReconcileFeatures(ctx, platform)
	assert.NoError(t, err)

	// Assert → AIService should now exist
	created := &aiApi.AIService{}
	serviceKey := types.NamespacedName{Name: "my-ai-feature1", Namespace: "default"}
	err = fakeClient.Get(ctx, serviceKey, created)
	assert.NoError(t, err, "AIService should have been created")

	assert.Equal(t, "feature1", created.Spec.Feature.Name)
	assert.Equal(t, "my-ai", created.Spec.AIPlatformRef.Name)
	assert.Equal(t, "weaviate-db", created.Spec.VectorDbUrl)
}

func TestReconcileFeatures_DoesNotRecreateExistingAIService(t *testing.T) {
	ctx := context.Background()
	scheme := buildTestScheme(t)

	// Existing AIService
	existingService := &aiApi.AIService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ai-feature1",
			Namespace: "default",
		},
	}

	platform := &aiApi.AIPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ai",
			Namespace: "default",
		},
		Spec: aiApi.AIPlatformSpec{
			Features: []aiApi.FeatureSpec{
				{Name: "feature1", Version: "v1", ServiceAccountName: "svc-account"},
			},
			ObjectStorage: aiApi.ObjectStorageSpec{Path: "/data"},
		},
		Status: aiApi.AIPlatformStatus{
			VectorDbServiceName: "weaviate-db",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, existingService).
		Build()

	reconciler := &AIPlatformReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Act → should NOT recreate because it already exists
	err := reconciler.ReconcileFeatures(ctx, platform)
	assert.NoError(t, err)

	// Assert → Still only one AIService, no duplication
	fetched := &aiApi.AIService{}
	serviceKey := types.NamespacedName{Name: "my-ai-feature1", Namespace: "default"}
	err = fakeClient.Get(ctx, serviceKey, fetched)
	assert.NoError(t, err)
	assert.Equal(t, "my-ai-feature1", fetched.Name)
}
