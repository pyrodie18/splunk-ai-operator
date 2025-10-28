package saia

import (
	"context"
	"strings"

	"fmt"
	"os"
	"reflect"
	"sort"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	aiv1 "github.com/splunk/splunk-ai-operator/api/v1"
	common "github.com/splunk/splunk-ai-operator/pkg/ai/features/common"
	"github.com/splunk/splunk-ai-operator/pkg/splunkutils"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SaiaReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Reconcile runs reconciliation stages for the CR.
func (r *SaiaReconciler) Reconcile(ctx context.Context, aiservice *aiv1.AIService) error {
	log := log.FromContext(ctx)

	var conditions []metav1.Condition
	defer func() {
		aiservice.Status.Conditions = conditions
		aiservice.Status.ObservedGeneration = aiservice.Generation
		_ = r.Status().Update(ctx, aiservice)
	}()

	stages := []struct {
		name string
		fn   func(context.Context, *aiv1.AIService) error
	}{
		{"Validate", r.validateAIService},
		{"ServiceAccount", r.reconcileServiceAccount},
		{"SAIAConfigMap", r.reconcileSAIAConfigMap},
		{"Certificate", r.reconcileCertificate},
		{"PostInstallHook", r.reconcilePostInstallHook},
		{"SAIADeployment", r.reconcileSAIADeployment},
		{"SAIAService", r.reconcileSAIAService},
		{"ServiceMonitor", r.reconcileServiceMonitor},
	}

	for _, stage := range stages {
		err := stage.fn(ctx, aiservice)

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
			//r.Recorder.Event(ai, corev1.EventTypeWarning, stage.name+"Failed", err.Error())
		} else {
			//		r.Recorder.Event(ai, corev1.EventTypeNormal, stage.name+"Succeeded", "stage succeeded")
		}
		conditions = append(conditions, cond)
		if err != nil {
			log.Error(err, "stage failed", "stage", stage.name)
			return err
		}
	}

	conditions = append(conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "AllReconciled",
		Message:            "all stages completed successfully",
		LastTransitionTime: metav1.Now(),
	})

	return nil
}

// validateAIService ensures required fields are set and defaults.
func (r *SaiaReconciler) validateAIService(
	ctx context.Context,
	ai *aiv1.AIService,
) error {
	if os.Getenv("RELATED_IMAGE_POST_INSTALL_HOOK") == "" {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "RELATED_IMAGE_POST_INSTALL_HOOK must be set")
		return fmt.Errorf("RELATED_IMAGE_POST_INSTALL_HOOK must be set")
	}
	// Validate that either AIPlatformRef or explicit URLs are provided
	if ai.Spec.AIPlatformRef.Name == "" && ai.Spec.AIPlatformUrl == "" {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "AIPlatformRef.Name or AIPlatformUrl must be set")
		return fmt.Errorf("either AIPlatformRef.Name or AIPlatformUrl must be set")
	}

	// Fetch and validate AIPlatform if using AIPlatformRef
	if ai.Spec.AIPlatformRef.Name != "" {
		aiPlatform, err := r.getAIPlatform(ctx, ai.Spec.AIPlatformRef)
		if err != nil {
			r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "fetching AIPlatform failed")
			return fmt.Errorf("fetching AIPlatform: %w", err)
		}

		// Validate AIPlatform infrastructure is ready before using its status fields
		if err := r.validateAIPlatformReady(ctx, aiPlatform); err != nil {
			return fmt.Errorf("AIPlatform infrastructure not ready: %w", err)
		}

		// Validate Vector Database readiness
		if err := r.validateVectorDatabaseReady(ctx, aiPlatform); err != nil {
			return fmt.Errorf("vector database not ready: %w", err)
		}

		// Only populate URLs if not already set (preserve explicit user values)
		clusterDomain := ai.Spec.ClusterDomain
		if clusterDomain == "" {
			clusterDomain = "cluster.local"
		}
		if ai.Spec.AIPlatformUrl == "" {
			ai.Spec.AIPlatformUrl = fmt.Sprintf("%s.%s.svc.%s:8000",
				aiPlatform.Status.RayServiceName, ai.Spec.AIPlatformRef.Namespace, clusterDomain)
		}
		if ai.Spec.VectorDbUrl == "" {
			ai.Spec.VectorDbUrl = fmt.Sprintf("%s.%s.svc.%s",
				aiPlatform.Status.VectorDbServiceName, ai.Spec.AIPlatformRef.Namespace, clusterDomain)
		}
	}

	// Final validation that URLs are populated (either from AIPlatform or provided explicitly)
	if ai.Spec.AIPlatformUrl == "" {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "AIPlatformUrl is not set")
		return fmt.Errorf("AIPlatformUrl must be set (either from AIPlatformRef or explicitly)")
	}
	if ai.Spec.VectorDbUrl == "" {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "VectorDbUrl is not set")
		return fmt.Errorf("VectorDbUrl must be set (either from AIPlatformRef or explicitly)")
	}

	// Default resources
	if ai.Spec.Resources.Requests == nil {
		ai.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		}
	}
	if ai.Spec.Resources.Limits == nil {
		ai.Spec.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		}
	}
	if ai.Spec.TaskVolume.Path == "" {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "task volume path must be set")
		return fmt.Errorf("task volume path must be set")
	}
	if ai.Spec.Replicas == 0 {
		ai.Spec.Replicas = 1
	}

	if ai.Spec.SplunkConfiguration.Endpoint == "" && ai.Spec.SplunkConfiguration.SplunkCustomResourceRef.Name == "" {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "SplunkConfigMissing", "Splunk configuration is missing assuming no logging")
		return nil
	}

	var resolver splunkutils.SplunkSecretResolver

	switch ai.Spec.SplunkConfiguration.SecretSource {
	case aiv1.SecretSourceVault:
		resolver = &splunkutils.VaultFileResolver{} // Read from /vault/secrets/splunk
	default:
		resolver = &splunkutils.KubernetesSecretResolver{Client: r.Client} // Default
	}

	return splunkutils.ValidateAndEnrichSplunkConfig(
		ctx,
		r.Client,
		ai.Namespace,
		ai.Spec.ClusterDomain,
		&ai.Spec.SplunkConfiguration,
		resolver,
	)
}

func (r *SaiaReconciler) getAIPlatform(ctx context.Context, ref corev1.ObjectReference) (*aiv1.AIPlatform, error) {
	var aiPlatform aiv1.AIPlatform
	key := types.NamespacedName{
		Name:      ref.Name,
		Namespace: ref.Namespace,
	}
	if err := r.Client.Get(ctx, key, &aiPlatform); err != nil {
		return nil, err
	}
	return &aiPlatform, nil
}

func (r *SaiaReconciler) validateAIPlatformReady(ctx context.Context, aiPlatform *aiv1.AIPlatform) error {
	// Check if RayService infrastructure is ready (not the overall Ready condition to avoid circular dependency)
	if !common.IsConditionTrue(aiPlatform.Status.Conditions, "RayServiceStatusReady") {
		return fmt.Errorf("RayService is not ready")
	}

	// Verify RayService endpoint name is populated in status
	if aiPlatform.Status.RayServiceName == "" {
		return fmt.Errorf("RayServiceName not populated in AIPlatform status")
	}

	// Check RayService endpoint is reachable
	// TODO: Re-enable once we have a way to skip in test environments
	// if err := common.CheckRayHeadService(ctx, aiPlatform.Status.RayServiceName); err != nil {
	// 	return fmt.Errorf("RayService endpoint %s is not reachable: %w", aiPlatform.Status.RayServiceName, err)
	// }

	return nil
}

func (r *SaiaReconciler) validateVectorDatabaseReady(ctx context.Context, aiPlatform *aiv1.AIPlatform) error {
	// Check VectorDatabase status condition (not just the creation condition to ensure it's actually running)
	if !common.IsConditionTrue(aiPlatform.Status.Conditions, "WeaviateDatabaseStatusReady") {
		return fmt.Errorf("vector database is not ready")
	}

	// Verify VectorDB service name is populated in status
	if aiPlatform.Status.VectorDbServiceName == "" {
		return fmt.Errorf("VectorDbServiceName not populated in AIPlatform status")
	}

	// Check if VectorDB service endpoint is accessible
	// TODO: Re-enable once we have a way to skip in test environments
	// if err := common.CheckWeaviateService(ctx, aiPlatform.Status.VectorDbServiceName); err != nil {
	// 	return fmt.Errorf("vector database endpoint %s is not reachable: %w", aiPlatform.Status.VectorDbServiceName, err)
	// }

	return nil
}

// reconcileServiceAccount creates or reuses a ServiceAccount.
func (r *SaiaReconciler) reconcileServiceAccount(
	ctx context.Context,
	ai *aiv1.AIService,
) error {
	if ai.Spec.ServiceAccountName == "" {

		ai.Spec.ServiceAccountName = ai.Name + "-sa"
		if err := r.Update(ctx, ai); err != nil {
			return fmt.Errorf("updating SA name in spec: %w", err)
		}
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ai.Spec.ServiceAccountName,
				Namespace: ai.Namespace,
			},
		}
		if err := controllerutil.SetControllerReference(ai, sa, r.Scheme); err != nil {
			r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "ownerref on SA failed")
			return fmt.Errorf("ownerref on SA: %w", err)
		}
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
			return nil
		}); err != nil {
			r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "create/update SA failed")
			return fmt.Errorf("create/update SA: %w", err)
		}
	}
	return nil
}

// reconcileSAIAConfigMap manages the SAIA config ConfigMap for SPLUNK_ISSUERS.
func (r *SaiaReconciler) reconcileSAIAConfigMap(
	ctx context.Context,
	ai *aiv1.AIService,
) error {
	cmName := fmt.Sprintf("%s-saia-config", ai.Name)

	// Defaults for static keys (override in user-managed CM if desired).
	defaults := map[string]string{
		// previously hardcoded
		"SERVICE_NAME":                    "splunk_ai_assistant",
		"SERVICE_INTERNAL_NAME":           "SAIA",
		"SPLUNK_ISSUERS":                  "https://splunk-splunk-standalone-standalone-service:8089",
		"SPLUNK_AI_ASSISTANT_SERVICE_CMP": "true",
		"ENABLE_AUTHZ":                    "false", // FIXME remove when ready
		"FEATURE_CONFIG_FILE_LOCATION":    "/etc/config/features_config.yaml",
		"PLATFORM_VERSION":                "0.3.0",    // TODO make configurable
		"SAIA_API_VERSION":                "0.3.1",    // TODO make configurable
		"TELEMETRY_ENV":                   "NOTLOCAL", // TODO make configurable
		"LOG_LEVEL":                       "info",
	}

	found := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ai.Namespace}, found)
	if apierrors.IsNotFound(err) {
		// Create new with defaults
		return r.createOrUpdateConfigMap(ctx, cmName, defaults, ai)
	} else if err != nil {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec",
			fmt.Sprintf("failed to retrieve ConfigMap %q: %v", cmName, err))
		return fmt.Errorf("fetching SAIA ConfigMap %q: %w", cmName, err)
	}

	// Merge defaults for any missing keys, but don't override user-set values.
	if found.Data == nil {
		found.Data = map[string]string{}
	}
	needsUpdate := false
	for k, v := range defaults {
		if _, ok := found.Data[k]; !ok || found.Data[k] == "" {
			found.Data[k] = v
			needsUpdate = true
		}
	}
	if needsUpdate {
		if err := controllerutil.SetControllerReference(ai, found, r.Scheme); err != nil {
			return fmt.Errorf("ownerref on ConfigMap: %w", err)
		}
		return r.Update(ctx, found)
	}
	return nil
}

// reconcileCertificate manages cert-manager Certificate for mTLS.
func (r *SaiaReconciler) reconcileCertificate(
	ctx context.Context,
	ai *aiv1.AIService,
) error {
	if !ai.Spec.MTLS.Enabled || ai.Spec.MTLS.Termination != "operator" {
		return nil
	}
	cert := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ai.Name + "-tls",
			Namespace: ai.Namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: ai.Spec.MTLS.SecretName,
			IssuerRef:  ai.Spec.MTLS.IssuerRef,
			DNSNames:   ai.Spec.MTLS.DNSNames,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageServerAuth,
				certmanagerv1.UsageClientAuth,
			},
		},
	}
	if err := controllerutil.SetControllerReference(ai, cert, r.Scheme); err != nil {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "ownerref on Certificate failed")
		return fmt.Errorf("ownerref on Certificate: %w", err)
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, cert, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("create/update Certificate: %w", err)
	}
	// Wait until Certificate is Ready
	for _, cond := range cert.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady && cond.Status == cmmeta.ConditionTrue {
			return nil
		}
	}
	r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "Certificate is not Ready")
	return fmt.Errorf("waiting for Certificate %q to become Ready", cert.Name)
}

// reconcilePostInstallHook creates and watches the schema setup Job.
func (r *SaiaReconciler) reconcilePostInstallHook(
	ctx context.Context,
	ai *aiv1.AIService,
) error {
	hookImage := os.Getenv("RELATED_IMAGE_POST_INSTALL_HOOK")
	if ai.Spec.VectorDbUrl == "" {
		return nil
	}
	if ai.Status.SchemaJobId != "" {
		job := &batchv1.Job{}
		err := r.Get(
			ctx,
			client.ObjectKey{Namespace: ai.Namespace, Name: ai.Status.SchemaJobId},
			job,
		)
		if apierrors.IsNotFound(err) {
			ai.Status.SchemaJobId = ""
		} else if err != nil {
			r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "fetching Job failed")
			return fmt.Errorf("fetching Job: %w", err)
		} else {
			for _, c := range job.Status.Conditions {
				if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
					return nil
				}
				if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
					r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", fmt.Sprintf("Job %q failed", job.Name))
					return fmt.Errorf("job %q failed", job.Name)
				}
			}
			return fmt.Errorf("job %q is still running", job.Name)
		}
	}
	uri := fmt.Sprintf("http://%s:80", ai.Spec.VectorDbUrl)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ai.Name + "-vector-db-setup-posthook",
			Namespace: ai.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "vector-db-setup-container",
							Image:           hookImage,
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{Name: "VECTOR_DB_URL", Value: uri},
								{Name: "SPLUNK_AI_ASSISTANT_SERVICE_CMP", Value: "true"},
							},
						},
					},
					Tolerations: ai.Spec.Tolerations,
					Affinity:    &ai.Spec.Affinity,
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(ai, job, r.Scheme); err != nil {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "ownerref on Job failed")
		return fmt.Errorf("ownerref on Job: %w", err)
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, job, func() error { return nil }); err != nil {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "create/update Job failed")
		return fmt.Errorf("create/update Job: %w", err)
	}
	ai.Status.SchemaJobId = job.Name
	return fmt.Errorf("created Job %q, waiting for completion", job.Name)
}

// reconcileSAIADeployment ensures the main Deployment exists and is configured.
func (r *SaiaReconciler) reconcileSAIADeployment(
	ctx context.Context,
	ai *aiv1.AIService,
) error {
	optional := true
	volumes := []corev1.Volume{
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "features-config"},
					Optional:             &optional,
				},
			},
		},
	}

	ports := []corev1.ContainerPort{
		{Name: "http", ContainerPort: 8080},
		{Name: "metrics", ContainerPort: 8088},
	}
	mounts := []corev1.VolumeMount{
		{Name: "config-volume", MountPath: "/etc/config"},
	}

	// Base env: keep ONLY dynamic values here.
	env := []corev1.EnvVar{
		// Dynamic or runtime-derived values:
		{Name: "PLATFORM_URL", Value: ai.Spec.AIPlatformUrl},
		{Name: "VECTOR_DB_URL", Value: ai.Spec.VectorDbUrl},
		// SAIA uses /tasks subdirectory within its feature path
		{Name: "S3_BUCKET", Value:  ai.Spec.TaskVolume.Path},
	}

	// MinIO support: Add MinIO-specific environment variables if endpoint is configured
	if strings.HasPrefix(ai.Spec.TaskVolume.Path, "minio") && ai.Spec.TaskVolume.Endpoint != "" {
		env = append(env, corev1.EnvVar{Name: "MINIO_ENDPOINT_URL", Value: ai.Spec.TaskVolume.Endpoint})
	}

	// MinIO credentials: If secretRef is provided, add MINIO_ACCESS_KEY and MINIO_SECRET_KEY from secret
	if ai.Spec.TaskVolume.SecretRef != "" {
		env = append(env,
			corev1.EnvVar{
				Name: "MINIO_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: ai.Spec.TaskVolume.SecretRef},
						Key:                  "s3_access_key",
					},
				},
			},
			corev1.EnvVar{
				Name: "MINIO_SECRET_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: ai.Spec.TaskVolume.SecretRef},
						Key:                  "s3_secret_key",
					},
				},
			},
		)
	}

	// mTLS handling (dynamic)
	if ai.Spec.MTLS.Enabled && ai.Spec.MTLS.Termination == "operator" {
		volumes = append(volumes, corev1.Volume{
			Name: "tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: ai.Spec.MTLS.SecretName},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{Name: "tls", MountPath: "/etc/tls", ReadOnly: true})
		env = append(env,
			corev1.EnvVar{Name: "TLS_CERT_FILE", Value: "/etc/tls/tls.crt"},
			corev1.EnvVar{Name: "TLS_KEY_FILE", Value: "/etc/tls/tls.key"},
		)
		ports = append(ports, corev1.ContainerPort{Name: "https", ContainerPort: 8443})
	} else {
		env = append(env, corev1.EnvVar{Name: "TLS_DISABLED", Value: "true"})
	}

	// Import ALL static keys from the SAIA ConfigMap as env vars.
	envFrom := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-saia-config", ai.Name),
				},
				// Optional: set Optional: &truePtr if you prefer soft-fail
			},
		},
	}

	// Sort only the explicit envs (envFrom remains as-is)
	sort.Slice(env, func(i, j int) bool { return env[i].Name < env[j].Name })

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ai.Name + "-saia-deployment",
			Namespace: ai.Namespace,
			Labels: map[string]string{
				"app":       ai.Name,
				"component": ai.Name,
				"area":      "ml",
				"team":      "ml",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &ai.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": ai.Name, "component": ai.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": ai.Name, "component": ai.Name},
					Annotations: map[string]string{
						"prometheus.io/port":   "8088",
						"prometheus.io/path":   "/metrics",
						"prometheus.io/scheme": "http",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: ai.Spec.ServiceAccountName,
					Containers: []corev1.Container{{
						Name:            ai.Name,
						Image:           os.Getenv("RELATED_IMAGE_SAIA_API"),
						ImagePullPolicy: corev1.PullAlways,
						Ports:           ports,
						VolumeMounts:    mounts,
						Resources:       ai.Spec.Resources,
						Env:             env,
						EnvFrom:         envFrom, // <— bring in ALL static config keys
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromInt(8080)},
							},
							PeriodSeconds:    30,
							FailureThreshold: 5,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromInt(8080)},
							},
							PeriodSeconds:    30,
							FailureThreshold: 5,
						},
						StartupProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromInt(8080)},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       30,
							FailureThreshold:    5,
						},
					}},
					Volumes:     volumes,
					Affinity:    &ai.Spec.Affinity,
					Tolerations: ai.Spec.Tolerations,
				},
			},
		},
	}

	// Merge labels/annotations from AIService
	for k, v := range ai.Labels {
		deployment.ObjectMeta.Labels[k] = v
	}
	for k, v := range ai.Annotations {
		if k == "kubectl.kubernetes.io/last-applied-configuration" || k == "kubectl.kubernetes.io/restartedAt" {
			continue
		}
		deployment.ObjectMeta.Annotations[k] = v
	}

	if err := controllerutil.SetControllerReference(ai, deployment, r.Scheme); err != nil {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "ownerref on Deployment failed")
		return fmt.Errorf("ownerref on Deployment: %w", err)
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error { return nil }); err != nil {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "create/update Deployment failed")
		return fmt.Errorf("create/update Deployment: %w", err)
	}
	return nil
}

// reconcileSAIAService ensures the Service for SAIA is created/updated. // remove me
func (r *SaiaReconciler) reconcileSAIAService(
	ctx context.Context,
	ai *aiv1.AIService,
) error {
	ports := []corev1.ServicePort{
		{Name: "http", Port: 8080, TargetPort: intstr.FromInt(8080)},
		{Name: "metrics", Port: 8088, TargetPort: intstr.FromInt(8088)},
	}
	if ai.Spec.MTLS.Enabled && ai.Spec.MTLS.Termination == "operator" {
		ports = append(ports, corev1.ServicePort{
			Name: "https", Port: 8443, TargetPort: intstr.FromInt(8443),
		})
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ai.Name + "-saia-service",
			Namespace: ai.Namespace,
			Labels:    map[string]string{"app": ai.Name},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": ai.Name, "component": ai.Name},
			Ports:    ports,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	for k, v := range ai.Labels {
		svc.ObjectMeta.Labels[k] = v
	}
	for k, v := range ai.Annotations {
		if k == "kubectl.kubernetes.io/last-applied-configuration" {
			continue
		} // Ignore last-applied-configuration annotation
		if k == "kubectl.kubernetes.io/restartedAt" {
			continue
		} // Ignore restartedAt annotation
		svc.ObjectMeta.Annotations[k] = v
	}

	switch ai.Spec.ServiceTemplate.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		svc.Spec.Type = corev1.ServiceTypeLoadBalancer
	case corev1.ServiceTypeNodePort:
		svc.Spec.Type = corev1.ServiceTypeNodePort
		// If NodePort values are specified, set them
		for i, port := range svc.Spec.Ports {
			for _, tplPort := range ai.Spec.ServiceTemplate.Spec.Ports {
				if port.Name == tplPort.Name && tplPort.NodePort != 0 {
					svc.Spec.Ports[i].NodePort = tplPort.NodePort
				}
			}
		}
	default:
		svc.Spec.Type = corev1.ServiceTypeClusterIP
	}

	if err := controllerutil.SetControllerReference(ai, svc, r.Scheme); err != nil {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "ownerref on Service failed")
		return fmt.Errorf("ownerref on Service: %w", err)
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error { return nil }); err != nil {
		r.Recorder.Event(ai, corev1.EventTypeWarning, "InvalidSpec", "create/update Service failed")
		return fmt.Errorf("create/update Service: %w", err)
	}
	return nil
}

// reconcileServiceMonitor creates a Prometheus ServiceMonitor if metrics are enabled.
func (r *SaiaReconciler) reconcileServiceMonitor(
	ctx context.Context,
	ai *aiv1.AIService,
) error {
	if !ai.Spec.Metrics.Enabled {
		return nil
	}
	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{Name: ai.Name + "-metrics", Namespace: ai.Namespace},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": ai.Name, "component": ai.Name},
			},
			Endpoints: []monitoringv1.Endpoint{
				{Port: "metrics", Path: ai.Spec.Metrics.Path, Scheme: "http"},
			},
		},
	}
	if err := controllerutil.SetControllerReference(ai, sm, r.Scheme); err != nil {
		return err
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sm, func() error { return nil })
	return err
}

// createOrUpdateConfigMap is a helper to create or patch a ConfigMap // remove me
func (r *SaiaReconciler) createOrUpdateConfigMap(
	ctx context.Context,
	name string,
	data map[string]string,
	ai *aiv1.AIService,
) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ai.Namespace,
		},
		Data: data,
	}
	if err := controllerutil.SetControllerReference(ai, cm, r.Scheme); err != nil {
		return err
	}

	found := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: ai.Namespace}, found)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, cm)
	} else if err != nil {
		return err
	}

	if !reflect.DeepEqual(found.Data, data) {
		found.Data = data
		return r.Update(ctx, found)
	}
	return nil
}
