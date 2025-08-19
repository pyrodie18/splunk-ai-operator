package k8s

import (
	"os/exec"

	//pathutil "github.com/splunk/splunk-ai-operator/test/e2e/internal/path"
	"github.com/splunk/splunk-ai-operator/test/utils"
)


// Best-effort deletes below. No error if missing.

func DeleteNamespace(ns string) {
	cmd := exec.Command("kubectl", "delete", "ns", ns, "--ignore-not-found=true")
	if err := withRepoDir(cmd); err == nil {
		_, _ = utils.Run(cmd)
	}
}

func DeleteCRB(name string) {
	cmd := exec.Command("kubectl", "delete", "clusterrolebinding", name, "--ignore-not-found=true")
	if err := withRepoDir(cmd); err == nil {
		_, _ = utils.Run(cmd)
	}
}

func DeletePod(ns, name string) {
	cmd := exec.Command("kubectl", "delete", "pod", name, "-n", ns, "--ignore-not-found=true")
	if err := withRepoDir(cmd); err == nil {
		_, _ = utils.Run(cmd)
	}
}

func MakeUndeploy() {
	cmd := exec.Command("make", "undeploy")
	if err := withRepoDir(cmd); err == nil {
		_, _ = utils.Run(cmd)
	}
}

func MakeUninstall() {
	cmd := exec.Command("make", "uninstall")
	if err := withRepoDir(cmd); err == nil {
		_, _ = utils.Run(cmd)
	}
}
