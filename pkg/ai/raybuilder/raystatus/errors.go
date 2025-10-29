package raystatus

import (
	"context"
	"fmt"
	"strings"

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RayErrorDetails contains structured error information from Ray components
type RayErrorDetails struct {
	HasError          bool
	ServiceErrors     []string
	ClusterErrors     []string
	PodErrors         []string
	ApplicationErrors map[string]string // application name -> error message
	Summary           string
}

// WeaviateErrorDetails contains structured error information from Weaviate
type WeaviateErrorDetails struct {
	HasError         bool
	StatefulSetError string
	PodErrors        []string
	ServiceError     string
	Summary          string
}

// ExtractRayErrors collects detailed error information from Ray components
func ExtractRayErrors(ctx context.Context, c client.Client, ns, name string) *RayErrorDetails {
	details := &RayErrorDetails{
		ApplicationErrors: make(map[string]string),
	}

	// 1) Check RayService status for errors
	rs := &rayv1.RayService{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, rs); err != nil {
		details.HasError = true
		details.ServiceErrors = append(details.ServiceErrors, fmt.Sprintf("Failed to get RayService: %v", err))
		details.Summary = fmt.Sprintf("RayService not found: %v", err)
		return details
	}

	// Check RayService conditions for errors
	for _, cond := range rs.Status.Conditions {
		if cond.Status == "False" && cond.Message != "" {
			details.HasError = true
			details.ServiceErrors = append(details.ServiceErrors,
				fmt.Sprintf("%s: %s - %s", cond.Type, cond.Reason, cond.Message))
		}
	}

	// Check application statuses from RayServe (both active and pending)
	checkAppStatuses := func(appStatuses map[string]rayv1.AppStatus, prefix string) {
		for appName, appStatus := range appStatuses {
			if appStatus.Status != "RUNNING" && appStatus.Message != "" {
				details.HasError = true
				// Extract the actual error from the message (may contain stack traces)
				errorMsg := extractConciseError(appStatus.Message)
				details.ApplicationErrors[appName] = fmt.Sprintf("[%s] %s: %s",
					prefix, appStatus.Status, errorMsg)
			}

			// Check deployment statuses within each application
			for deploymentName, deployStatus := range appStatus.Deployments {
				if deployStatus.Status != "HEALTHY" && deployStatus.Message != "" {
					details.HasError = true
					errorMsg := extractConciseError(deployStatus.Message)
					key := fmt.Sprintf("%s:%s", appName, deploymentName)
					details.ApplicationErrors[key] = fmt.Sprintf("[%s] %s: %s",
						prefix, deployStatus.Status, errorMsg)
				}
			}
		}
	}

	if rs.Status.ActiveServiceStatus.Applications != nil {
		checkAppStatuses(rs.Status.ActiveServiceStatus.Applications, "active")
	}
	if rs.Status.PendingServiceStatus.Applications != nil {
		checkAppStatuses(rs.Status.PendingServiceStatus.Applications, "pending")
	}

	// 2) Check RayCluster status
	clusterName := rs.Status.ActiveServiceStatus.RayClusterName
	if clusterName == "" {
		clusterName = fmt.Sprintf("%s-raycluster", name)
	}

	rc := &rayv1.RayCluster{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: clusterName}, rc); err == nil {
		// Check cluster conditions
		for _, cond := range rc.Status.Conditions {
			if cond.Status == "False" && cond.Message != "" {
				details.HasError = true
				details.ClusterErrors = append(details.ClusterErrors,
					fmt.Sprintf("%s: %s - %s", cond.Type, cond.Reason, cond.Message))
			}
		}

		// Check cluster state
		if rc.Status.State != rayv1.Ready {
			details.HasError = true
			details.ClusterErrors = append(details.ClusterErrors,
				fmt.Sprintf("Cluster state: %s (expected: ready)", rc.Status.State))
		}
	}

	// 3) Check Ray pods for errors
	var pods corev1.PodList
	listOpts := &client.ListOptions{
		Namespace: ns,
	}
	client.MatchingLabels{"ray.io/cluster": clusterName}.ApplyToList(listOpts)
	if err := c.List(ctx, &pods, listOpts); err == nil {
		for _, pod := range pods.Items {
			if podError := extractPodError(&pod); podError != "" {
				details.HasError = true
				details.PodErrors = append(details.PodErrors,
					fmt.Sprintf("%s: %s", pod.Name, podError))
			}
		}
	}

	// Generate summary
	if details.HasError {
		summaryParts := []string{}
		if len(details.ServiceErrors) > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("RayService: %s", details.ServiceErrors[0]))
		}
		if len(details.ApplicationErrors) > 0 {
			// Get first app error
			for appName, appError := range details.ApplicationErrors {
				summaryParts = append(summaryParts, fmt.Sprintf("App %s: %s", appName, truncate(appError, 100)))
				break
			}
		}
		if len(details.ClusterErrors) > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("Cluster: %s", details.ClusterErrors[0]))
		}
		if len(details.PodErrors) > 0 && len(summaryParts) < 2 {
			summaryParts = append(summaryParts, fmt.Sprintf("Pods: %d errors", len(details.PodErrors)))
		}
		details.Summary = strings.Join(summaryParts, "; ")
	}

	return details
}

// ExtractWeaviateErrors collects detailed error information from Weaviate components
func ExtractWeaviateErrors(ctx context.Context, c client.Client, ns, name string) *WeaviateErrorDetails {
	details := &WeaviateErrorDetails{}

	weaviateName := fmt.Sprintf("%s-weaviate", name)

	// 1) Check StatefulSet status
	sts := &appsv1.StatefulSet{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: weaviateName}, sts); err != nil {
		details.HasError = true
		details.StatefulSetError = fmt.Sprintf("StatefulSet not found: %v", err)
		details.Summary = details.StatefulSetError
		return details
	}

	// Check if replicas are ready
	if sts.Status.ReadyReplicas != *sts.Spec.Replicas {
		details.HasError = true
		details.StatefulSetError = fmt.Sprintf("Ready replicas %d/%d",
			sts.Status.ReadyReplicas, *sts.Spec.Replicas)
	}

	// Check conditions
	for _, cond := range sts.Status.Conditions {
		if cond.Status == corev1.ConditionFalse && cond.Message != "" {
			details.HasError = true
			if details.StatefulSetError != "" {
				details.StatefulSetError += "; "
			}
			details.StatefulSetError += fmt.Sprintf("%s: %s", cond.Type, cond.Message)
		}
	}

	// 2) Check Weaviate pods
	var pods corev1.PodList
	listOpts := &client.ListOptions{
		Namespace: ns,
	}
	client.MatchingLabels{"app": "weaviate"}.ApplyToList(listOpts)
	if err := c.List(ctx, &pods, listOpts); err == nil {
		for _, pod := range pods.Items {
			if podError := extractPodError(&pod); podError != "" {
				details.HasError = true
				details.PodErrors = append(details.PodErrors,
					fmt.Sprintf("%s: %s", pod.Name, podError))
			}
		}
	}

	// 3) Check Weaviate service
	svc := &corev1.Service{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: weaviateName}, svc); err != nil {
		details.HasError = true
		details.ServiceError = fmt.Sprintf("Service not found: %v", err)
	}

	// Generate summary
	if details.HasError {
		summaryParts := []string{}
		if details.StatefulSetError != "" {
			summaryParts = append(summaryParts, details.StatefulSetError)
		}
		if len(details.PodErrors) > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d pod error(s)", len(details.PodErrors)))
		}
		if details.ServiceError != "" {
			summaryParts = append(summaryParts, details.ServiceError)
		}
		details.Summary = strings.Join(summaryParts, "; ")
	}

	return details
}

// extractPodError gets the most relevant error from a pod
func extractPodError(pod *corev1.Pod) string {
	// Check pod phase
	if pod.Status.Phase == corev1.PodFailed {
		return fmt.Sprintf("Pod failed: %s - %s", pod.Status.Reason, pod.Status.Message)
	}

	// Check pod conditions
	for _, cond := range pod.Status.Conditions {
		if cond.Status == corev1.ConditionFalse && cond.Type != corev1.PodScheduled {
			if cond.Message != "" {
				return fmt.Sprintf("%s: %s - %s", cond.Type, cond.Reason, cond.Message)
			}
		}
	}

	// Check container statuses
	allStatuses := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
	for _, cs := range allStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			msg := cs.State.Waiting.Message
			if msg == "" {
				msg = cs.State.Waiting.Reason
			}
			return fmt.Sprintf("Container %s waiting: %s", cs.Name, msg)
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			return fmt.Sprintf("Container %s terminated: exit code %d - %s",
				cs.Name, cs.State.Terminated.ExitCode, cs.State.Terminated.Reason)
		}
		if cs.RestartCount > 0 && !cs.Ready {
			return fmt.Sprintf("Container %s: %d restarts, not ready", cs.Name, cs.RestartCount)
		}
	}

	return ""
}

// extractConciseError extracts the key error message from potentially long error strings
func extractConciseError(fullError string) string {
	// Look for common error patterns

	// ValidationError from vLLM
	if idx := strings.Index(fullError, "ValidationError:"); idx != -1 {
		rest := fullError[idx:]
		// Get up to the first line break or 200 chars
		if newlineIdx := strings.Index(rest, "\n"); newlineIdx != -1 && newlineIdx < 200 {
			return rest[:newlineIdx]
		}
		if len(rest) > 200 {
			return rest[:200] + "..."
		}
		return rest
	}

	// RuntimeError
	if idx := strings.Index(fullError, "RuntimeError:"); idx != -1 {
		rest := fullError[idx:]
		if newlineIdx := strings.Index(rest, "\n"); newlineIdx != -1 && newlineIdx < 200 {
			return rest[:newlineIdx]
		}
		if len(rest) > 200 {
			return rest[:200] + "..."
		}
		return rest
	}

	// FileNotFoundError
	if idx := strings.Index(fullError, "FileNotFoundError:"); idx != -1 {
		rest := fullError[idx:]
		if newlineIdx := strings.Index(rest, "\n"); newlineIdx != -1 && newlineIdx < 200 {
			return rest[:newlineIdx]
		}
		return rest
	}

	// Generic error extraction - get first meaningful line
	lines := strings.Split(fullError, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 20 && !strings.HasPrefix(line, "File ") &&
			!strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "Traceback") {
			if len(line) > 300 {
				return line[:300] + "..."
			}
			return line
		}
	}

	// Fallback: truncate the full error
	return truncate(fullError, 300)
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
