package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/splunk/splunk-ai-operator/test/e2e/internal/cfg"
	"github.com/splunk/splunk-ai-operator/test/e2e/internal/k8s"
)

// Webhook Validation E2E Tests
// These tests verify that the webhook defaulting and validation logic works correctly

var _ = Describe("Webhook Validation E2E", Ordered, func() {
	var testNS string

	BeforeAll(func() {
		testNS = cfg.WorkloadNS + "-webhook-test"
		By(fmt.Sprintf("creating test namespace: %s", testNS))
		Expect(k8s.CreateNamespace(testNS)).To(Succeed())

		DeferCleanup(func() {
			By("cleaning up test resources")
			cleanupTestResources(testNS)
			k8s.DeleteNamespace(testNS)
		})

		By("labeling namespace for PSA")
		_ = k8s.LabelNamespace(testNS, "pod-security.kubernetes.io/enforce", "baseline")

		By("creating test Splunk secret")
		err := createTestSplunkSecret(testNS)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("AIPlatform Webhook Defaulting", func() {
		Context("When creating AIPlatform with minimal config", func() {
			It("should apply default values", func() {
				manifestPath := createMinimalAIPlatformManifest(testNS)
				defer os.Remove(manifestPath)

				By("applying minimal AIPlatform")
				_, err := k8s.Apply(testNS, manifestPath)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for resource to be created")
				time.Sleep(5 * time.Second)

				By("verifying clusterDomain was defaulted")
				Eventually(func(g Gomega) {
					clusterDomain, err := getAIPlatformField(testNS, "minimal-test", ".spec.clusterDomain")
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(clusterDomain).To(Equal("cluster.local"))
				}, 30*time.Second, 2*time.Second).Should(Succeed())

				By("verifying storage.vectorDB.size was defaulted to 50Gi")
				Eventually(func(g Gomega) {
					size, err := getAIPlatformField(testNS, "minimal-test", ".spec.storage.vectorDB.size")
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(size).To(Equal("50Gi"))
				}, 30*time.Second, 2*time.Second).Should(Succeed())
			})
		})
	})

	Describe("AIPlatform Webhook Validation", func() {
		Context("When creating AIPlatform with invalid objectStorage path", func() {
			It("should reject the resource", func() {
				manifestPath := createInvalidObjectStoragePathManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply invalid AIPlatform")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject invalid objectStorage path")

				By("verifying error message mentions path validation")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(ContainSubstring("path"), "Error should mention 'path'")
			})
		})

		Context("When creating AIPlatform with missing S3 region", func() {
			It("should reject the resource", func() {
				manifestPath := createMissingS3RegionManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply AIPlatform without S3 region")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject S3 path without region")

				By("verifying error message mentions region requirement")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(ContainSubstring("region"), "Error should mention 'region'")
			})
		})

		Context("When creating AIPlatform with missing SplunkConfiguration", func() {
			It("should reject the resource", func() {
				manifestPath := createMissingSplunkConfigManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply AIPlatform without SplunkConfiguration")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject missing SplunkConfiguration")

				By("verifying error message mentions SplunkConfiguration")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(ContainSubstring("splunkconfiguration"), "Error should mention 'SplunkConfiguration'")
			})
		})

		Context("When creating AIPlatform with invalid storage size", func() {
			It("should reject the resource", func() {
				manifestPath := createInvalidStorageSizeManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply AIPlatform with invalid storage size")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject invalid storage size")

				By("verifying error message mentions size validation")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(Or(ContainSubstring("size"), ContainSubstring("invalid")))
			})
		})

		Context("When creating AIPlatform with both pvcName and size", func() {
			It("should reject the resource", func() {
				manifestPath := createConflictingStorageManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply AIPlatform with both pvcName and size")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject conflicting storage config")

				By("verifying error message mentions the conflict")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(Or(ContainSubstring("both"), ContainSubstring("cannot")))
			})
		})

		Context("When creating AIPlatform with invalid ingress pathType", func() {
			It("should reject the resource", func() {
				manifestPath := createInvalidIngressPathTypeManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply AIPlatform with invalid pathType")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject invalid pathType")

				By("verifying error message mentions pathType")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(ContainSubstring("pathtype"))
			})
		})
	})

	Describe("AIPlatform Immutability Validation", func() {
		Context("When updating objectStorage.path", func() {
			It("should reject the update", func() {
				By("creating AIPlatform with initial path")
				manifestPath := createImmutableTestManifest(testNS)
				defer os.Remove(manifestPath)

				_, err := k8s.Apply(testNS, manifestPath)
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(5 * time.Second)

				By("attempting to update objectStorage.path")
				manifestPath2 := createImmutableTestManifestUpdated(testNS)
				defer os.Remove(manifestPath2)

				output, err := k8s.Apply(testNS, manifestPath2)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject immutable field update")

				By("verifying error message mentions immutability")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(Or(ContainSubstring("immutable"), ContainSubstring("forbidden")))
			})
		})
	})

	Describe("AIService Webhook Defaulting", func() {
		Context("When creating AIService with minimal config", func() {
			It("should apply default values", func() {
				manifestPath := createMinimalAIServiceManifest(testNS)
				defer os.Remove(manifestPath)

				By("applying minimal AIService")
				_, err := k8s.Apply(testNS, manifestPath)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for resource to be created")
				time.Sleep(5 * time.Second)

				By("verifying port was defaulted to 80")
				Eventually(func(g Gomega) {
					port, err := getAIServiceField(testNS, "minimal-service", ".spec.port")
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(port).To(Equal("80"))
				}, 30*time.Second, 2*time.Second).Should(Succeed())

				By("verifying replicas was defaulted to 1")
				Eventually(func(g Gomega) {
					replicas, err := getAIServiceField(testNS, "minimal-service", ".spec.replicas")
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(replicas).To(Equal("1"))
				}, 30*time.Second, 2*time.Second).Should(Succeed())
			})
		})
	})

	Describe("AIService Webhook Validation", func() {
		Context("When creating AIService without aiPlatformRef", func() {
			It("should reject the resource", func() {
				manifestPath := createMissingAIPlatformRefManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply AIService without aiPlatformRef")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject missing aiPlatformRef")

				By("verifying error message mentions aiPlatformRef")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(ContainSubstring("aiplatformref"))
			})
		})

		Context("When creating AIService with invalid vectorDbUrl", func() {
			It("should reject the resource", func() {
				manifestPath := createInvalidVectorDbUrlManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply AIService with invalid vectorDbUrl")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject invalid vectorDbUrl")

				By("verifying error message mentions vectorDbUrl or URL format")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(Or(ContainSubstring("vectordburl"), ContainSubstring("http")))
			})
		})

		Context("When creating AIService with invalid port", func() {
			It("should reject the resource", func() {
				manifestPath := createInvalidPortManifest(testNS)
				defer os.Remove(manifestPath)

				By("attempting to apply AIService with invalid port")
				output, err := k8s.Apply(testNS, manifestPath)
				Expect(err).To(HaveOccurred(), "Expected webhook to reject invalid port")

				By("verifying error message mentions port")
				errorMsg := strings.ToLower(output + err.Error())
				Expect(errorMsg).To(ContainSubstring("port"))
			})
		})
	})
})

// Helper functions for creating test manifests

func createMinimalAIPlatformManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: minimal-test
  namespace: %s
spec:
  objectStorage:
    path: s3://test-bucket/models
    region: us-west-2
  serviceAccountName: test-sa
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns)
	return writeTempManifest("minimal-aiplatform", manifest)
}

func createInvalidObjectStoragePathManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: invalid-path-test
  namespace: %s
spec:
  objectStorage:
    path: /invalid/local/path
    region: us-west-2
  serviceAccountName: test-sa
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns)
	return writeTempManifest("invalid-path", manifest)
}

func createMissingS3RegionManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: missing-region-test
  namespace: %s
spec:
  objectStorage:
    path: s3://test-bucket/models
  serviceAccountName: test-sa
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns)
	return writeTempManifest("missing-region", manifest)
}

func createMissingSplunkConfigManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: missing-splunk-test
  namespace: %s
spec:
  objectStorage:
    path: s3://test-bucket/models
    region: us-west-2
  serviceAccountName: test-sa
  splunkConfiguration: {}
`, ns)
	return writeTempManifest("missing-splunk", manifest)
}

func createInvalidStorageSizeManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: invalid-size-test
  namespace: %s
spec:
  objectStorage:
    path: s3://test-bucket/models
    region: us-west-2
  serviceAccountName: test-sa
  storage:
    vectorDB:
      size: "invalid-size"
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns)
	return writeTempManifest("invalid-size", manifest)
}

func createConflictingStorageManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: conflict-storage-test
  namespace: %s
spec:
  objectStorage:
    path: s3://test-bucket/models
    region: us-west-2
  serviceAccountName: test-sa
  storage:
    vectorDB:
      pvcName: existing-pvc
      size: 50Gi
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns)
	return writeTempManifest("conflict-storage", manifest)
}

func createInvalidIngressPathTypeManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: invalid-pathtype-test
  namespace: %s
spec:
  objectStorage:
    path: s3://test-bucket/models
    region: us-west-2
  serviceAccountName: test-sa
  ingress:
    enabled: true
    hosts:
      - host: test.example.com
        paths:
          - path: /
            pathType: InvalidType
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns)
	return writeTempManifest("invalid-pathtype", manifest)
}

func createImmutableTestManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: immutable-test
  namespace: %s
spec:
  objectStorage:
    path: s3://original-bucket/models
    region: us-west-2
  serviceAccountName: test-sa
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns)
	return writeTempManifest("immutable-original", manifest)
}

func createImmutableTestManifestUpdated(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: immutable-test
  namespace: %s
spec:
  objectStorage:
    path: s3://updated-bucket/models
    region: us-west-2
  serviceAccountName: test-sa
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns)
	return writeTempManifest("immutable-updated", manifest)
}

func createMinimalAIServiceManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIService
metadata:
  name: minimal-service
  namespace: %s
spec:
  aiPlatformRef:
    name: test-platform
    namespace: %s
  vectorDbUrl: http://weaviate.%s.svc.cluster.local
  taskVolume:
    path: s3://test-bucket/tasks
    region: us-west-2
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns, ns, ns)
	return writeTempManifest("minimal-aiservice", manifest)
}

func createMissingAIPlatformRefManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIService
metadata:
  name: missing-ref-test
  namespace: %s
spec:
  vectorDbUrl: http://weaviate.%s.svc.cluster.local
  taskVolume:
    path: s3://test-bucket/tasks
    region: us-west-2
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns, ns)
	return writeTempManifest("missing-ref", manifest)
}

func createInvalidVectorDbUrlManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIService
metadata:
  name: invalid-url-test
  namespace: %s
spec:
  aiPlatformRef:
    name: test-platform
    namespace: %s
  vectorDbUrl: weaviate:8080
  taskVolume:
    path: s3://test-bucket/tasks
    region: us-west-2
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns, ns)
	return writeTempManifest("invalid-url", manifest)
}

func createInvalidPortManifest(ns string) string {
	manifest := fmt.Sprintf(`apiVersion: ai.splunk.com/v1
kind: AIService
metadata:
  name: invalid-port-test
  namespace: %s
spec:
  aiPlatformRef:
    name: test-platform
    namespace: %s
  vectorDbUrl: http://weaviate.%s.svc.cluster.local
  taskVolume:
    path: s3://test-bucket/tasks
    region: us-west-2
  port: 70000
  splunkConfiguration:
    endpoint: http://test-splunk-service.%s.svc.cluster.local:8089
    secretRef:
      name: splunk-%s-secret
      namespace: %s
`, ns, ns, ns, ns, ns, ns)
	return writeTempManifest("invalid-port", manifest)
}

// Helper functions to get resource fields using kubectl
func getAIPlatformField(ns, name, jsonpath string) (string, error) {
	cmd := exec.Command("kubectl", "get", "aiplatform", name, "-n", ns, "-o", fmt.Sprintf("jsonpath={%s}", jsonpath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get field: %w, output: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func getAIServiceField(ns, name, jsonpath string) (string, error) {
	cmd := exec.Command("kubectl", "get", "aiservice", name, "-n", ns, "-o", fmt.Sprintf("jsonpath={%s}", jsonpath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get field: %w, output: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}
