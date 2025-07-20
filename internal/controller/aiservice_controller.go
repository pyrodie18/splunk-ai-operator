/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	aiv1 "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/splunk/splunk-ai-operator/pkg/ai/features"
	"github.com/splunk/splunk-ai-operator/pkg/config"
)

// AIServiceReconciler reconciles a AIService object
type AIServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Recorder record.EventRecorder
	Config   *config.OperatorConfig // injected runtime config
}

// +kubebuilder:rbac:groups=ai.splunk.com,resources=aiservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ai.splunk.com,resources=aiservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ai.splunk.com,resources=aiservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=podmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="core",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="monitoring",resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create;get;list;watch;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AIService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *AIServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling AIService", "name", req.Name, "namespace", req.Namespace)

	ai := &aiv1.AIService{}
	if err := r.Get(ctx, req.NamespacedName, ai); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Lookup factory by feature name
	factory, ok := features.FeatureFactories[ai.Spec.Feature.Name]
	if !ok {
		log.Error(nil, "No factory registered for feature", "feature", ai.Spec.Feature.Name)
		return ctrl.Result{}, nil
	}

	// Instantiate feature-specific reconciler via factory
	handler, err := factory.New(ctx, r.Client, r.Scheme, ai, r.Recorder)
	if err != nil {
		log.Error(err, "failed to initialize feature handler", "feature", ai.Spec.Feature.Name)
		return ctrl.Result{}, err
	}

	// Reconcile the feature
	if err := handler.Reconcile(ctx, ai); err != nil {
		log.Error(err, "feature reconciliation failed", "feature", ai.Spec.Feature.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AIServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1.AIService{}).
		Named("aiservice").
		Owns(&corev1.ServiceAccount{}).
		Owns(&certmanagerv1.Certificate{}).
		Owns(&batchv1.Job{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&monitoringv1.ServiceMonitor{}).
		Complete(r)
}

// --- 8️⃣ reconcileStatus: update CR status/conditions ---
func (r *AIServiceReconciler) reconcileStatus(ctx context.Context, p *aiv1.AIService) error {
	p.Status.ObservedGeneration = p.Generation
	cond := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "All resources are up-to-date",
		LastTransitionTime: metav1.Now(),
	}
	p.Status.Conditions = []metav1.Condition{cond}
	return r.Status().Update(ctx, p)
}
