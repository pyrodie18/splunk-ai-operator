package k8s

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"

	pathutil "github.com/splunk/splunk-ai-operator/test/e2e/internal/path"
	"github.com/splunk/splunk-ai-operator/test/utils"
)

// withRepoDir sets the working directory of the command to the repository root
// (detected by walking up to the directory that contains go.mod).
func withRepoDir(cmd *exec.Cmd) error {
	root, err := pathutil.RepoRoot()
	if err != nil {
		return err
	}
	cmd.Dir = root
	return nil
}

// CreateNamespace creates a Kubernetes namespace.
func CreateNamespace(ns string) error {
	cmd := exec.Command("kubectl", "create", "ns", ns)
	if err := withRepoDir(cmd); err != nil {
		return err
	}
	out, err := utils.Run(cmd)
	if err != nil && strings.Contains(out, "AlreadyExists") {
		return nil // ignore if namespace already exists
	}
	return err
}

// LabelNamespace applies/overwrites a label on a namespace.
func LabelNamespace(ns, key, val string) error {
	cmd := exec.Command("kubectl", "label", "--overwrite", "ns", ns, fmt.Sprintf("%s=%s", key, val))
	if err := withRepoDir(cmd); err != nil {
		return err
	}
	_, err := utils.Run(cmd)
	return err
}

// Apply applies a manifest file into the target namespace.
func Apply(ns, manifestPath string) (string, error) {
	cmd := exec.Command("kubectl", "apply", "-n", ns, "-f", manifestPath)
	if err := withRepoDir(cmd); err != nil {
		return "", err
	}
	return utils.Run(cmd)
}

// Delete deletes a manifest file from the target namespace (best-effort).
func Delete(ns, manifestPath string) {
	cmd := exec.Command("kubectl", "delete", "-n", ns, "-f", manifestPath, "--ignore-not-found=true")
	if err := withRepoDir(cmd); err == nil {
		_, _ = utils.Run(cmd)
	}
}

// GetControllerPodName returns the name of the controller-manager pod.
func GetControllerPodName(ns string) (string, error) {
	cmd := exec.Command(
		"kubectl", "get", "pods",
		"-l", "control-plane=controller-manager",
		"-o", `go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}{{ "\n" }}{{ end }}{{ end }}`,
		"-n", ns,
	)
	if err := withRepoDir(cmd); err != nil {
		return "", err
	}
	out, err := utils.Run(cmd)
	if err != nil {
		return "", err
	}
	names := utils.GetNonEmptyLines(out)
	if len(names) != 1 {
		return "", fmt.Errorf("expected 1 controller pod, got %d (names=%v)", len(names), names)
	}
	return names[0], nil
}

// PodPhase returns the phase (Running/Pending/Failed/...) of a pod.
func PodPhase(ns, name string) (string, error) {
	cmd := exec.Command("kubectl", "get", "pods", name, "-o", "jsonpath={.status.phase}", "-n", ns)
	if err := withRepoDir(cmd); err != nil {
		return "", err
	}
	return utils.Run(cmd)
}

// ServiceHasEndpointPort checks Endpoints first, then EndpointSlices for a ready address on the port.
func ServiceHasEndpointPort(ns, svc, port string) (bool, error) {
	// Try Endpoints
	{
		cmd := exec.Command("kubectl", "get", "endpoints", svc, "-n", ns, "-o", "json")
		if err := withRepoDir(cmd); err != nil {
			return false, err
		}
		out, err := cmd.CombinedOutput()
		if err == nil && len(out) > 0 {
			var ep struct {
				Subsets []struct {
					Ports []struct {
						Port int `json:"port"`
					} `json:"ports"`
					Addresses []struct {
						IP string `json:"ip"`
					} `json:"addresses"`
				} `json:"subsets"`
			}
			if json.Unmarshal(out, &ep) == nil {
				for _, s := range ep.Subsets {
					for _, p := range s.Ports {
						if fmt.Sprint(p.Port) == port && len(s.Addresses) > 0 {
							return true, nil
						}
					}
				}
			}
		}
	}
	// Fallback to EndpointSlice
	{
		cmd := exec.Command("kubectl", "get", "endpointslice", "-n", ns, "-l", "kubernetes.io/service-name="+svc, "-o", "json")
		if err := withRepoDir(cmd); err != nil {
			return false, err
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false, err
		}
		var es struct {
			Items []struct {
				Ports []struct {
					Port *int `json:"port"`
				} `json:"ports"`
				Endpoints []struct {
					Addresses  []string `json:"addresses"`
					Conditions struct {
						Ready *bool `json:"ready"`
					} `json:"conditions"`
				} `json:"endpoints"`
			} `json:"items"`
		}
		if json.Unmarshal(out, &es) == nil {
			for _, item := range es.Items {
				for _, p := range item.Ports {
					if p.Port != nil && fmt.Sprint(*p.Port) == port {
						for _, e := range item.Endpoints {
							if (e.Conditions.Ready == nil || *e.Conditions.Ready) && len(e.Addresses) > 0 {
								return true, nil
							}
						}
					}
				}
			}
		}
	}
	return false, nil
}

// PortForwardService runs `kubectl port-forward` for a Service and returns a cancel func.
// It waits until the forwarder reports readiness in its output.
func PortForwardService(ns, svc, localPort, remotePort string) (func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(
		ctx, "kubectl", "-n", ns, "port-forward",
		"svc/"+svc, fmt.Sprintf("%s:%s", localPort, remotePort),
	)
	if err := withRepoDir(cmd); err != nil {
		cancel()
		return nil, err
	}

	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	readyCh := make(chan struct{})
	go func() {
		defer close(readyCh)
		sc := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for sc.Scan() {
			line := sc.Text()
			_, _ = fmt.Fprintf(GinkgoWriter, "[port-forward] %s\n", line)
			if strings.Contains(line, "Forwarding from 127.0.0.1:"+localPort) ||
				strings.Contains(line, "Handling connection for "+localPort) {
				readyCh <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-readyCh:
		return cancel, nil
	case <-time.After(20 * time.Second):
		cancel()
		return nil, fmt.Errorf("timed out waiting for port-forward on %s", localPort)
	}
}

// WaitCRReady waits until a namespaced CR shows Ready=True in status.conditions.
// readyCondition is the condition type to check (e.g., "Ready").
func WaitCRReady(kind, name, ns, readyCondition string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastOut []byte

	for time.Now().Before(deadline) {
		cmd := exec.Command("kubectl", "get", kind, name, "-n", ns, "-o", "json")
		if err := withRepoDir(cmd); err != nil {
			return err
		}
		out, err := cmd.CombinedOutput()
		if err == nil {
			var obj struct {
				Status struct {
					Conditions []struct {
						Type   string `json:"type"`
						Status string `json:"status"`
					} `json:"conditions"`
				} `json:"status"`
			}
			_ = json.Unmarshal(out, &obj)
			for _, c := range obj.Status.Conditions {
				if c.Type == readyCondition && strings.EqualFold(c.Status, "true") {
					return nil
				}
			}
			lastOut = out
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("%s/%s not Ready within %s; last get output:\n%s", kind, name, timeout, string(lastOut))
}

// GetLogs returns `kubectl logs` for a pod.
func GetLogs(ns, pod string) (string, error) {
	cmd := exec.Command("kubectl", "logs", pod, "-n", ns)
	if err := withRepoDir(cmd); err != nil {
		return "", err
	}
	return utils.Run(cmd)
}
