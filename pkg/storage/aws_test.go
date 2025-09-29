package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
)

// ---- Mock S3 Client ----
type mockS3Client struct {
	s3iface.S3API
	HeadErr  error
	ListKeys []string
	ListErr  error
}

func (m *mockS3Client) HeadObjectWithContext(ctx aws.Context, input *s3.HeadObjectInput, opts ...request.Option) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{}, m.HeadErr
}

func (m *mockS3Client) ListObjectsV2PagesWithContext(ctx aws.Context, input *s3.ListObjectsV2Input, fn func(*s3.ListObjectsV2Output, bool) bool, opts ...request.Option) error {
	if m.ListErr != nil {
		return m.ListErr
	}
	page := &s3.ListObjectsV2Output{}
	for _, key := range m.ListKeys {
		page.Contents = append(page.Contents, &s3.Object{Key: aws.String(key)})
	}
	fn(page, true)
	return nil
}

// ---- Tests ----

func TestExists(t *testing.T) {
	ctx := context.Background()

	t.Run("object exists", func(t *testing.T) {
		mock := &mockS3Client{
			HeadErr: nil, // no error => object exists
		}
		c := &s3Client{cli: mock, bucket: "my-bucket", prefix: "myprefix"}

		exists, err := c.Exists(ctx, "some-key")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("object missing returns false", func(t *testing.T) {
		mock := &mockS3Client{
			HeadErr: awserr.NewRequestFailure(
				awserr.New("NotFound", "not found", nil), 404, "reqid",
			),
		}
		c := &s3Client{cli: mock, bucket: "my-bucket", prefix: "myprefix"}

		exists, err := c.Exists(ctx, "missing-key")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("unexpected error returns error", func(t *testing.T) {
		mock := &mockS3Client{
			HeadErr: errors.New("some network issue"),
		}
		c := &s3Client{cli: mock, bucket: "my-bucket", prefix: "myprefix"}

		exists, err := c.Exists(ctx, "foo")
		assert.Error(t, err)
		assert.False(t, exists)
	})
}

func TestListObjects(t *testing.T) {
	ctx := context.Background()

	t.Run("list multiple keys", func(t *testing.T) {
		mock := &mockS3Client{
			ListKeys: []string{"a.txt", "b.txt", "c.txt"},
		}
		c := &s3Client{cli: mock, bucket: "bucket", prefix: "prefix"}

		keys, err := c.ListObjects(ctx)
		assert.NoError(t, err)
		assert.Equal(t, []string{"a.txt", "b.txt", "c.txt"}, keys)
	})

	t.Run("list returns error", func(t *testing.T) {
		mock := &mockS3Client{
			ListErr: errors.New("s3 list failed"),
		}
		c := &s3Client{cli: mock, bucket: "bucket", prefix: "prefix"}

		keys, err := c.ListObjects(ctx)
		assert.Error(t, err)
		assert.Nil(t, keys)
	})
}

func TestBuildLoaderBlock(t *testing.T) {
	c := &s3Client{bucket: "my-bucket", prefix: "myprefix"}

	uri := "s3://my-bucket/myprefix/model/file.zip"
	got := c.BuildLoaderBlock(uri)

	expected := `        s3_artifact:
          bucket: my-bucket
          s3_key_prefix: myprefix/model
`
	assert.Equal(t, expected, got)
}

func TestBuildWorkingDir(t *testing.T) {
	t.Run("with prefix", func(t *testing.T) {
		c := &s3Client{bucket: "bucket", prefix: "myprefix"}
		got := c.BuildWorkingDir("my-model")
		assert.Equal(t, "s3://bucket/myprefix/my-model", got)
	})

	t.Run("no prefix", func(t *testing.T) {
		c := &s3Client{bucket: "bucket", prefix: ""}
		got := c.BuildWorkingDir("my-model")
		assert.Equal(t, "s3://bucket/my-model", got)
	})
}

func TestBuildArtifactURI(t *testing.T) {
	t.Run("with prefix", func(t *testing.T) {
		c := &s3Client{bucket: "bucket", prefix: "myprefix"}
		got := c.BuildArtifactURI("/model.zip")
		assert.Equal(t, "s3://bucket/myprefix/model.zip", got)
	})

	t.Run("no prefix", func(t *testing.T) {
		c := &s3Client{bucket: "bucket", prefix: ""}
		got := c.BuildArtifactURI("model.zip")
		assert.Equal(t, "s3://bucket/model.zip", got)
	})
}

func TestGetProviderBucketPrefix(t *testing.T) {
	c := &s3Client{bucket: "bucket", prefix: "myprefix"}
	assert.Equal(t, "s3", c.GetProvider())
	assert.Equal(t, "bucket", c.GetBucket())
	assert.Equal(t, "myprefix", c.GetPrefix())
}
