// controllers/storage/s3.go
package storage

import (
	"context"
	"fmt"
	"path"
	"strings"

	ai "github.com/splunk/splunk-ai-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// s3Client implements StorageClient for AWS S3.
type fixtureClient struct {
	bucket string
	prefix string
}

func NewFixtureClient(
	k8sClient client.Client,
	namespace, bucket, prefix string,
	vs ai.ObjectStorageSpec,
) (StorageClient, error) {
	// Use env var or fallback default cert path
	return &fixtureClient{
		bucket: bucket,
		prefix: prefix,
	}, nil
}

// ListObjects returns all object keys under the configured prefix, across all pages.
func (c *fixtureClient) ListObjects(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (c *fixtureClient) Exists(ctx context.Context, key string) (bool, error) {
	return true, nil
}

// BuildLoaderBlock returns the `model_loader` YAML snippet for this URI.
func (c *fixtureClient) BuildLoaderBlock(uri string) string {
	// uri is "s3://bucket/prefix/.../file"
	trim := fmt.Sprintf("s3://%s/", c.bucket)
	p := strings.TrimPrefix(uri, trim)
	dir := path.Dir(p)
	return fmt.Sprintf(`        s3_artifact:
          bucket: %s
          s3_key_prefix: %s
`, c.bucket, dir)
}

// BuildWorkingDir returns the working_dir URI for the application ZIP.
func (c *fixtureClient) BuildWorkingDir(modelName string) string {
	if c.prefix == "" {
		return fmt.Sprintf("s3://%s/%s", c.bucket, modelName)
	}
	return fmt.Sprintf("s3://%s/%s/%s", c.bucket, c.prefix, modelName)
}

// BuildArtifactURI builds a “s3://bucket[/prefix]/key” URI.
func (c *fixtureClient) BuildArtifactURI(key string) string {
	// strip any leading slash on key
	k := strings.TrimPrefix(key, "/")
	if c.prefix != "" {
		return fmt.Sprintf("s3://%s/%s/%s", c.bucket, c.prefix, k)
	}
	return fmt.Sprintf("s3://%s/%s", c.bucket, k)
}

func (c *fixtureClient) GetProvider() string { return "s3" }
func (c *fixtureClient) GetBucket() string   { return c.bucket }
func (c *fixtureClient) GetPrefix() string   { return c.prefix }
