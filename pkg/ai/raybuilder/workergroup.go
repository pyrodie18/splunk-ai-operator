package raybuilder

import (
	//"context"
	"encoding/json"
	//"fmt"
	"os"
	//"strconv"
	"strings"

	//rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	//"k8s.io/apimachinery/pkg/api/resource"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//utilpointer "k8s.io/utils/pointer"
	//"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/log"
)

type InstanceMap map[string]map[string]InstanceDetails

func LoadInstanceMap(filename string) (InstanceMap, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var instanceMap InstanceMap
	if err := json.Unmarshal(data, &instanceMap); err != nil {
		return nil, err
	}
	return instanceMap, nil
}

func GetProviderAndInstance(node *corev1.Node) (string, string) {
	providerID := node.Spec.ProviderID
	instanceType := node.Labels["node.kubernetes.io/instance-type"] // or "beta.kubernetes.io/instance-type"
	provider := "unknown"

	switch {
	case strings.HasPrefix(providerID, "aws://"):
		provider = "aws"
	case strings.HasPrefix(providerID, "gce://"):
		provider = "gcp"
	case strings.HasPrefix(providerID, "azure://"):
		provider = "azure"
	}

	return provider, instanceType
}

func LookupGPUFromInstanceMap(instanceMap InstanceMap, provider, instanceType string) (*InstanceDetails, bool) {
	providerMap, ok := instanceMap[provider]
	if !ok {
		return nil, false
	}
	info, ok := providerMap[instanceType]
	return &info, ok
}
