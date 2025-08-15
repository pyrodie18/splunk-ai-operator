package raybuilder

import (
	"context"
	"errors"
	"os"
	"testing"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func TestReadInstanceMapFromConfigMap_Success(t *testing.T) {
	instanceYaml := `
foo:
  details:
    image: "rayproject/ray:latest"
    cpu: 2
baz:
  details:
    image: "rayproject/ray:latest"
    cpu: 4
`
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "default"},
		Data:       map[string]string{"instance.yaml": instanceYaml},
	}
	cl := fake.NewClientBuilder().WithObjects(cm).Build()
	got, err := ReadInstanceMapFromConfigMap(context.Background(), cl, "test-cm", "default")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil {
		t.Fatalf("expected instanceMap, got nil")
	}
	// Optionally, check the contents
	if _, ok := got["foo"]["details"]; !ok {
		t.Fatalf("expected foo.details in instanceMap")
	}
	if _, ok := got["baz"]["details"]; !ok {
		t.Fatalf("expected baz.details in instanceMap")
	}
}

func TestReadInstanceMapFromConfigMap_NotFound(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "default"},
		Data:       map[string]string{},
	}
	cl := fake.NewClientBuilder().WithObjects(cm).Build()
	_, err := ReadInstanceMapFromConfigMap(context.Background(), cl, "test-cm", "default")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestReadInstanceMapFromConfigMap_BadYaml(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "default"},
		Data:       map[string]string{"instance.yaml": "bad: [unclosed"},
	}
	cl := fake.NewClientBuilder().WithObjects(cm).Build()
	_, err := ReadInstanceMapFromConfigMap(context.Background(), cl, "test-cm", "default")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
type fakeBuilder struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func TestReconcileInstancesConfigMap_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "instance.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	content := "foo: bar"
	tmpFile.WriteString(content)
	tmpFile.Close()
	os.Setenv("INSTANCE_FILE", tmpFile.Name())
	defer os.Unsetenv("INSTANCE_FILE")

	s := k8sscheme.Scheme
	p := &aiApi.AIPlatform{ObjectMeta: metav1.ObjectMeta{Name: "plat", Namespace: "default"}}
	b := &Builder{Client: fake.NewClientBuilder().WithScheme(s).Build(), Scheme: s}
	utilruntime.Must(aiApi.AddToScheme(s))
	err = b.ReconcileInstancesConfigMap(context.Background(), p)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestReconcileInstancesConfigMap_FileNotFound(t *testing.T) {
	os.Setenv("INSTANCE_FILE", "/nonexistent/file.yaml")
	defer os.Unsetenv("INSTANCE_FILE")
	s := k8sscheme.Scheme
	p := &aiApi.AIPlatform{ObjectMeta: metav1.ObjectMeta{Name: "plat", Namespace: "default"}}
	b := &Builder{Client: fake.NewClientBuilder().WithScheme(s).Build(), Scheme: s}
	err := b.ReconcileInstancesConfigMap(context.Background(), p)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDetectProvider_AWS(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{"eks.amazonaws.com/nodegroup": "ng1"},
		},
	}
	cl := fake.NewClientBuilder().WithObjects(node).Build()
	provider, err := detectProvider(cl, context.Background())
	if err != nil || provider != "aws" {
		t.Fatalf("expected aws, got %v, err: %v", provider, err)
	}
}

func TestDetectProvider_GCP(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{"cloud.google.com/gke-nodepool": "np1"},
		},
	}
	cl := fake.NewClientBuilder().WithObjects(node).Build()
	provider, err := detectProvider(cl, context.Background())
	if err != nil || provider != "gcp" {
		t.Fatalf("expected gcp, got %v, err: %v", provider, err)
	}
}

func TestDetectProvider_Azure(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{"kubernetes.azure.com/cluster": "az1"},
		},
	}
	cl := fake.NewClientBuilder().WithObjects(node).Build()
	provider, err := detectProvider(cl, context.Background())
	if err != nil || provider != "azure" {
		t.Fatalf("expected azure, got %v, err: %v", provider, err)
	}
}

func TestDetectProvider_Oracle(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "oke-node1",
			Labels: map[string]string{"oke.oraclecloud.com/name": "oke"},
		},
	}
	cl := fake.NewClientBuilder().WithObjects(node).Build()
	provider, err := detectProvider(cl, context.Background())
	if err != nil || provider != "oracle" {
		t.Fatalf("expected oracle, got %v, err: %v", provider, err)
	}
}

func TestDetectProvider_Generic(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{},
		},
	}
	cl := fake.NewClientBuilder().WithObjects(node).Build()
	provider, err := detectProvider(cl, context.Background())
	if err != nil || provider != "generic" {
		t.Fatalf("expected generic, got %v, err: %v", provider, err)
	}
}

func TestDetectProvider_Error(t *testing.T) {
	cl := &errorClient{}
	_, err := detectProvider(cl, context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// errorClient implements client.Client and always returns error for List
type errorClient struct {
	client.Client
}

func (e *errorClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	return errors.New("list error")
}
