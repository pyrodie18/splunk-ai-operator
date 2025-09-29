package ai_platform

import (
	"context"
	"fmt"
	"os"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AIPlatformReconciler) ReconcileWeaviateDatabaseStatus(ctx context.Context, p *aiApi.AIPlatform) error {
	// 1️⃣ Fetch the up-to-date StatefulSet for Weaviate
	sts := &appsv1.StatefulSet{}
	key := types.NamespacedName{Namespace: p.Namespace, Name: fmt.Sprintf("%s-weaviate", p.Name)}
	if err := r.Get(ctx, key, sts); err != nil {
		return err
	}

	// 2️⃣ Update the status based on StatefulSet readiness
	ready := metav1.ConditionFalse
	reason := "WeaviateNotReady"
	msg := "Weaviate database is not ready"
	if sts.Status.ReadyReplicas == *sts.Spec.Replicas {
		ready = metav1.ConditionTrue
		reason = "WeaviateReady"
		msg = "Weaviate database is ready"
	}

	cond := metav1.Condition{
		Type:               "WeaviateReady",
		Status:             ready,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	}
	meta.SetStatusCondition(&p.Status.Conditions, cond)

	// 3️⃣ Add Weaviate service name to status
	p.Status.VectorDbServiceName = fmt.Sprintf("%s-weaviate", p.Name)

	return nil
}

// ReconcileWeaviateDatabase manages ServiceAccount, StatefulSet, and Service for Weaviate
func (r *AIPlatformReconciler) ReconcileWeaviateDatabase(ctx context.Context, instance *aiApi.AIPlatform) error {
	// Resolve Weaviate image from env
	weaviateImage := os.Getenv("RELATED_IMAGE_WEAVIATE")
	if weaviateImage == "" {
		return fmt.Errorf("RELATED_IMAGE_WEAVIATE environment variable is required")
	}

	// Derive default values
	name := fmt.Sprintf("%s-weaviate", instance.Name)
	defaultReplicas := int32(1)
	defaultSA := name

	replicas := &defaultReplicas
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}

	labels := map[string]string{"app": name}

	// 1) Ensure ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultSA,
			Namespace: instance.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(instance, sa, r.Scheme); err != nil {
		return err
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error { return nil }); err != nil {
		return err
	}

	// 2) Ensure StatefulSet
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(instance, sts, r.Scheme); err != nil {
		return err
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		sts.Spec.ServiceName = name
		sts.Spec.Replicas = replicas
		sts.Spec.Template.ObjectMeta.Labels = labels
		sts.Spec.Template.Spec.ServiceAccountName = defaultSA
		sts.Spec.Template.Spec.Affinity = instance.Spec.CPUSchedulingSpec.Affinity
		sts.Spec.Template.Spec.Tolerations = instance.Spec.CPUSchedulingSpec.Tolerations
		sts.Spec.Template.Spec.NodeSelector = instance.Spec.CPUSchedulingSpec.NodeSelector

		// Container definition
		sts.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:      "weaviate",
			Image:     weaviateImage,
			Resources: resources,
			Ports: []corev1.ContainerPort{{
				Name:          "http",
				ContainerPort: 8080,
			}},
		}}
		return nil
	}); err != nil {
		return err
	}

	// 3) Ensure Service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(instance, svc, r.Scheme); err != nil {
		return err
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Spec.Selector = labels
		svc.Spec.Ports = []corev1.ServicePort{{
			Name:       "http",
			Port:       80,
			TargetPort: intstr.FromInt(8080),
		}}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
