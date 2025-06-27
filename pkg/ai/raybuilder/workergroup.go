package raybuilder

import (
	"context"
	"fmt"
	"strconv"

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func GenerateWorkerGroups(ctx context.Context, models []ModelSpec, instanceMap map[string]InstanceMapping) []rayv1.WorkerGroupSpec {
	groupMap := make(map[WorkerGroupKey]int)
	modelMetadata := make(map[WorkerGroupKey]ModelSpec)

	for _, model := range models {
		if model.InstanceType != "" {
			instance, ok := instanceMap[model.InstanceType]
			if !ok {
				log.Log.Error(fmt.Errorf("unknown instance type %s for model %s", model.InstanceType, model.Name), "Warning")
				continue
			}
			model.GPUType = instance.GPUType
			if len(model.NodeSelector) == 0 {
				model.NodeSelector = instance.NodeSelector
			}
			if instance.NumGPUs > 0 && model.GPUsPerReplica > 0 {
				cpuPerGPU := instance.TotalCPU / instance.NumGPUs
				model.CPU = strconv.Itoa(cpuPerGPU * model.GPUsPerReplica)
				model.Memory = instance.TotalMemory
			}
		}

		if model.GPUsPerReplica > 0 && model.TensorParallelism != model.GPUsPerReplica {
			log.Log.Error(fmt.Errorf("model %s: tensorParallelism (%d) != GPUsPerReplica (%d)", model.Name, model.TensorParallelism, model.GPUsPerReplica), "Warning",
				"name", model.Name, "tensorParallelism", model.TensorParallelism, "GPUsPerReplicas", model.GPUsPerReplica)
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
			groupName = fmt.Sprintf("%s-gpu%d-tp%d", key.GPUType, key.GPUsPerReplica, key.TensorParallelism)
		}

		resources := corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{},
			Requests: corev1.ResourceList{},
		}
		if key.GPUsPerReplica > 0 {
			gpuQty := resource.MustParse(strconv.Itoa(key.GPUsPerReplica))
			resources.Limits["nvidia.com/gpu"] = gpuQty
			resources.Requests["nvidia.com/gpu"] = gpuQty
		}
		if key.Memory != "" {
			memQty := resource.MustParse(key.Memory)
			resources.Limits[corev1.ResourceMemory] = memQty
			resources.Requests[corev1.ResourceMemory] = memQty
		}
		if key.CPU != "" {
			cpuQty := resource.MustParse(key.CPU)
			resources.Limits[corev1.ResourceCPU] = cpuQty
			resources.Requests[corev1.ResourceCPU] = cpuQty
		}

		envs := []corev1.EnvVar{}
		if key.TensorParallelism > 0 {
			envs = append(envs, corev1.EnvVar{
				Name:  "TENSOR_PARALLELISM",
				Value: strconv.Itoa(key.TensorParallelism),
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
				"num-gpus": strconv.Itoa(key.GPUsPerReplica),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"ray.io/group": groupName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:      "ray",
						Resources: resources,
						Env:       envs,
					}},
					NodeSelector: model.NodeSelector,
					Tolerations:  tolerations,
					Affinity:     affinity,
				},
			},
		})
	}

	return workerGroups
}
