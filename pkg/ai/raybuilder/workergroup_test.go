package raybuilder

import (
	"encoding/json"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Dummy InstanceDetails for testing

func TestLoadInstanceMap(t *testing.T) {
	// Prepare a temporary file with valid JSON
	tmpFile, err := os.CreateTemp("", "instance_map_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testMap := InstanceMap{
		"aws": {
			"m5.large": InstanceDetails{GPUType: "none"},
		},
	}
	data, _ := json.Marshal(testMap)
	if _, err := tmpFile.Write(data); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	loadedMap, err := LoadInstanceMap(tmpFile.Name())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if loadedMap["aws"]["m5.large"].GPUType != "none" {
		t.Errorf("Expected GPU 'none', got %v", loadedMap["aws"]["m5.large"].GPUType)
	}
}

func TestLoadInstanceMap_FileNotFound(t *testing.T) {
	_, err := LoadInstanceMap("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestLoadInstanceMap_InvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bad_json_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("{invalid json")); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	_, err = LoadInstanceMap(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestGetProviderAndInstance(t *testing.T) {
	tests := []struct {
		providerID   string
		instanceType string
		wantProvider string
	}{
		{"aws://123", "m5.large", "aws"},
		{"gce://456", "n1-standard-4", "gcp"},
		{"azure://789", "Standard_D2_v3", "azure"},
		{"other://000", "custom-type", "unknown"},
	}

	for _, tt := range tests {
		node := &corev1.Node{
			Spec: corev1.NodeSpec{
				ProviderID: tt.providerID,
			},
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"node.kubernetes.io/instance-type": tt.instanceType,
				},
			},
		}
		provider, instanceType := GetProviderAndInstance(node)
		if provider != tt.wantProvider {
			t.Errorf("ProviderID %q: expected provider %q, got %q", tt.providerID, tt.wantProvider, provider)
		}
		if instanceType != tt.instanceType {
			t.Errorf("ProviderID %q: expected instanceType %q, got %q", tt.providerID, tt.instanceType, instanceType)
		}
	}
}

func TestLookupGPUFromInstanceMap(t *testing.T) {
	instanceMap := InstanceMap{
		"aws": {
			"p3.2xlarge": InstanceDetails{GPUType: "V100"},
		},
		"gcp": {
			"n1-standard-4": InstanceDetails{GPUType: "none"},
		},
	}

	// Existing provider and instanceType
	info, ok := LookupGPUFromInstanceMap(instanceMap, "aws", "p3.2xlarge")
	if !ok || info.GPUType != "V100" {
		t.Errorf("Expected GPU 'V100', got %v, ok=%v", info, ok)
	}

	// Non-existing provider
	info, ok = LookupGPUFromInstanceMap(instanceMap, "azure", "Standard_D2_v3")
	if ok || info != nil {
		t.Errorf("Expected nil and false for non-existing provider, got %v, ok=%v", info, ok)
	}

	// Non-existing instanceType
	info, ok = LookupGPUFromInstanceMap(instanceMap, "aws", "nonexistent")
	if ok || info == nil {
		t.Errorf("Expected nil and false for non-existing instanceType, got %v, ok=%v", info, ok)
	}
}