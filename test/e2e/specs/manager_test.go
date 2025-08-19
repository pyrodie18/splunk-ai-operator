package e2e

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/splunk/splunk-ai-operator/test/e2e/internal/cfg"
	"github.com/splunk/splunk-ai-operator/test/e2e/internal/k8s"
	pathutil "github.com/splunk/splunk-ai-operator/test/e2e/internal/path"
	"github.com/splunk/splunk-ai-operator/test/utils"
)

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string



	BeforeAll(func() {
		By("creating manager namespace")
		Expect(k8s.CreateNamespace(cfg.OperatorNS)).To(Succeed())

		// Always try to clean this namespace & RBAC, even if later steps fail.
		DeferCleanup(func() {
			k8s.DeletePod(cfg.OperatorNS, "curl-metrics")
			k8s.DeleteCRB(cfg.MetricsRoleBindName)
			k8s.MakeUndeploy()
			k8s.MakeUninstall()
			k8s.DeleteNamespace(cfg.OperatorNS)
		})

		By("labeling namespace restricted")
		Expect(k8s.LabelNamespace(cfg.OperatorNS, "pod-security.kubernetes.io/enforce", "restricted")).To(Succeed())

		By("installing CRDs")
		cmd := exec.Command("make", "install")
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		} 
		fmt.Printf("Running command: %s", cmd.String())
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("deploying controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", cfg.ProjectImage))
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", cfg.OperatorNS)
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		_, _ = utils.Run(cmd)

		cmd = exec.Command("make", "undeploy")
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		_, _ = utils.Run(cmd)

		cmd = exec.Command("make", "uninstall")
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		_, _ = utils.Run(cmd)

		cmd = exec.Command("kubectl", "delete", "ns", cfg.OperatorNS)
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() && controllerPodName != "" {
			if logs, err := k8s.GetLogs(cfg.OperatorNS, controllerPodName); err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n%s\n", logs)
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	It("controller-manager pod is Running", func() {
		verify := func(g Gomega) {
			pod, err := k8s.GetControllerPodName(cfg.OperatorNS)
			g.Expect(err).NotTo(HaveOccurred())
			controllerPodName = pod

			phase, err := k8s.PodPhase(cfg.OperatorNS, pod)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(phase).To(Equal("Running"))
		}
		Eventually(verify).Should(Succeed())
	})

	It("serves metrics", func() {
		By("creating RBAC for metrics")
		cmd := exec.Command("kubectl", "create", "clusterrolebinding", cfg.MetricsRoleBindName,
			"--clusterrole=splunk-ai-operator-metrics-reader",
			fmt.Sprintf("--serviceaccount=%s:%s", cfg.OperatorNS, cfg.ServiceAccountName))
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("checking service exists")
		cmd = exec.Command("kubectl", "get", "service", cfg.MetricsServiceName, "-n", cfg.OperatorNS)
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for endpoint port 8443")
		Eventually(func(g Gomega) {
			ok, err := k8s.ServiceHasEndpointPort(cfg.OperatorNS, cfg.MetricsServiceName, "8443")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(ok).To(BeTrue())
		}).Should(Succeed())

		By("creating curl pod that fetches metrics")
		cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
			"--namespace", cfg.OperatorNS,
			"--image=curlimages/curl:latest",
			"--", "sh", "-c", "curl -vk https://"+cfg.MetricsServiceName+"."+cfg.OperatorNS+".svc.cluster.local:8443/metrics")
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			cmd = exec.Command("kubectl", "get", "pods", "curl-metrics", "-n", cfg.OperatorNS, "-o", "jsonpath={.status.phase}")
			if root, err := pathutil.RepoRoot(); err == nil {
				cmd.Dir = root
			}
			out, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out).To(Equal("Succeeded"))
		}, 5*time.Minute).Should(Succeed())

		cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", cfg.OperatorNS)
		if root, err := pathutil.RepoRoot(); err == nil {
			cmd.Dir = root
		}
		out, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("controller_runtime_reconcile_total"))
	})
})
