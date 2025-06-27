package raybuilder

import (
	"fmt"
	"strconv"
aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"
)

type ModelSpec struct {
	Name               string `json:"name"`
	GPUType            string `json:"gpuType,omitempty"`
	GPUsPerReplica     int    `json:"gpusPerReplica"`
	TensorParallelism  int    `json:"tensorParallelism"`
	Replicas           int    `json:"replicas"`
	CPU                string `json:"cpu,omitempty"`
	Memory             string `json:"memory,omitempty"`
}

type WorkerGroupKey struct {
	GPUType           string
	GPUsPerReplica    int
	TensorParallelism int
	CPU               string
	Memory            string
}

// GenerateWorkerGroups takes a slice of ModelSpec and generates a slice of rayv1.WorkerGroupSpec,
// grouping models by their resource requirements (GPU type, GPUs per replica, tensor parallelism, CPU, and memory).
// For each unique group, it aggregates the total number of replicas and constructs a corresponding WorkerGroupSpec
// with appropriate resource requests, environment variables, and node selectors. Models with mismatched
// tensor parallelism and GPUs per replica are skipped with a warning.
//
// Parameters:
//   - models: A slice of ModelSpec representing the models to be deployed.
//
// Returns:
//   - A slice of rayv1.WorkerGroupSpec, each representing a group of workers with shared resource requirements.
func GenerateWorkerGroups(models []ModelSpec, spec aiApi.AIPlatformSpec) []rayv1.WorkerGroupSpec {
	groupMap := make(map[WorkerGroupKey]int)

	for _, model := range models {
		// Validate tensor parallelism
		if model.GPUsPerReplica > 0 && model.TensorParallelism != model.GPUsPerReplica {
			fmt.Printf("Warning: model %s: tensorParallelism (%d) does not match GPUsPerReplica (%d)\n",
				model.Name, model.TensorParallelism, model.GPUsPerReplica)
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
	}

	var workerGroups []rayv1.WorkerGroupSpec

	for key, totalReplicas := range groupMap {
		groupName := "cpu-group"
		if key.GPUsPerReplica > 0 {
			groupName = fmt.Sprintf("%s-gpu%d-tp%d", key.GPUType, key.GPUsPerReplica, key.TensorParallelism)
		}

		// Build resources conditionally
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

		var nodeSelector map[string]string
		if key.GPUsPerReplica > 0 {
			nodeSelector = spec.GPUSchedulingSpec.NodeSelector
		} else {
			nodeSelector = spec.CPUSchedulingSpec.NodeSelector
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
					Containers: []corev1.Container{
						{
							Name:      "ray",
							Resources: resources,
							Env:       envs,
						},
					},
					NodeSelector: nodeSelector,
				},
			},
		})
	}

	return workerGroups
}
