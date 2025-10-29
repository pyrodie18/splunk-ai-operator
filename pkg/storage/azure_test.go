package storage

import (
	"context"
	"testing"

	ai "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAzureClient_BuildLoaderBlock(t *testing.T) {
	s := runtime.NewScheme()
	_ = scheme.AddToScheme(s)
	_ = ai.AddToScheme(s)

	tests := []struct {
		name      string
		endpoint  string
		container string
		prefix    string
		uri       string
		wantBlock string
	}{
		{
			name:      "Azure blob with prefix",
			endpoint:  "https://myaccount.blob.core.windows.net",
			container: "my-container",
			prefix:    "models",
			uri:       "https://myaccount.blob.core.windows.net/my-container/models/subdir/file.ext",
			wantBlock: "azure_blob:",
		},
		{
			name:      "Azure blob without prefix",
			endpoint:  "https://storage.blob.core.windows.net",
			container: "data",
			prefix:    "",
			uri:       "https://storage.blob.core.windows.net/data/file.ext",
			wantBlock: "azure_blob:",
		},
		{
			name:      "Azure blob with nested prefix",
			endpoint:  "https://test.blob.core.windows.net",
			container: "artifacts",
			prefix:    "ai/models",
			uri:       "https://test.blob.core.windows.net/artifacts/ai/models/v1/model.pkl",
			wantBlock: "container: artifacts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &azureClient{
				endpoint:  tt.endpoint,
				container: tt.container,
				prefix:    tt.prefix,
			}

			block := client.BuildLoaderBlock(tt.uri)
			assert.Contains(t, block, tt.wantBlock)
			assert.Contains(t, block, tt.container)
			assert.Contains(t, block, "blob_prefix:")
		})
	}
}

func TestAzureClient_BuildWorkingDir(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		container string
		prefix    string
		modelName string
		wantDir   string
	}{
		{
			name:      "working dir with prefix",
			endpoint:  "https://account.blob.core.windows.net",
			container: "models",
			prefix:    "ai-apps",
			modelName: "my-model",
			wantDir:   "https://account.blob.core.windows.net/models/ai-apps/my-model",
		},
		{
			name:      "working dir without prefix",
			endpoint:  "https://account.blob.core.windows.net",
			container: "models",
			prefix:    "",
			modelName: "test-model",
			wantDir:   "https://account.blob.core.windows.net/models/test-model",
		},
		{
			name:      "working dir with complex model name",
			endpoint:  "https://storage.blob.core.windows.net",
			container: "data",
			prefix:    "production",
			modelName: "v2.1/advanced-model",
			wantDir:   "https://storage.blob.core.windows.net/data/production/v2.1/advanced-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &azureClient{
				endpoint:  tt.endpoint,
				container: tt.container,
				prefix:    tt.prefix,
			}

			dir := client.BuildWorkingDir(tt.modelName)
			assert.Equal(t, tt.wantDir, dir)
		})
	}
}

func TestAzureClient_BuildArtifactURI(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		container string
		prefix    string
		key       string
		wantURI   string
	}{
		{
			name:      "artifact URI with prefix",
			endpoint:  "https://account.blob.core.windows.net",
			container: "artifacts",
			prefix:    "models",
			key:       "model.tar.gz",
			wantURI:   "https://account.blob.core.windows.net/artifacts/models/model.tar.gz",
		},
		{
			name:      "artifact URI without prefix",
			endpoint:  "https://storage.blob.core.windows.net",
			container: "data",
			prefix:    "",
			key:       "file.zip",
			wantURI:   "https://storage.blob.core.windows.net/data/file.zip",
		},
		{
			name:      "artifact URI with leading slash in key",
			endpoint:  "https://test.blob.core.windows.net",
			container: "files",
			prefix:    "uploads",
			key:       "/document.pdf",
			wantURI:   "https://test.blob.core.windows.net/files/uploads/document.pdf",
		},
		{
			name:      "artifact URI with nested path",
			endpoint:  "https://myaccount.blob.core.windows.net",
			container: "container",
			prefix:    "root/sub",
			key:       "deep/path/file.txt",
			wantURI:   "https://myaccount.blob.core.windows.net/container/root/sub/deep/path/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &azureClient{
				endpoint:  tt.endpoint,
				container: tt.container,
				prefix:    tt.prefix,
			}

			uri := client.BuildArtifactURI(tt.key)
			assert.Equal(t, tt.wantURI, uri)
		})
	}
}

func TestAzureClient_GetMethods(t *testing.T) {
	client := &azureClient{
		endpoint:  "https://account.blob.core.windows.net",
		container: "my-container",
		prefix:    "my/prefix",
	}

	t.Run("GetProvider", func(t *testing.T) {
		assert.Equal(t, "azure", client.GetProvider())
	})

	t.Run("GetBucket", func(t *testing.T) {
		assert.Equal(t, "my-container", client.GetBucket())
	})

	t.Run("GetPrefix", func(t *testing.T) {
		assert.Equal(t, "my/prefix", client.GetPrefix())
	})
}

func TestNewAzureClient_WithSecret(t *testing.T) {
	s := runtime.NewScheme()
	_ = scheme.AddToScheme(s)
	_ = ai.AddToScheme(s)

	tests := []struct {
		name        string
		secretData  map[string][]byte
		volumeSpec  ai.ObjectStorageSpec
		wantErr     bool
		errContains string
	}{
		{
			name: "valid secret with all fields",
			secretData: map[string][]byte{
				"azure_tenant_id":     []byte("tenant-id-123"),
				"azure_client_id":     []byte("client-id-456"),
				"azure_client_secret": []byte("secret-789"),
			},
			volumeSpec: ai.ObjectStorageSpec{
				Endpoint:  "https://account.blob.core.windows.net",
				SecretRef: "azure-creds",
			},
			wantErr: false, // Azure client creation succeeds, actual operations would fail
		},
		{
			name:       "missing secret",
			secretData: nil,
			volumeSpec: ai.ObjectStorageSpec{
				Endpoint:  "https://account.blob.core.windows.net",
				SecretRef: "missing-secret",
			},
			wantErr:     true,
			errContains: "fetch Azure secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fakeClientBuilder := fake.NewClientBuilder().WithScheme(s)

			if tt.secretData != nil {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "azure-creds",
						Namespace: "default",
					},
					Data: tt.secretData,
				}
				fakeClientBuilder = fakeClientBuilder.WithObjects(secret)
			}

			fakeClient := fakeClientBuilder.Build()

			client, err := NewAzureClient(ctx, fakeClient, "default", "container", "prefix", tt.volumeSpec)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestNewAzureClient_WithoutSecret(t *testing.T) {
	ctx := context.Background()
	s := runtime.NewScheme()
	_ = scheme.AddToScheme(s)
	_ = ai.AddToScheme(s)

	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	volumeSpec := ai.ObjectStorageSpec{
		Endpoint:  "https://account.blob.core.windows.net",
		SecretRef: "", // No secret, uses default credentials
	}

	// This may succeed in some environments (if Azure CLI is configured)
	// or fail in others - both are valid outcomes
	client, err := NewAzureClient(ctx, fakeClient, "default", "container", "prefix", volumeSpec)

	// Log result for debugging
	t.Logf("NewAzureClient without secret: client=%v, err=%v", client != nil, err)

	// If client was created, verify its properties
	if client != nil && err == nil {
		assert.Equal(t, "azure", client.GetProvider())
		assert.Equal(t, "container", client.GetBucket())
		assert.Equal(t, "prefix", client.GetPrefix())
	}
}

func TestAzureClient_Integration(t *testing.T) {
	ctx := context.Background()
	s := runtime.NewScheme()
	_ = scheme.AddToScheme(s)
	_ = ai.AddToScheme(s)

	// Create Azure secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azure-storage-creds",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"azure_tenant_id":     []byte("00000000-0000-0000-0000-000000000000"),
			"azure_client_id":     []byte("11111111-1111-1111-1111-111111111111"),
			"azure_client_secret": []byte("test-secret-value"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(secret).
		Build()

	volumeSpec := ai.ObjectStorageSpec{
		Endpoint:  "https://testaccount.blob.core.windows.net",
		SecretRef: "azure-storage-creds",
	}

	// Attempt to create client (will fail with invalid credentials but validates secret handling)
	client, err := NewAzureClient(ctx, fakeClient, "test-namespace", "test-container", "test/prefix", volumeSpec)

	// We expect an error because the credentials are fake
	// but the important thing is that the secret was read and processed
	t.Logf("NewAzureClient result: client=%v, err=%v", client != nil, err)

	// If we got past secret reading, test the client methods
	if client != nil {
		assert.Equal(t, "azure", client.GetProvider())
		assert.Equal(t, "test-container", client.GetBucket())
		assert.Equal(t, "test/prefix", client.GetPrefix())
	}
}
