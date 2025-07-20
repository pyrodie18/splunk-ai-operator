/*
Copyright 2024.

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

package v1

import (
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AIPlatform is the Schema for the AIPlatform API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=aiplatforms,scope=Namespaced,shortName=spai
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type AIPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AIPlatformSpec   `json:"spec,omitempty"`
	Status AIPlatformStatus `json:"status,omitempty"`
}

// AIPlatformSpec defines the desired state
type AIPlatformSpec struct {
	// user needs to create directory structure
	// s3://bucket/artifacts for AI artifacts
	// s3://bucket/tasks for AI tasks (read and write permission)
	// s3://bucket/models for AI models
	// preferred authentication is via IAM role
	ObjectStorage ObjectStorageSpec `json:"objectStorage"`
	// ServiceAccountName is the name of the service account to use for the AIPlatform
	// used for Ray, Weaviate, SAIA, etc and also IAM role for S3 access
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// GpuInstanceType is the type of GPU instance to use for Ray worker groups
	GpuInstanceType string `json:"gpuInstanceType,omitempty"` // e.g. "g6.24xlarge" or "p4d.24xlarge"
	// options are "saia", "seca"
	// Features to enable in the AIPlatform
	Features []FeatureSpec `json:"features,omitempty"`
	// RayService defines the Ray cluster configuration
	//HeadGroupSpec *HeadGroupSpec `json:"headGroupSpec,omitempty"`
	// WorkerGroupSpec defines the Ray worker group configuration
	WorkerGroupSpec *WorkerGroupSpec `json:"workerGroupSpec,omitempty"`
	// Which sidecars to inject
	Sidecars SidecarConfig `json:"sidecars,omitempty"`

	// cert-manager Certificate for mTLS
	CertificateRef string `json:"certificateRef,omitempty"`

	// Cluster domain (default: cluster.local)
	// +kubebuilder:default=cluster.local
	ClusterDomain string `json:"clusterDomain,omitempty"`

	Images Images `json:"images,omitempty"` // list of image registries to use for Ray
	// DefaultAcceleratorType is the default GPU type to use for Ray worker groups
	DefaultAcceleratorType string `json:"defaultAcceleratorType,omitempty"` // e.g. "nvidia-tesla-t4"

	// SplunkConfiguration instance reference
	SplunkConfiguration SplunkConfiguration `json:"splunkConfiguration,omitempty"`

	//Weaviate       WeaviateSpec     `json:"weaviate,omitempty"`
	Storage StorageSpec `json:"storage,omitempty"`
	// GPUSchedulingSpec defines the scheduling configuration for GPU-based Ray worker groups
	GPUSchedulingSpec *SchedulingSpec `json:"gpuScheduler,omitempty"` // NodeSelector, Tolerations, Affinity
	// CPUSchedulingSpec defines the scheduling configuration for CPU-based Ray worker groups
	CPUSchedulingSpec *SchedulingSpec `json:"cpuScheduler,omitempty"` // NodeSelector, Tolerations, Affinity
	// Ingress defines the Ingress configuration for the AIPlatform
	Ingress *IngressSpec `json:"ingress,omitempty"`
	// MTLS defines the mTLS configuration for the AIPlatform
	MTLS MTLSConfig `json:"mtls,omitempty"`
	//  ServiceTemplate is a template used to create Kubernetes services
	ServiceTemplate corev1.Service `json:"serviceTemplate,omitempty"`
}
type Images struct {
	SAIAImage string `json:"saiaImage,omitempty"`
	// Weaviate image, e.g. "docker.io/weaviate:latest"
	WeaviateImage string `json:"weaviateImage,omitempty"`
	// Ray head group image, e.g. "rayproject/ray-head:latest"
	RayHeadGroupImage string `json:"rayHeadGroupImage,omitempty"`
	// Ray worker group image, e.g. "rayproject/ray-worker:latest"
	RayWorkerGroupImage string `json:"rayWorkerGroupImage,omitempty"`
}

type StorageSpec struct {
	VectorDB VectorDBStorageSpec `json:"vectorDB,omitempty"`
	// Add other storage categories here if needed, e.g., for model artifacts
}

type VectorDBStorageSpec struct {
	// Optional name of an existing PVC to use
	// +optional
	PVCName string `json:"pvcName,omitempty"`

	// Size of the volume to create if PVCName is not provided
	// +kubebuilder:default="50Gi"
	// +optional
	Size string `json:"size,omitempty"`

	// Optional StorageClassName to use for dynamic PVC provisioning
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
}

// FeatureSpec defines the features to enable in the AIPlatform
type FeatureSpec struct {
	// Name of the feature, e.g. "saia" or "seca"
	// +kubebuilder:validation:Enum=saia;seca
	Name string `json:"name,omitempty"`
	// ServiceAccountName is the name of the service account to use for the feature
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// Version of the feature, e.g. "1.0.0"
	Version string `json:"version,omitempty"`
}

type WeaviateSpec struct {
	// +kubebuilder:validation:Minimum=1
	Replicas *int32 `json:"replicas"`
	//Image              string                      `json:"image"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// ServiceAccountName is the name of the service account to use for Weaviate
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// SchedulingSpec defines the scheduling configuration for Weaviate pods
	SchedulingSpec `json:",inline"` // inlines NodeSelector, Tolerations, Affinity
}

type HeadGroupSpec struct {
	// ServiceAccountName is the name of the service account to use for the Ray head group
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// SchedulingSpec defines the scheduling configuration for Ray head group pods
	SchedulingSpec `json:",inline"` // inlines NodeSelector, Tolerations, Affinity
	// ImageRegistry is the image registry to use for the Ray head group
	// image registries for Ray
	ImageRegistry string `json:"imageRegistry,omitempty"`
}

type WorkerGroupSpec struct {
	// ServiceAccountName is the name of the service account to use for Ray worker groups
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// ImageRegistry is the image registry to use for Ray worker groups
	ImageRegistry string `json:"imageRegistry,omitempty"`
	// GPUConfigs defines the GPU worker tiers
	GPUConfigs []GPUConfig `json:"gpuConfigs,omitempty"`
	//SchedulingSpec     `json:",inline"` // inlines NodeSelector, Tolerations, Affinity
}

// GPUConfig defines one worker-tier with scheduling and accelerator settings.
type GPUConfig struct {
	Tier        string                      `json:"tier"`
	MinReplicas int32                       `json:"minReplicas"`
	MaxReplicas int32                       `json:"maxReplicas"`
	GPUsPerPod  int32                       `json:"gpusPerPod"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
}

// SchedulingSpec exposes common pod-scheduling knobs.
type SchedulingSpec struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
}

type SplunkConfiguration struct {
	// Name of the SplunkConfiguration instance
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:MinLength=1
	CRName string `json:"crName,omitempty"`
	// Namespace of the SplunkConfiguration instance
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:MinLength=1
	CRNamespace string `json:"crNamespace,omitempty"`
	// Splunk secret reference
	SecretRef corev1.SecretReference `json:"secretRef,omitempty"`
	Endpoint  string                 `json:"endpoint,omitempty"`
	Token     string                 `json:"token,omitempty"`
}

// ReplicasSpec sets min/max worker replicas
type ReplicasSpec struct {
	Min int32 `json:"min,omitempty"`
	Max int32 `json:"max,omitempty"`
}

// MachineClass configures CPU, memory, GPU per-worker
type MachineClass struct {
	ResourceRequirements corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
	GPU                  int32                       `json:"gpu,omitempty"`
	EphimeralStorage     string                      `json:"ephemeral-storage,omitempty"` // e.g. "100Gi"
}

// SidecarConfig toggles injection of sidecars
type SidecarConfig struct {
	// +kubebuilder:default=true
	Envoy bool `json:"envoy,omitempty"`
	// +kubebuilder:default=true
	FluentBit bool `json:"fluentBit,omitempty"`
	// +kubebuilder:default=true
	Otel bool `json:"otel,omitempty"`
	// +kubebuilder:default=true
	PrometheusOperator bool `json:"prometheusOperator,omitempty"`
}

type ObjectStorageSpec struct {
	// Remote volume URI in the format s3://bucketname/<path prefix>
	Path string `json:"path"` // s3://bucketname/<path prefix> or gs://bucketname/<path prefix> or azure://containername/<path prefix>

	// optional override endpoint (only really needed for S3-compatible like MinIO)
	Endpoint string `json:"endpoint,omitempty"`

	// Region of the remote storage volume where apps reside. Used for aws, if provided. Not used for minio and azure.
	Region string `json:"region"`

	// Secret object name
	SecretRef string `json:"secretRef,omitempty"`
}

type IngressSpec struct {
	Enabled     bool              `json:"enabled,omitempty"`
	ClassName   string            `json:"className,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Hosts       []IngressHost     `json:"hosts,omitempty"`
	TLS         []IngressTLS      `json:"tls,omitempty"`
}

type IngressHost struct {
	Host  string        `json:"host"`
	Paths []IngressPath `json:"paths"`
}

type IngressPath struct {
	Path     string `json:"path"`
	PathType string `json:"pathType"` // e.g., Prefix or Exact
}

type IngressTLS struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secretName"`
}

// AIPlatformStatus defines observed state
type AIPlatformStatus struct {
	RayServiceName      string                     `json:"rayServiceName,omitempty"`
	VectorDbServiceName string                     `json:"vectorDbServiceName,omitempty"`
	RayServiceStatus    rayv1.ServiceStatus        `json:"rayServiceStatus,omitempty"`
	Conditions          []metav1.Condition         `json:"conditions,omitempty"`
	ObservedGeneration  int64                      `json:"observedGeneration,omitempty"`
	Ingress             corev1.LoadBalancerIngress `json:"ingress,omitempty"` // Ingress for the AIPlatform, e.g. for SAIA or Weaviate
}

// +kubebuilder:object:root=true
type AIPlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AIPlatform `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AIPlatform{}, &AIPlatformList{})
}
