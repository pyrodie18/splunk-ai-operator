package splunkutils

import (
	"context"
	"os"
	"strings"
	"testing"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestVaultFileResolver_GetHECToken(t *testing.T) {
	ctx := context.Background()
	resolver := &VaultFileResolver{}

	t.Run("error: VaultFilePath missing", func(t *testing.T) {
		cfg := &aiApi.SplunkConfigurationSpec{} // no VaultFilePath

		token, err := resolver.GetHECToken(ctx, "test-ns", cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "VaultFilePath must be provided")
		assert.Empty(t, token)
	})

	t.Run("error: VaultFilePath points to non-existent file", func(t *testing.T) {
		cfg := &aiApi.SplunkConfigurationSpec{
			VaultFilePath: "/tmp/non-existent-file-12345",
		}

		token, err := resolver.GetHECToken(ctx, "test-ns", cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read Vault-injected token file")
		assert.Empty(t, token)
	})

	t.Run("error: Vault file exists but is empty", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "vault-empty-*")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name()) // cleanup after test

		// Write nothing → empty file
		cfg := &aiApi.SplunkConfigurationSpec{
			VaultFilePath: tmpFile.Name(),
		}

		token, err := resolver.GetHECToken(ctx, "test-ns", cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is empty")
		assert.Empty(t, token)
	})

	t.Run("success: Vault file exists with token", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "vault-token-*")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		expectedToken := "super-secret-token"
		_, err = tmpFile.WriteString(expectedToken + "\n") // simulate trailing newline
		assert.NoError(t, err)
		tmpFile.Close()

		cfg := &aiApi.SplunkConfigurationSpec{
			VaultFilePath: tmpFile.Name(),
		}

		token, err := resolver.GetHECToken(ctx, "test-ns", cfg)

		assert.NoError(t, err)
		// Trimmed correctly
		assert.Equal(t, expectedToken, token)
		assert.False(t, strings.HasSuffix(token, "\n"))
	})
}
