/*
File: controllers/raybuilder/builder.go
*/
package raybuilder

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/splunk/splunk-ai-operator/pkg/ai/sidecars"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	rbacv1 "k8s.io/api/rbac/v1"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Builder encapsulates RayService generation logic.
type Builder struct {
	ai *aiApi.AIPlatform
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// New returns a new Builder for the given AIPlatform instance.
func New(ai *aiApi.AIPlatform, client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) *Builder {
	return &Builder{
		ai:       ai,
		Client:   client,
		Scheme:   scheme,
		Recorder: recorder,
	}
}

// --- 7️⃣ ReconcileRayService: build & create/update the RayService CR ---
func (b *Builder) ReconcileRayService(ctx context.Context, p *aiApi.AIPlatform) error {
	logger := log.FromContext(ctx) // Define logger
	rs, err := b.Build(ctx)
	if err != nil {
		logger.Error(err, "failed to build RayService")
		return fmt.Errorf("failed to build RayService: %w", err)
	}

	// Fetch the ServeConfigMap
	serveConfigMap := &corev1.ConfigMap{}
	serveConfigMapKey := types.NamespacedName{Namespace: p.Namespace, Name: p.Name + "-serveconfig"}
	if err := b.Client.Get(ctx, serveConfigMapKey, serveConfigMap); err != nil {
		return err
	}

	annotations, labels := buildHeadAnnotationsAndLabels(p)
	rayService := &rayv1.RayService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
	}
	err = b.Client.Get(ctx, types.NamespacedName{Namespace: p.Namespace, Name: p.Name}, rayService)
	if errors.IsNotFound(err) {
		rayService = &rayv1.RayService{
			ObjectMeta: metav1.ObjectMeta{
				Name:        p.Name,
				Namespace:   p.Namespace,
				Annotations: annotations,
				Labels:      labels,
			},
		}
	}

	// Add ServeConfigMap to RayService annotations FIXME
	if serveConfig, exists := serveConfigMap.Data["serveconfig.yaml"]; exists {
		rs.Spec.ServeConfigV2 = serveConfig
	} else {
		logger.Error(fmt.Errorf("serveconfig.yaml not found"), "ServeConfigMap is missing serveconfig.yaml key")
		return fmt.Errorf("serveconfig.yaml not found in ConfigMap %s", serveConfigMapKey.Name)
	}

	rayService.Spec = rs.Spec
	key := types.NamespacedName{Namespace: rayService.Namespace, Name: rayService.Name}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current rayv1.RayService
		if err := b.Client.Get(ctx, key, &current); err != nil {
			if errors.IsNotFound(err) {
				controllerutil.SetOwnerReference(p, rayService, b.Scheme)
				return b.Client.Create(ctx, rayService)
			}
			b.Recorder.Eventf(p, corev1.EventTypeWarning, "ReconcileFailed", "Failed to reconcile RayService %v", err)
			return err
		}

		// mutate current.Spec to match desired svc.Spec
		current.Spec = rs.Spec
		// now try update
		controllerutil.SetOwnerReference(p, &current, b.Scheme)
		return b.Client.Update(ctx, &current)
	})
}

// FIXME work with @shang to find if rayserve support this internally
func (b *Builder) ReconcileRayAutoscalerRBAC(ctx context.Context, p *aiApi.AIPlatform) error {
	logger := log.FromContext(ctx)
	saName := p.Spec.ServiceAccountName
	if saName == "" {
		logger.Info("No ServiceAccount specified for Ray head group, skipping RBAC reconciliation")
		return nil
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ray-autoscaler",
			Namespace: p.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"ray.io"},
				Resources: []string{"rayclusters", "rayservices", "rayjobs"},
				Verbs:     []string{"get", "list", "watch", "patch", "update", "delete"},
			},
		},
	}

	if err := b.Client.Create(ctx, role); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	controllerutil.SetOwnerReference(p, role, b.Scheme)

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ray-autoscaler-binding-" + p.Namespace + "-" + saName,
			Namespace: p.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: p.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "ray-autoscaler",
		},
	}

	if err := b.Client.Create(ctx, roleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	controllerutil.SetOwnerReference(p, roleBinding, b.Scheme)
	return nil
}

func (b *Builder) ReconcileRayServiceStatus(
	ctx context.Context,
	p *aiApi.AIPlatform,
) error {
	// 1️⃣ fetch the up-to-date RayService
	rs := &rayv1.RayService{}
	key := types.NamespacedName{Namespace: p.Namespace, Name: p.Name}
	if err := b.Client.Get(ctx, key, rs); err != nil {
		return err
	}

	// 2️⃣ mirror its status into your CR
	p.Status.RayServiceStatus = rs.Status.ServiceStatus

	// Add Ray head service name to status
	p.Status.RayServiceName = fmt.Sprintf("%s-head-svc", p.Name)

	// 3️⃣ set a Condition based on whatever flag you like—e.g. the top-level Ready
	ready := metav1.ConditionFalse
	reason := "RayServiceStatus"
	msg := "ray service is not yet ready"
	if rs.Status.ServiceStatus == rayv1.Running {
		ready = metav1.ConditionTrue
		reason = "RayServiceReady"
		msg = "ray service is running"
	}

	cond := metav1.Condition{
		Type:               "RayServiceReady",
		Status:             ready,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	}
	meta.SetStatusCondition(&p.Status.Conditions, cond)

	return nil
}

// Build constructs a RayService resource based on the AI CR.
func (b *Builder) Build(ctx context.Context) (*rayv1.RayService, error) {
	clusterConfig, err := b.buildClusterConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to build cluster config: %w", err)
	}
	rs := &rayv1.RayService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.ai.Name,
			Namespace:   b.ai.Namespace,
			Annotations: b.ai.Annotations,
			Labels:      b.ai.Labels,
		},
		Spec: rayv1.RayServiceSpec{
			RayClusterSpec: clusterConfig,
		},
	}
	return rs, nil
}

func (b *Builder) buildClusterConfig(ctx context.Context) (rayv1.RayClusterSpec, error) {
	annotations, labels := buildHeadAnnotationsAndLabels(b.ai)
	head := rayv1.HeadGroupSpec{
		RayStartParams: map[string]string{
			"dashboard-host": "0.0.0.0",
			"num-cpus":       "0",
		},
		HeadService: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        b.ai.Name + "-head-svc",
				Namespace:   b.ai.Namespace,
				Annotations: annotations,
				Labels:      labels,
			},
		},
		Template: b.makeHeadTemplate(),
	}

	head.Template.ObjectMeta.Annotations = annotations
	head.Template.ObjectMeta.Labels = labels

	yamlData, err := ReadApplicationsYAMLFromConfigMap(ctx, b.Client, b.ai.Name+"-applications", b.ai.Namespace)
	if err != nil {
		return rayv1.RayClusterSpec{}, err
	}
	modelSpecs, err := BuildModelSpecsFromApplicationsYAML(yamlData)
	if err != nil {
		return rayv1.RayClusterSpec{}, fmt.Errorf("failed to build model specs from applications YAML: %w", err)
	}
	instanceMap, err := ReadInstanceMapFromConfigMap(ctx, b.Client, b.ai.Name+"-instances", b.ai.Namespace)
	if err != nil {
		return rayv1.RayClusterSpec{}, fmt.Errorf("failed to read instance map from config map: %w", err)
	}
	workerGroups := b.GenerateWorkerGroups(ctx, b.Client, modelSpecs, instanceMap)

	return rayv1.RayClusterSpec{
		RayVersion:              os.Getenv("RAY_VERSION"),
		EnableInTreeAutoscaling: boolPtr(true),
		HeadGroupSpec:           head,
		WorkerGroupSpecs:        workerGroups,
	}, nil
}

func (b *Builder) makeHeadTemplate() corev1.PodTemplateSpec {
	spec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:  "ray-head",
			Image: SetImageRegistry("RELATED_IMAGE_RAY_HEAD", b.ai.Spec.Images.RayHeadGroupImage),
			Args: []string{
				"ulimit -n 65536; echo head; $KUBERAY_GEN_RAY_START_CMD",
			},
			Command: []string{
				"/bin/bash",
				"-lc",
				"--",
			},
			Env: []corev1.EnvVar{
				{Name: "DEFAULT_ACCELERATOR_TYPE", Value: b.ai.Spec.DefaultAcceleratorType},
				{Name: "CLUSTER_NAME", Value: os.Getenv("CLUSTER_NAME")},
			},
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/bin/sh",
							"-c",
							"ray stop",
						},
					},
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 6379,
					Name:          "gcs-server",
					Protocol:      corev1.ProtocolTCP,
				},
				{
					ContainerPort: 8265,
					Name:          "dashboard",
					Protocol:      corev1.ProtocolTCP,
				},
				{
					ContainerPort: 10001,
					Name:          "client",
					Protocol:      corev1.ProtocolTCP,
				},
				{
					ContainerPort: 8000,
					Name:          "serve",
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse("1"),
					corev1.ResourceMemory:           resource.MustParse("2Gi"),
					corev1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
					"nvidia.com/gpu":                resource.MustParse("0"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse("4"),
					corev1.ResourceMemory:           resource.MustParse("8Gi"),
					corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
					"nvidia.com/gpu":                resource.MustParse("0"),
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/tmp/ray",
					Name:      "ray-logs",
				},
			},
		}},
	}

	spec.NodeSelector = b.ai.Spec.GPUSchedulingSpec.NodeSelector
	spec.Tolerations = b.ai.Spec.GPUSchedulingSpec.Tolerations
	spec.Affinity = b.ai.Spec.GPUSchedulingSpec.Affinity
	spec.ServiceAccountName = b.ai.Spec.ServiceAccountName
	// FIXME need to find better way to add sidecars
	sidecars := sidecars.New(b.Client, b.Scheme, b.Recorder, b.ai)
	sidecars.AddFluentBitSidecar(&spec)
	found := false
	for _, vol := range spec.Volumes {
		if vol.Name == "ray-logs" {
			found = true
			break
		}
	}
	if !found {
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name: "ray-logs",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	return corev1.PodTemplateSpec{Spec: spec}
}

func (b *Builder) makeWorkerTemplate(cfg aiApi.GPUConfig) corev1.PodTemplateSpec {
	rayCommand := fmt.Sprintf(`echo %s worker;
        ulimit -n 65536;
    	export PATH="/home/ray/anaconda3/bin:$PATH";
        KUBERAY_GEN_RAY_START_CMD=$(echo $KUBERAY_GEN_RAY_START_CMD | sed -e 's/"{/{/g' -e 's/}"/}/g' -e 's/\\\"/"/g');
        $KUBERAY_GEN_RAY_START_CMD;`, cfg.Tier)
	spec := corev1.PodSpec{
		ServiceAccountName: b.ai.Spec.ServiceAccountName,
		Containers: []corev1.Container{{
			Name:            "ray-worker",
			Image:           SetImageRegistry("RELATED_IMAGE_RAY_WORKER", b.ai.Spec.Images.RayWorkerGroupImage),
			ImagePullPolicy: corev1.PullAlways,
			Command: []string{
				"/bin/bash",
				"-lc",
				"--",
			},
			Args: []string{
				rayCommand,
			},
			Env: []corev1.EnvVar{
				{Name: "DEFAULT_ACCELERATOR_TYPE", Value: b.ai.Spec.DefaultAcceleratorType},
				{Name: "RAY_HEAD_SERVICE_HOST", Value: fmt.Sprintf("%s.%s.svc.%s", b.ai.Name+"-head-svc", b.ai.Namespace, os.Getenv("CLUSTER_DOMAIN"))},
				{Name: "SERVICE_NAME", Value: b.ai.Name},
				{Name: "SERVICE_INTERNAL_NAME", Value: b.ai.Name},
				{Name: "USE_SYSTEM_PERMISSIONS", Value: "true"},
				{Name: "GPG_PUBLICKEY_PATH", Value: "kv-splunk/al-platform.ray-worker-sa/gpgkey"}, // FIXME
				{Name: "GPU_TYPE", Value: "L40S"},                                                 // FIXME
				{Name: "NVIDIA_VISIBLE_DEVICES", Value: "all"},
			},
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/bin/sh",
							"-c",
							"ray stop",
						},
					},
				},
			},
			Resources: cfg.Resources,
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/tmp/ray",
					Name:      "ray-logs",
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 8080,
					Name:          "metrics",
					Protocol:      corev1.ProtocolTCP,
				},
			},
		}},
	}

	// apply scheduling
	spec.NodeSelector = b.ai.Spec.GPUSchedulingSpec.NodeSelector
	spec.Tolerations = b.ai.Spec.GPUSchedulingSpec.Tolerations
	spec.Affinity = b.ai.Spec.GPUSchedulingSpec.Affinity

	found := false
	for _, vol := range spec.Volumes {
		if vol.Name == "ray-logs" {
			found = true
			break
		}
	}

	if !found {
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name: "ray-logs",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	// FIXME need to find better way to add sidecars
	sidecars := sidecars.New(b.Client, b.Scheme, b.Recorder, b.ai)
	sidecars.AddFluentBitSidecar(&spec)
	return corev1.PodTemplateSpec{Spec: spec}
}

func SetImageRegistry(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func buildWorkerAnnotationsAndLabels(aiPlatform *aiApi.AIPlatform, cfg aiApi.GPUConfig) (map[string]string, map[string]string) {
	annotations := make(map[string]string)
	labels := make(map[string]string)

	// Example: propagate tier and GPU type as labels/annotations
	annotations["gpu-tier"] = cfg.Tier
	labels["gpu-tier"] = cfg.Tier
	if aiPlatform.Annotations != nil {
		for k, v := range aiPlatform.Annotations {
			if strings.Contains(k, "last-applied-configuration") {
				continue
			}
			annotations[k] = v
		}
	}
	if aiPlatform.Labels != nil {
		for k, v := range aiPlatform.Labels {
			if strings.Contains(k, "last-applied-configuration") {
				continue
			}
			labels[k] = v
		}
	}
	annotations["prometheus.io/path"] = "/metrics"
	annotations["prometheus.io/port"] = "8080"
	annotations["prometheus.io/scheme"] = "http"
	annotations["ray.io/overwrite-container-cmd"] = "true"
	if aiPlatform.Spec.Sidecars.Otel {
		annotations["sidecar.opentelemetry.io/inject"] = fmt.Sprintf("%s-otel-coll", aiPlatform.Name)
		annotations["sidecar.opentelemetry.io/auto-instrument"] = "true"
	}

	// Add any additional logic as needed

	return annotations, labels
}

func buildHeadAnnotationsAndLabels(aiPlatform *aiApi.AIPlatform) (map[string]string, map[string]string) {
	annotations := make(map[string]string)
	labels := make(map[string]string)

	// Example: propagate tier and GPU type as labels/annotations
	if aiPlatform.Annotations != nil {
		for k, v := range aiPlatform.Annotations {
			if strings.Contains(k, "last-applied-configuration") {
				continue
			}
			annotations[k] = v
		}
	}
	if aiPlatform.Labels != nil {
		for k, v := range aiPlatform.Labels {
			if strings.Contains(k, "last-applied-configuration") {
				continue
			}
			labels[k] = v
		}
	}
	annotations["prometheus.io/path"] = "/metrics"
	annotations["prometheus.io/port"] = "8080"
	annotations["prometheus.io/scheme"] = "http"
	annotations["ray.io/overwrite-container-cmd"] = "true"

	if aiPlatform.Spec.Sidecars.Otel {
		annotations["sidecar.opentelemetry.io/inject"] = fmt.Sprintf("%s-otel-coll", aiPlatform.Name)
		annotations["sidecar.opentelemetry.io/auto-instrument"] = "true"
	}

	return annotations, labels
}

// boolPtr returns a pointer to the given boolean value.
func boolPtr(b bool) *bool {
	return &b
}

func (b *Builder) GenerateWorkerGroups(ctx context.Context, k8sClient client.Client, models []ModelSpec, instanceMap InstanceMap) []rayv1.WorkerGroupSpec {
	logger := log.FromContext(ctx)
	groupMap := make(map[WorkerGroupKey]int)
	modelMetadata := make(map[WorkerGroupKey]ModelSpec)

	for _, model := range models {
		// If instance type is not specified, try to find one that satisfies model requirements
		if model.InstanceType == "" {
			found := false
			for _, instances := range instanceMap {
				for instance, info := range instances {
					// compare against model requirements
					modelCPUQty := resource.MustParse(fmt.Sprintf("%v", model.CPU))
					modelMemQty := resource.MustParse(model.Memory)
					requiredGPUs := model.GPUsPerReplica

					instanceCPUQty := resource.MustParse(fmt.Sprintf("%v", info.VCPUs))
					instanceMemQty := resource.MustParse(info.Memory)

					if instanceCPUQty.Cmp(modelCPUQty) >= 0 &&
						instanceMemQty.Cmp(modelMemQty) >= 0 &&
						info.GPUs >= requiredGPUs {
						model.InstanceType = instance
						if model.GPUType == "" {
							model.GPUType = info.GPUType
						}
						found = true
						break
					}
				}
				if found {
					break
				}
			}

			// fallback to default instance
			if !found {
				logger.Info("No matching instance found for model, using fallback instance", "model", model.Name)

				// Detect cloud provider from label or configuration (assume model.Provider is set)
				provider, err := detectProvider(k8sClient, ctx)
				if err != nil {
					logger.Error(err, "Failed to detect provider")
					continue
				}
				fallbackInfo, ok := instanceMap[provider]["default-gpu-instance"]
				if !ok {
					logger.Error(fmt.Errorf("fallback instance not found for provider %s", provider), "Fallback failed")
					continue
				}

				model.InstanceType = "default-gpu-instance"
				model.GPUType = fallbackInfo.GPUType
				model.Memory = fallbackInfo.Memory
				model.CPU = fallbackInfo.VCPUs
				model.GPUsPerReplica = fallbackInfo.GPUs
			}
		}

		if model.GPUsPerReplica > 0 && model.TensorParallelism != model.GPUsPerReplica {
			logger.Error(fmt.Errorf("tensorParallelism (%f) != GPUsPerReplica (%f)", model.TensorParallelism, model.GPUsPerReplica), "Invalid model config", "model", model.Name)
			continue
		}

		key := WorkerGroupKey{
			GPUType:           model.GPUType,
			GPUsPerReplica:    model.GPUsPerReplica,
			TensorParallelism: model.TensorParallelism,
			CPU:               model.CPU,
			Memory:            model.Memory,
		}
		groupMap[key] += model.Replicas
		modelMetadata[key] = model
	}

	var workerGroups []rayv1.WorkerGroupSpec

	for key, totalReplicas := range groupMap {
		model := modelMetadata[key]
		groupName := "cpu-group"
		if key.GPUsPerReplica > 0 {
			groupName = fmt.Sprintf("%s-gpu%d-tp%d", key.GPUType, int(key.GPUsPerReplica), int(key.TensorParallelism))
		}

		resources := corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{},
			Requests: corev1.ResourceList{},
		}
		if key.GPUsPerReplica > 0 {
			gpuQty := resource.MustParse(strconv.Itoa(int(key.GPUsPerReplica)))
			resources.Limits["nvidia.com/gpu"] = gpuQty
			resources.Requests["nvidia.com/gpu"] = gpuQty
		}
		if key.Memory != "" {
			memQty := resource.MustParse(key.Memory)
			resources.Limits[corev1.ResourceMemory] = memQty
			resources.Requests[corev1.ResourceMemory] = memQty
		}
		if key.CPU != 0 {
			cpuQty := resource.MustParse(fmt.Sprintf("%v", model.CPU))
			resources.Limits[corev1.ResourceCPU] = cpuQty
			resources.Requests[corev1.ResourceCPU] = cpuQty
		}

		gpuConfig := aiApi.GPUConfig{
			Tier:      groupName,
			Resources: resources,
		}
		podSpec := b.makeWorkerTemplate(gpuConfig)

		envs := []corev1.EnvVar{}
		if key.TensorParallelism > 0 {
			envs = append(envs, corev1.EnvVar{
				Name:  "TENSOR_PARALLELISM",
				Value: strconv.Itoa(int(key.TensorParallelism)),
			})
		}

		tolerations := model.Tolerations
		if key.GPUsPerReplica > 0 && len(tolerations) == 0 {
			tolerations = []corev1.Toleration{{
				Key:      "nvidia.com/gpu",
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			}}
		}

		affinity := model.Affinity
		if affinity == nil {
			affinity = &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
						Weight: 100,
						PodAffinityTerm: corev1.PodAffinityTerm{
							TopologyKey: "topology.kubernetes.io/zone",
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"ray.io/group": groupName,
								},
							},
						},
					}},
				},
			}
		}

		workerGroups = append(workerGroups, rayv1.WorkerGroupSpec{
			GroupName:   groupName,
			Replicas:    utilpointer.Int32(int32(totalReplicas)),
			MinReplicas: utilpointer.Int32(int32(totalReplicas)),
			MaxReplicas: utilpointer.Int32(int32(totalReplicas)),
			RayStartParams: map[string]string{
				"num-gpus": strconv.Itoa(int(key.GPUsPerReplica)),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"ray.io/group": groupName,
					},
				},
				Spec: corev1.PodSpec{
					Containers:   podSpec.Spec.Containers,
					NodeSelector: model.NodeSelector,
					Tolerations:  tolerations,
					Affinity:     affinity,
					Volumes:      podSpec.Spec.Volumes,
				},
			},
		})
	}

	return workerGroups
}
