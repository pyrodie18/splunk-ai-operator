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
	Name        string                 `json:"name"`
	ImportPath  string                 `json:"import_path,omitempty"`
	RoutePrefix string                 `json:"route_prefix,omitempty"`
	Args        map[string]interface{} `json:"args,omitempty"`
	RuntimeEnv  *RuntimeEnv            `json:"runtime_env,omitempty"`

	// catch any unmodeled keys:
	//Extras map[string]interface{} `json:",inline"`
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
	GPUsPerReplica    int                 `json:"gpusPerReplica"`
	TensorParallelism int                 `json:"tensorParallelism"`
	Replicas          int                 `json:"replicas"`
	CPU               string              `json:"cpu,omitempty"`
	Memory            string              `json:"memory,omitempty"`
	NodeSelector      map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations       []corev1.Toleration `json:"tolerations,omitempty"`
	Affinity          *corev1.Affinity    `json:"affinity,omitempty"`
}

type InstanceMapping struct {
	GPUType         string            `json:"gpuType"`
	AcceleratorType string            `json:"acceleratorType"`
	NumGPUs         int               `json:"numGPUs"`
	TotalCPU        int               `json:"totalCPU"`
	TotalMemory     string            `json:"totalMemory"`
	NodeSelector    map[string]string `json:"nodeSelector"`
}

type WorkerGroupKey struct {
	GPUType           string
	GPUsPerReplica    int
	TensorParallelism int
	CPU               string
	Memory            string
}
