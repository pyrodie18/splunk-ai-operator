package common

import (
	"context"
	"fmt"
	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"os"
	"strings"
)

type VaultFileResolver struct{}

func (r *VaultFileResolver) GetHECToken(ctx context.Context, namespace string, cfg *aiApi.SplunkConfigurationSpec) (string, error) {
	if cfg.VaultFilePath == "" {
		return "", fmt.Errorf("VaultFilePath must be provided for SecretSource=vault")
	}

	data, err := os.ReadFile(cfg.VaultFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read Vault-injected token file %q: %w", cfg.VaultFilePath, err)
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("vault-injected file %q is empty", cfg.VaultFilePath)
	}
	return token, nil
}
