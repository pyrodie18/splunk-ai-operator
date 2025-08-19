package cfg

import (
	"os"
	"time"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

var (
	// operator image under test
	ProjectImage = envOr("IMG", "docker.com/splunk/splunk-ai-operator:v0.0.1")

	// namespaces
	OperatorNS = envOr("OPERATOR_NAMESPACE", "splunk-ai-operator-system")
	WorkloadNS = envOr("WORKLOAD_NAMESPACE", "aiplatform-e2e")

	// service account + metrics
	ServiceAccountName   = envOr("SERVICE_ACCOUNT", "splunk-ai-operator-controller-manager")
	MetricsServiceName   = envOr("METRICS_SERVICE", "splunk-ai-operator-controller-manager-metrics-service")
	MetricsRoleBindName  = envOr("METRICS_RBAC", "splunk-ai-operator-metrics-binding")

	// cert-manager behavior
	SkipCertManagerInstall = os.Getenv("CERT_MANAGER_INSTALL_SKIP") == "true"

	// samples (override in CI if needed)
	SampleAIPlatform = envOr("AIPLATFORM_SAMPLE", "config/samples/ai_v1_aiplatform.yaml")
	SampleAIService  = envOr("AISERVICE_SAMPLE",  "config/samples/ai_v1_aiservice.yaml")

	// CR names
	AIPlatformName = envOr("AIPLATFORM_NAME", "testtenant")
	AIServiceName  = envOr("AISERVICE_NAME",  "saia")

	// readiness
	ReadyConditionType     = envOr("READY_CONDITION", "Ready")
	AIPlatformReadyTimeout = durationEnv("AIPLATFORM_READY_TIMEOUT", 12*time.Minute)
	AIServiceReadyTimeout  = durationEnv("AISERVICE_READY_TIMEOUT", 10*time.Minute)

	// port-forward target (prefer a Service)
	ForwardLocalPort  = envOr("FORWARD_LOCAL",  "8080")
	ForwardRemotePort = envOr("FORWARD_REMOTE", "8080")
	ServiceToForward  = envOr("FORWARD_SERVICE","saia-gateway")
	ServiceNamespace  = envOr("FORWARD_NAMESPACE", WorkloadNS)

	// REST request
	SAIAPOSTPath = envOr("SAIA_POST_PATH", "/testtenant/saia-api/v1alpha1/saia/search")
	SAIABody     = []byte(`{
  "chat_history": "[{\"content\":\"Using data models, get top 10 users with the most failed login attempts over the past week \",\"role\":\"user\"}]",
  "user_prompt": "Using data models, get top 2 users with the most failed login attempts over the past week",
  "classification": "1",
  "user_id": "tester",
  "user_roles": "[\"admin\", \"user\"]"
}`)
)

func durationEnv(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
