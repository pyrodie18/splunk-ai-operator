package e2e

import (
	"fmt"
	//"os"
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/splunk/splunk-ai-operator/test/e2e/internal/cfg"
	"github.com/splunk/splunk-ai-operator/test/e2e/internal/k8s"
	"github.com/splunk/splunk-ai-operator/test/utils"
)

var isCertManagerAlreadyInstalled bool

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting splunk-ai-operator integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	By("building the manager image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", cfg.ProjectImage))
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager image")

	By("loading image into kind")
	err = utils.LoadImageToKindClusterWithName(cfg.ProjectImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load image into Kind")

	if !cfg.SkipCertManagerInstall {
		By("checking if cert-manager is installed already")
		isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
		if !isCertManagerAlreadyInstalled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager...\n")
			Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: CertManager already installed. Skipping...\n")
		}
	}
})

var _ = AfterSuite(func() {
	if !cfg.SkipCertManagerInstall && !isCertManagerAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling CertManager...\n")
		utils.UninstallCertManager()
	}
})

var _ = ReportAfterSuite("global cleanup", func(report Report) {
	// Make these best-effort nukes; they’ll keep the workspace clean between runs.
	k8s.DeletePod(cfg.OperatorNS, "curl-metrics")
	k8s.DeleteCRB(cfg.MetricsRoleBindName)
	k8s.MakeUndeploy()
	k8s.MakeUninstall()
	k8s.DeleteNamespace(cfg.OperatorNS)
	k8s.DeleteNamespace(cfg.WorkloadNS)
})
