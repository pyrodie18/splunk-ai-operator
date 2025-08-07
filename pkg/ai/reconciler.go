package ai_platform

import (
	"context"
	"fmt"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/splunk/splunk-ai-operator/pkg/ai/raybuilder"
	"github.com/splunk/splunk-ai-operator/pkg/ai/sidecars"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type AIPlatformReconciler struct {
	p *aiApi.AIPlatform
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func New(p *aiApi.AIPlatform, client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) *AIPlatformReconciler {
	return &AIPlatformReconciler{
		p:        p,
		Client:   client,
		Scheme:   scheme,
		Recorder: recorder,
	}
}

func (r *AIPlatformReconciler) Reconcile(ctx context.Context, p *aiApi.AIPlatform) (reconcile.Result, error) {

	var conditions []metav1.Condition
	defer func() {
		// Fetch the latest version of the CR before updating the status
		latest := &aiApi.AIPlatform{}
		namespacedName := client.ObjectKey{Namespace: p.Namespace, Name: p.Name}
		if err := r.Get(ctx, namespacedName, latest); err != nil {
			log.FromContext(ctx).Error(err, "failed to fetch latest CR")
			return
		}
		latest.Status = p.Status
		latest.Status.Conditions = conditions
		latest.Status.ObservedGeneration = p.Generation
		latest.Status.RayServiceName = p.Status.RayServiceName
		latest.Status.VectorDbServiceName = p.Status.VectorDbServiceName
		_ = r.Status().Update(ctx, latest)
	}()
	raybuilder := raybuilder.New(r.p, r.Client, r.Scheme, r.Recorder)
	sidecarBuilder := sidecars.New(r.Client, r.Scheme, r.Recorder, r.p)

	stages := []struct {
		name string
		fn   func(context.Context, *aiApi.AIPlatform) error
	}{
		{"Validate", r.validate},
		//{"ApplicationsConfigMap", raybuilder.ReconcileApplicationsConfigMap},
		//{"InstancesConfigMap", raybuilder.ReconcileInstancesConfigMap},
		//{"ServeConfigMap", raybuilder.ReconcileServeConfigMap},
		{"Sidecars", sidecarBuilder.Reconcile},
		{"rayAutoscalerRBAC", raybuilder.ReconcileRayAutoscalerRBAC},
		{"RayService", raybuilder.ReconcileRayService},
		{"WeaviateDatabase", r.ReconcileWeaviateDatabase},
		// collect status of each stage
		{"RayServiceStatus", raybuilder.ReconcileRayServiceStatus},
		{"WeaviateDatabaseStatus", r.ReconcileWeaviateDatabaseStatus},
		{"AIService", r.ReconcileFeatures},
	}

	for _, stage := range stages {
		err := stage.fn(ctx, p)
		cond := metav1.Condition{
			Type:               stage.name + "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "Reconciled",
			Message:            "stage succeeded",
			LastTransitionTime: metav1.Now(),
		}
		if err != nil {
			cond.Status = metav1.ConditionFalse
			cond.Reason = "Error"
			cond.Message = err.Error()
			//r.Recorder.Event(p, corev1.EventTypeWarning, stage.name+"Failed", err.Error())
		} else {
			//r.Recorder.Event(p, corev1.EventTypeNormal, stage.name+"Succeeded", "stage succeeded")
		}
		conditions = append(conditions, cond)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// all done
	conditions = append(conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "AllReconciled",
		Message:            "all stages completed successfully",
		LastTransitionTime: metav1.Now(),
	})

	return reconcile.Result{}, nil
}

func (r *AIPlatformReconciler) ReconcileFeatures(ctx context.Context, platform *aiApi.AIPlatform) error {

	for _, feature := range platform.Spec.Features {
		serviceName := fmt.Sprintf("%s-%s", platform.Name, feature.Name)
		var existing aiApi.AIService
		err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: platform.Namespace}, &existing)

		if errors.IsNotFound(err) {
			newService := r.buildAIService(ctx, platform, feature, serviceName)
			if err := controllerutil.SetControllerReference(platform, newService, r.Scheme); err != nil {
				return err
			}
			if err := r.Create(ctx, newService); err != nil {
				return err
			}
			// You can log here if needed
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (r *AIPlatformReconciler) buildAIService(ctx context.Context, platform *aiApi.AIPlatform, feature aiApi.FeatureSpec, name string) *aiApi.AIService {
	vectorDbUrl := platform.Status.VectorDbServiceName

	return &aiApi.AIService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"aiplatform": platform.Name,
				"feature":    feature.Name,
			},
		},
		Spec: aiApi.AIServiceSpec{
			Feature: feature,
			Version: feature.Version,
			AIPlatformRef: corev1.ObjectReference{
				APIVersion: "ai.splunk.com/v1",
				Kind:       "AIPlatform",
				Name:       platform.Name,
				Namespace:  platform.Namespace,
			},
			ServiceAccountName:  feature.ServiceAccountName,
			TaskVolume:          platform.Spec.ObjectStorage,
			SplunkConfiguration: platform.Spec.SplunkConfiguration,
			VectorDbUrl:         vectorDbUrl,
			Replicas:            1,
			Metrics: aiApi.MetricsConfig{
				Enabled: true,
				Port:    8080,
				Path:    "/metrics",
			},
			MTLS: platform.Spec.MTLS,
		},
	}
}
