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
	RoutePrefix   string        `json:"route_prefix,omitempty"`
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
	GPUsPerReplica    int
	TensorParallelism int
	CPU               int
	Memory            string
}

type InstanceInfo struct {
	GPUType string  `json:"gpuType"`
	GPUs    float64 `json:"gpus"`
	Memory  string  `json:"memory"`
	VCPUs   float64 `json:"vcpus"`
}

type RayServiceSpec struct {
	RayService RayService `yaml:"rayService"`
}

type RayService struct {
	Applications []Application `yaml:"applications"`
}

type Application struct {
	Name          string         `yaml:"name"`
	Args          *Args          `yaml:"args,omitempty"`
	RuntimeEnv    *RuntimeEnv    `yaml:"runtime_env,omitempty"`
	LLMDeployment *LLMDeployment `yaml:"LLMDeployment,omitempty"`
	WorkingDir    string         `yaml:"working_dir,omitempty"`
	ImportPath    string         `yaml:"import_path,omitempty"`
	RoutePrefix   string         `yaml:"route_prefix,omitempty"`
}

type Args struct {
	DeploymentType             string                      `yaml:"deployment_type,omitempty"`
	CustomDeploymentImportPath string                      `yaml:"custom_deployment_import_path,omitempty"`
	DeploymentConfigs          map[string]DeploymentConfig `yaml:"deployment_configs,omitempty"`
	ModelDefinition            *ModelDefinition            `yaml:"model_definition,omitempty"`
	TokenizerDefinition        *TokenizerDefinition        `yaml:"tokenizer_definition,omitempty"`
}

type DeploymentConfig struct {
	Options                *DeploymentOptions           `yaml:"options,omitempty"`
	GPUTypeOptionsOverride map[string]DeploymentOptions `yaml:"gpu_type_options_override,omitempty"`
	EnvOptionsOverride     map[string]DeploymentOptions `yaml:"env_options_override,omitempty"`
}

type DeploymentOptions struct {
	AutoscalingConfig  *AutoscalingConfig `yaml:"autoscaling_config,omitempty"`
	RayActorOptions    *RayActorOptions   `yaml:"ray_actor_options,omitempty"`
	MaxOngoingRequests *int               `yaml:"max_ongoing_requests,omitempty"`
	Memory             string             `yaml:"memory,omitempty"`
}

type AutoscalingConfig struct {
	MaxReplicas           *int `yaml:"max_replicas,omitempty"`
	MinReplicas           *int `yaml:"min_replicas,omitempty"`
	TargetOngoingRequests *int `yaml:"target_ongoing_requests,omitempty"`
}

type RayActorOptions struct {
	NumGPUs float64 `yaml:"num_gpus,omitempty"`
	NumCPUs float64 `yaml:"num_cpus,omitempty"`
	Memory  string  `yaml:"memory,omitempty"`
}

type ModelDefinition struct {
	ModelID                    string                            `yaml:"model_id,omitempty"`
	ModelType                  string                            `yaml:"model_type,omitempty"`
	CustomModelImportPath      string                            `yaml:"custom_model_import_path,omitempty"`
	ModelLoader                *ModelLoader                      `yaml:"model_loader,omitempty"`
	GPUTypeModelConfigOverride map[string]GPUModelConfigOverride `yaml:"gpu_type_model_config_override,omitempty"`
}

type GPUModelConfigOverride struct {
	EngineArgs *EngineArgs `yaml:"engine_args,omitempty"`
}

type EngineArgs struct {
	DType                string  `yaml:"dtype,omitempty"`
	MaxModelLen          int     `yaml:"max_model_len,omitempty"`
	TensorParallelSize   int     `yaml:"tensor_parallel_size,omitempty"`
	GPUMemoryUtilization float64 `yaml:"gpu_memory_utilization,omitempty"`
}

type ModelLoader struct {
	RemoteArtifact *RemoteArtifact `yaml:"remote_artifact,omitempty"`
}

type RemoteArtifact struct {
	KeyPrefix     string   `yaml:"key_prefix,omitempty"`
	ArtifactsList []string `yaml:"artifacts_list,omitempty"`
}

type TokenizerDefinition struct {
	ModelID     string       `yaml:"model_id,omitempty"`
	ModelLoader *ModelLoader `yaml:"model_loader,omitempty"`
}

type RuntimeEnv struct {
	Pip        []string          `yaml:"pip,omitempty"`
	EnvVars    map[string]string `yaml:"env_vars,omitempty"`
	WorkingDir string            `yaml:"working_dir,omitempty"`
}

type LLMDeployment struct {
	GPUTypeOptionsOverride map[string]DeploymentOptions `yaml:"gpu_type_options_override,omitempty"`
}
