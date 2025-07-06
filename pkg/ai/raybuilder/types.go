package raybuilder

import (
	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// types.go
type ServeConfig struct {
	ProxyLocation string        `json:"proxy_location,omitempty"`
	HTTPOptions   HTTPOptions   `json:"http_options"`
	GRPCOptions   GRPCOptions   `json:"grpc_options"`
	LoggingConfig LoggingConfig `json:"logging_config"`
	Applications  []Application `json:"applications"`
}

type HTTPOptions struct {
	Host              string `json:"host,omitempty"`
	Port              int    `json:"port,omitempty"`
	RequestTimeoutS   int    `json:"request_timeout_s,omitempty"`
	KeepAliveTimeoutS int    `json:"keep_alive_timeout_s"`
}

type GRPCOptions struct {
	Port                  int      `json:"port,omitempty"`
	GRPCServicerFunctions []string `json:"grpc_servicer_functions,omitempty"`
	RequestTimeoutS       int      `json:"request_timeout_s,omitempty"`
}

type LoggingConfig struct {
	LogLevel        string `json:"log_level,omitempty"`
	LogsDir         string `json:"logs_dir,omitempty"`
	Encoding        string `json:"encoding,omitempty"`
	EnableAccessLog bool   `json:"enable_access_log,omitempty"`
}

// Application mirrors one rayService application
type Application struct {
	Name        string      `yaml:"name"`
	ImportPath  string      `yaml:"import_path,omitempty"` // Optional, used for Python imports
	RoutePrefix string      `yaml:"route_prefix,omitempty"`
	RuntimeEnv  *RuntimeEnv `yaml:"runtime_env,omitempty"`
	Args        struct {
		DeploymentConfigs map[string]struct {
			GPUTypeOptionsOverride map[string]struct {
				AutoscalingConfig struct {
					MinReplicas int `yaml:"min_replicas"`
					MaxReplicas int `yaml:"max_replicas"`
				} `yaml:"autoscaling_config"`
				RayActorOptions struct {
					NumGPUs float64 `yaml:"num_gpus"` // Can be fractional like 0.01
					NumCPUs float64 `yaml:"num_cpus"` // Optional
				} `yaml:"ray_actor_options"`
			} `yaml:"gpu_type_options_override"`
		} `yaml:"deployment_configs"`
		ModelDefinition struct {
			ModelID                    string `yaml:"model_id"`
			GPUTypeModelConfigOverride map[string]struct {
				EngineArgs struct {
					TensorParallelSize int `yaml:"tensor_parallel_size"`
				} `yaml:"engine_args"`
			} `yaml:"gpu_type_model_config_override"`
		} `yaml:"model_definition"`
	} `yaml:"args"`
}

// RuntimeEnv mirrors the runtime_env field in rayService
type RuntimeEnv struct {
	WorkingDir string            `json:"working_dir,omitempty"`
	EnvVars    map[string]string `json:"env_vars,omitempty"`
	Pip        []string          `json:"pip,omitempty"`
}

// Config is just a thin wrapper around rayService.applications
type Config struct {
	RayService ServeConfig `json:"rayService"`
}

type RayWorkerGroupSpec struct {
	GroupName      string                      `json:"groupName"`
	Replicas       int32                       `json:"replicas"`
	RayStartParams map[string]string           `json:"rayStartParams,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources"`
	Scheduling     aiApi.SchedulingSpec        `json:"scheduling"`
}

type ModelSpec struct {
	Name              string              `json:"name"`
	InstanceType      string              `json:"instanceType,omitempty"`
	GPUType           string              `json:"gpuType,omitempty"`
	GPUsPerReplica    float64             `json:"gpusPerReplica"`
	TensorParallelism float64             `json:"tensorParallelism"`
	Replicas          int                 `json:"replicas"`
	CPU               float64             `json:"cpu,omitempty"`
	Memory            string              `json:"memory,omitempty"`
	NodeSelector      map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations       []corev1.Toleration `json:"tolerations,omitempty"`
	Affinity          *corev1.Affinity    `json:"affinity,omitempty"`
}

type InstanceDetails struct {
	GPUType         string            `yaml:"gpuType"`
	AcceleratorType string            `yaml:"acceleratorType,omitempty"`
	GPUs            float64           `yaml:"gpus"`
	Memory          string            `yaml:"memory"`
	VCPUs           float64           `yaml:"vcpus"`
	NodeSelector    map[string]string `yaml:"nodeSelector,omitempty"`
}

type WorkerGroupKey struct {
	GPUType           string
	GPUsPerReplica    float64
	TensorParallelism float64
	CPU               float64
	Memory            string
}

type InstanceInfo struct {
	GPUType string `json:"gpuType"`
	GPUs    int    `json:"gpus"`
	Memory  int    `json:"memory"`
	VCPUs   int    `json:"vcpus"`
}

type RayServiceSpec struct {
	RayService ApplicationsYaml `yaml:"rayService"`
}

type ApplicationsYaml struct {
	Applications []ApplicationEntry `yaml:"applications"`
}

type ApplicationEntry struct {
	Name string `yaml:"name"`
	Args struct {
		DeploymentConfigs map[string]DeploymentConfig `yaml:"deployment_configs"`
	} `yaml:"args"`
}

type DeploymentConfig struct {
	GPUTypeOptionsOverride map[string]struct {
		AutoscalingConfig struct {
			MinReplicas int `yaml:"min_replicas"`
			MaxReplicas int `yaml:"max_replicas"`
		} `yaml:"autoscaling_config"`
		RayActorOptions struct {
			NumGPUs float64 `yaml:"num_gpus"`
			NumCPUs float64 `yaml:"num_cpus"`
			Memory  string  `yaml:"memory,omitempty"` // Optional, e.g., "8Gi"
		} `yaml:"ray_actor_options"`
	} `yaml:"gpu_type_options_override"`
	Options struct {
		AutoscalingConfig struct {
			MinReplicas int `yaml:"min_replicas"`
			MaxReplicas int `yaml:"max_replicas"`
		} `yaml:"autoscaling_config"`
		RayActorOptions struct {
			NumGPUs float64 `yaml:"num_gpus"`
			NumCPUs float64 `yaml:"num_cpus"`
			Memory  string  `yaml:"memory,omitempty"` // Optional, e.g., "8Gi"
		} `yaml:"ray_actor_options"`
	} `yaml:"options"`
}
