package common

import (
	"context"
	"fmt"
	//"io"
	"net/http"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var IsConditionTrue = func(conditions []metav1.Condition, condType string) bool {
	for _, cond := range conditions {
		if cond.Type == condType && cond.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

var CheckRayHeadService = func(ctx context.Context, rayHeadURL string) error {
	if rayHeadURL == "" {
		return fmt.Errorf("ray head endpoint is empty")
	}

	// Ensure scheme
	if !strings.HasPrefix(rayHeadURL, "http://") && !strings.HasPrefix(rayHeadURL, "https://") {
		rayHeadURL = "http://" + rayHeadURL
	}

	// Ray's default health endpoint
	healthURL := fmt.Sprintf("%s/api/health", strings.TrimSuffix(rayHeadURL, "/"))

	//healthURL = "http://localhost:8265" // Default Ray head service endpoint //FIXME: should be configurable

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("failed to reach Ray head endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ray head health endpoint returned %d", resp.StatusCode)
	}

	//body, err := io.ReadAll(resp.Body)
	//if err != nil {
	//    return fmt.Errorf("failed to read Ray head response: %w", err)
	//}

	// Health endpoint returns a plain string "ok" if healthy
	//if strings.TrimSpace(string(body)) != "ok" {
	//    return fmt.Errorf("ray head reported unhealthy: %s", string(body))
	//}

	return nil
}

var CheckWeaviateService = func(ctx context.Context, weaviateURL string) error {
	// Weaviate readiness endpoint
	readyURL := fmt.Sprintf("%s/v1/.well-known/ready", strings.TrimSuffix(weaviateURL, "/")) // FIXME port is not configured
	//readyURL = "http://localhost:8999/v1/.well-known/ready" // Default Weaviate service endpoint //FIXME: should be configurable

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(readyURL)
	if err != nil {
		return fmt.Errorf("failed to reach Weaviate endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("weaviate not ready, returned status=%d", resp.StatusCode)
	}

	return nil
}
