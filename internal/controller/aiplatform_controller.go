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

	aiv1 "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/splunk/splunk-ai-operator/internal/controller/common"
	aiplatform "github.com/splunk/splunk-ai-operator/pkg/ai"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"github.com/splunk/splunk-ai-operator/pkg/config"
)

// +kubebuilder:rbac:groups=ai.splunk.com,resources=aiplatforms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ai.splunk.com,resources=aiplatforms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ai.splunk.com,resources=aiplatforms/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=podmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ray.io,resources=rayservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ray.io,resources=rayclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ray.io,resources=rayjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ray.io,resources=jobs,verbs=get;list;watch;create;update;patch;delete
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

// AIPlatformReconciler reconciles a AIPlatform
type AIPlatformReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Config   *config.OperatorConfig // injected runtime config
}

func (r *AIPlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	p := &aiv1.AIPlatform{}
	if err := r.Get(ctx, req.NamespacedName, p); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	aiplatform := aiplatform.New(p, r.Client, r.Scheme, r.Recorder)
	return aiplatform.Reconcile(ctx, p)
}

// --- 8️⃣ reconcileStatus: update CR status/conditions ---
func (r *AIPlatformReconciler) reconcileStatus(ctx context.Context, p *aiv1.AIPlatform) error {
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

// SetupWithManager sets up the controller with the Manager.
func (r *AIPlatformReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1.AIPlatform{}).
		WithEventFilter(predicate.Or(
			common.GenerationChangedPredicate(),
			common.AnnotationChangedPredicate(),
			common.LabelChangedPredicate(),
			common.SecretChangedPredicate(),
			common.DeploymentChangedPredicate(),
			common.PodChangedPredicate(),
			common.ConfigMapChangedPredicate(),
			common.CrdChangedPredicate(),
		)).
		Watches(&appsv1.Deployment{},
			handler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&aiv1.AIPlatform{},
			)).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&aiv1.AIPlatform{},
			)).
		Watches(&corev1.Pod{},
			handler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&aiv1.AIPlatform{},
			)).
		Watches(&corev1.ConfigMap{},
			handler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&aiv1.AIPlatform{},
			)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: aiv1.TotalWorker,
		}).
		Complete(r)
}
