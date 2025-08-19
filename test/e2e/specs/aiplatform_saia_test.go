package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/splunk/splunk-ai-operator/test/e2e/internal/cfg"
	"github.com/splunk/splunk-ai-operator/test/e2e/internal/k8s"
	netutil "github.com/splunk/splunk-ai-operator/test/e2e/internal/net"
	pathutil "github.com/splunk/splunk-ai-operator/test/e2e/internal/path"
	"github.com/splunk/splunk-ai-operator/test/utils"
)

var _ = Describe("AIPlatform + AIService (SAIA)", Ordered, func() {
	var pfCancel func()

	BeforeAll(func() {
		By("creating workload namespace")
		Expect(k8s.CreateNamespace(cfg.WorkloadNS)).To(Succeed())

				// Always cleanup, even if any later step fails.
		DeferCleanup(func() {
			// stop any port-forward
			if pfCancel != nil { pfCancel() }
			// delete CRs (best-effort)
			k8s.Delete(cfg.WorkloadNS, cfg.SampleAIService)
			k8s.Delete(cfg.WorkloadNS, cfg.SampleAIPlatform)
			// delete ns
			k8s.DeleteNamespace(cfg.WorkloadNS)
		})

		// baseline PSA (adjust to your policy)
		_ = k8s.LabelNamespace(cfg.WorkloadNS, "pod-security.kubernetes.io/enforce", "baseline")
	})

	AfterAll(func() {
		By("stopping port-forward if running")
		if pfCancel != nil {
			pfCancel()
		}
		By("deleting resources")
		k8s.Delete(cfg.WorkloadNS, cfg.SampleAIService)
		k8s.Delete(cfg.WorkloadNS, cfg.SampleAIPlatform)

		By("deleting workload namespace")
		// Use kubectl directly so this file stays self-contained.
		if root, err := pathutil.RepoRoot(); err == nil {
			_ = os.Chdir(root) // best-effort; DeleteNamespace helper would be cleaner
		}
		_ = k8s.LabelNamespace(cfg.WorkloadNS, "cleanup", "true") // harmless
		// Actually delete the namespace for clean reruns.
		// (We reuse kubectl via k8s.CreateNamespace pattern for consistency.)
		_ = execCmd("kubectl", "delete", "ns", cfg.WorkloadNS)
	})

	It("applies AIPlatform and reaches Ready", func() {
		By("apply AIPlatform sample")
		_, err := k8s.Apply(cfg.WorkloadNS, cfg.SampleAIPlatform)
		Expect(err).NotTo(HaveOccurred())

		By("wait AIPlatform Ready")
		Expect(k8s.WaitCRReady("AIPlatform", cfg.AIPlatformName, cfg.WorkloadNS, cfg.ReadyConditionType, cfg.AIPlatformReadyTimeout)).
			To(Succeed())
	})

	It("ensures SAIA AIService exists and is Ready", func() {
		// Resolve sample path against repo root before Stat (since tests run from test/e2e/specs).
		shouldApplyAIService := true
		if root, err := pathutil.RepoRoot(); err == nil {
			if _, err := os.Stat(filepath.Join(root, cfg.SampleAIService)); err != nil {
				shouldApplyAIService = false
			}
		}

		if shouldApplyAIService {
			By("apply AIService sample")
			_, err := k8s.Apply(cfg.WorkloadNS, cfg.SampleAIService)
			Expect(err).NotTo(HaveOccurred())
		}

		By("wait AIService Ready")
		Expect(k8s.WaitCRReady("AIService", cfg.AIServiceName, cfg.WorkloadNS, cfg.ReadyConditionType, cfg.AIServiceReadyTimeout)).
			To(Succeed())
	})

	It("serves SAIA search via REST", func() {
		By("ensure service endpoints are ready")
		Eventually(func(g Gomega) {
			ok, err := k8s.ServiceHasEndpointPort(cfg.ServiceNamespace, cfg.ServiceToForward, cfg.ForwardRemotePort)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(ok).To(BeTrue())
		}, 5*time.Minute, 3*time.Second).Should(Succeed())

		By("port-forwarding service")
		cancel, err := k8s.PortForwardService(cfg.ServiceNamespace, cfg.ServiceToForward, cfg.ForwardLocalPort, cfg.ForwardRemotePort)
		Expect(err).NotTo(HaveOccurred())
		pfCancel = cancel
		time.Sleep(2 * time.Second)

		By("POST /saia/search")
		status, body, err := netutil.PostJSONLocal(cfg.ForwardLocalPort, cfg.SAIAPOSTPath, cfg.SAIABody, nil)
		_, _ = fmt.Fprintf(GinkgoWriter, "SAIA HTTP %d\nResponse:\n%s\n", status, string(body))
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(BeNumerically(">=", 200))
		Expect(status).To(BeNumerically("<", 300))
		Expect(len(body)).To(BeNumerically(">", 0))

		var js map[string]any
		Expect(json.Unmarshal(body, &js)).To(Succeed())
		Expect(js).NotTo(BeEmpty())
	})
})

// execCmd is a tiny helper to run kubectl without dragging in more imports here.
func execCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if root, err := pathutil.RepoRoot(); err == nil {
		cmd.Dir = root
	}
	_, err := utils.Run(cmd)
	return err
}
