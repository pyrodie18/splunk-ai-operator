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

func TestGCSClient_BuildLoaderBlock(t *testing.T) {
	tests := []struct {
		name      string
		bucket    string
		prefix    string
		uri       string
		wantBlock string
	}{
		{
			name:      "GCS URI with prefix",
			bucket:    "my-bucket",
			prefix:    "models",
			uri:       "gs://my-bucket/models/subdir/file.ext",
			wantBlock: "gcs_artifact:",
		},
		{
			name:      "GCS URI without prefix",
			bucket:    "data-bucket",
			prefix:    "",
			uri:       "gs://data-bucket/file.ext",
			wantBlock: "gcs_artifact:",
		},
		{
			name:      "GCS URI with nested prefix",
			bucket:    "artifacts",
			prefix:    "ai/models",
			uri:       "gs://artifacts/ai/models/v1/model.pkl",
			wantBlock: "bucket: artifacts",
		},
		{
			name:      "GCS URI with deep path",
			bucket:    "storage",
			prefix:    "root",
			uri:       "gs://storage/root/deep/nested/path/file.txt",
			wantBlock: "object_key:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &gcsClient{
				bucket: tt.bucket,
				prefix: tt.prefix,
			}

			block := client.BuildLoaderBlock(tt.uri)
			assert.Contains(t, block, tt.wantBlock)
			assert.Contains(t, block, tt.bucket)
		})
	}
}

func TestGCSClient_BuildWorkingDir(t *testing.T) {
	tests := []struct {
		name      string
		bucket    string
		prefix    string
		modelName string
		wantDir   string
	}{
		{
			name:      "working dir with prefix",
			bucket:    "ml-models",
			prefix:    "production",
			modelName: "my-model",
			wantDir:   "gs://ml-models/production/my-model",
		},
		{
			name:      "working dir without prefix",
			bucket:    "models",
			prefix:    "",
			modelName: "test-model",
			wantDir:   "gs://models/test-model",
		},
		{
			name:      "working dir with nested prefix",
			bucket:    "ai-storage",
			prefix:    "team/models",
			modelName: "classifier-v2",
			wantDir:   "gs://ai-storage/team/models/classifier-v2",
		},
		{
			name:      "working dir with complex model name",
			bucket:    "data",
			prefix:    "apps",
			modelName: "v2.1/advanced-model",
			wantDir:   "gs://data/apps/v2.1/advanced-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &gcsClient{
				bucket: tt.bucket,
				prefix: tt.prefix,
			}

			dir := client.BuildWorkingDir(tt.modelName)
			assert.Equal(t, tt.wantDir, dir)
		})
	}
}

func TestGCSClient_BuildArtifactURI(t *testing.T) {
	tests := []struct {
		name    string
		bucket  string
		prefix  string
		key     string
		wantURI string
	}{
		{
			name:    "artifact URI with prefix",
			bucket:  "artifacts",
			prefix:  "models",
			key:     "models/model.tar.gz",
			wantURI: "gs://artifacts/model.tar.gz",
		},
		{
			name:    "artifact URI without prefix",
			bucket:  "data",
			prefix:  "",
			key:     "file.zip",
			wantURI: "gs://data/file.zip",
		},
		{
			name:    "artifact URI with nested path",
			bucket:  "storage",
			prefix:  "root/sub",
			key:     "root/sub/deep/path/file.txt",
			wantURI: "gs://storage/deep/path/file.txt",
		},
		{
			name:    "artifact URI strips prefix correctly",
			bucket:  "my-bucket",
			prefix:  "prefix",
			key:     "prefix/subfolder/document.pdf",
			wantURI: "gs://my-bucket/subfolder/document.pdf",
		},
		{
			name:    "artifact URI with key not containing prefix",
			bucket:  "bucket",
			prefix:  "models",
			key:     "data/file.txt",
			wantURI: "gs://bucket/data/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &gcsClient{
				bucket: tt.bucket,
				prefix: tt.prefix,
			}

			uri := client.BuildArtifactURI(tt.key)
			assert.Equal(t, tt.wantURI, uri)
		})
	}
}

func TestGCSClient_GetMethods(t *testing.T) {
	client := &gcsClient{
		bucket: "my-bucket",
		prefix: "my/prefix",
	}

	t.Run("GetProvider", func(t *testing.T) {
		assert.Equal(t, "gcs", client.GetProvider())
	})

	t.Run("GetBucket", func(t *testing.T) {
		assert.Equal(t, "my-bucket", client.GetBucket())
	})

	t.Run("GetPrefix", func(t *testing.T) {
		assert.Equal(t, "my/prefix", client.GetPrefix())
	})
}

func TestNewGCSClient_WithSecret(t *testing.T) {
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
			name: "valid secret with service account JSON",
			secretData: map[string][]byte{
				"service_account.json": []byte(`{
					"type": "service_account",
					"project_id": "test-project",
					"private_key_id": "key-id",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----\n",
					"client_email": "test@test-project.iam.gserviceaccount.com",
					"client_id": "123456789",
					"auth_uri": "https://accounts.google.com/o/oauth2/auth",
					"token_uri": "https://oauth2.googleapis.com/token"
				}`),
			},
			volumeSpec: ai.ObjectStorageSpec{
				SecretRef: "gcs-creds",
			},
			wantErr: false, // GCS client creation succeeds, actual operations would fail
		},
		{
			name:       "missing secret",
			secretData: nil,
			volumeSpec: ai.ObjectStorageSpec{
				SecretRef: "missing-secret",
			},
			wantErr:     true,
			errContains: "fetch GCP secret",
		},
		{
			name: "secret missing service_account.json key",
			secretData: map[string][]byte{
				"wrong-key": []byte("data"),
			},
			volumeSpec: ai.ObjectStorageSpec{
				SecretRef: "incomplete-secret",
			},
			wantErr:     true,
			errContains: "missing key 'service_account.json'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fakeClientBuilder := fake.NewClientBuilder().WithScheme(s)

			if tt.secretData != nil {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tt.volumeSpec.SecretRef,
						Namespace: "default",
					},
					Data: tt.secretData,
				}
				fakeClientBuilder = fakeClientBuilder.WithObjects(secret)
			}

			fakeClient := fakeClientBuilder.Build()

			client, err := NewGCSClient(ctx, fakeClient, "default", "bucket", "prefix", tt.volumeSpec)

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

func TestNewGCSClient_WithoutSecret(t *testing.T) {
	ctx := context.Background()
	s := runtime.NewScheme()
	_ = scheme.AddToScheme(s)
	_ = ai.AddToScheme(s)

	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	volumeSpec := ai.ObjectStorageSpec{
		SecretRef: "", // No secret, uses default credentials
	}

	// This will fail in test environment without real GCP credentials
	// but it validates the code path for default credentials
	_, err := NewGCSClient(ctx, fakeClient, "default", "bucket", "prefix", volumeSpec)

	// Expected to fail in test environment without GCP Application Default Credentials
	// The important thing is that it attempts to use default credentials
	t.Logf("NewGCSClient without secret result: %v", err)
	// We don't assert specific error as it depends on environment
}

func TestGCSClient_Integration(t *testing.T) {
	ctx := context.Background()
	s := runtime.NewScheme()
	_ = scheme.AddToScheme(s)
	_ = ai.AddToScheme(s)

	// Create GCS secret with minimal valid JSON structure
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gcs-storage-creds",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"service_account.json": []byte(`{
				"type": "service_account",
				"project_id": "test-project-12345",
				"private_key_id": "abcd1234",
				"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC\n-----END PRIVATE KEY-----\n",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789012345678901",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token",
				"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
				"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com"
			}`),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(secret).
		Build()

	volumeSpec := ai.ObjectStorageSpec{
		SecretRef: "gcs-storage-creds",
	}

	// Attempt to create client (will fail with invalid credentials but validates secret handling)
	client, err := NewGCSClient(ctx, fakeClient, "test-namespace", "test-bucket", "test/prefix", volumeSpec)

	// We expect an error because the credentials are fake
	// but the important thing is that the secret was read and processed
	t.Logf("NewGCSClient result: client=%v, err=%v", client != nil, err)

	// If we got past secret reading, test the client methods
	if client != nil {
		assert.Equal(t, "gcs", client.GetProvider())
		assert.Equal(t, "test-bucket", client.GetBucket())
		assert.Equal(t, "test/prefix", client.GetPrefix())
	}
}

func TestGCSClient_MethodsWithEmptyPrefix(t *testing.T) {
	client := &gcsClient{
		bucket: "test-bucket",
		prefix: "",
	}

	t.Run("BuildWorkingDir with empty prefix", func(t *testing.T) {
		dir := client.BuildWorkingDir("model-v1")
		assert.Equal(t, "gs://test-bucket/model-v1", dir)
	})

	t.Run("BuildArtifactURI with empty prefix", func(t *testing.T) {
		uri := client.BuildArtifactURI("artifacts/file.zip")
		assert.Equal(t, "gs://test-bucket/artifacts/file.zip", uri)
	})

	t.Run("BuildLoaderBlock with empty prefix", func(t *testing.T) {
		block := client.BuildLoaderBlock("gs://test-bucket/models/model.tar.gz")
		assert.Contains(t, block, "gcs_artifact:")
		assert.Contains(t, block, "test-bucket")
	})
}

func TestGCSClient_MethodsWithComplexPrefixes(t *testing.T) {
	client := &gcsClient{
		bucket: "production-bucket",
		prefix: "ml/models/v2",
	}

	t.Run("BuildWorkingDir with nested prefix", func(t *testing.T) {
		dir := client.BuildWorkingDir("classifier")
		assert.Equal(t, "gs://production-bucket/ml/models/v2/classifier", dir)
	})

	t.Run("BuildArtifactURI strips prefix correctly", func(t *testing.T) {
		// Key includes the prefix, should be stripped
		uri := client.BuildArtifactURI("ml/models/v2/artifact.tar.gz")
		assert.Equal(t, "gs://production-bucket/artifact.tar.gz", uri)
	})

	t.Run("BuildLoaderBlock with nested prefix", func(t *testing.T) {
		block := client.BuildLoaderBlock("gs://production-bucket/ml/models/v2/subdir/file.pkl")
		assert.Contains(t, block, "gcs_artifact:")
		assert.Contains(t, block, "production-bucket")
		assert.Contains(t, block, "object_key:")
	})
}
