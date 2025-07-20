package ai_platform

import (
	"context"
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
)

func buildTestScheme(t *testing.T) *runtime.Scheme {
	s := runtime.NewScheme()
	err := aiApi.AddToScheme(s)
	assert.NoError(t, err)

	err = certmanagerv1.AddToScheme(s)
	assert.NoError(t, err)

	err = corev1.AddToScheme(s)
	assert.NoError(t, err)

	return s
}

func TestReconcileCertificate_CreatesOrUpdatesCert(t *testing.T) {
	ctx := context.Background()
	scheme := buildTestScheme(t)

	// ✅ Create fake AIPlatform CR
	aiPlatform := &aiApi.AIPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ai",
			Namespace: "default",
		},
		Spec: aiApi.AIPlatformSpec{
			CertificateRef: "test-cluster-issuer",
			ClusterDomain:  "cluster.local",
		},
	}

	// ✅ Build fake client (no Certificate exists initially)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(aiPlatform).
		Build()

	reconciler := &AIPlatformReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// ✅ Call reconcileCertificate
	err := reconciler.reconcileCertificate(ctx, aiPlatform)
	assert.NoError(t, err)

	// ✅ Verify Certificate is created with expected fields
	cert := &certmanagerv1.Certificate{}
	certKey := client.ObjectKey{Name: "test-ai-tls", Namespace: "default"}
	err = fakeClient.Get(ctx, certKey, cert)
	assert.NoError(t, err, "Certificate should have been created")

	assert.Equal(t, "test-ai-tls-secret", cert.Spec.SecretName)
	assert.Equal(t, "test-cluster-issuer", cert.Spec.IssuerRef.Name)
	assert.Equal(t, "ClusterIssuer", cert.Spec.IssuerRef.Kind)

	expectedDNS := "test-ai.default.svc.cluster.local"
	assert.Contains(t, cert.Spec.DNSNames, expectedDNS)

	// ✅ Ensure owner reference is set correctly
	foundOwner := false
	for _, owner := range cert.OwnerReferences {
		if owner.Name == "test-ai" && owner.Kind == "AIPlatform" {
			foundOwner = true
		}
	}
	assert.True(t, foundOwner, "Certificate should have AIPlatform owner reference")
}

func TestReconcileCertificate_UpdatesExistingCert(t *testing.T) {
	ctx := context.Background()
	scheme := buildTestScheme(t)

	// ✅ AIPlatform CR
	aiPlatform := &aiApi.AIPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ai",
			Namespace: "default",
		},
		Spec: aiApi.AIPlatformSpec{
			CertificateRef: "new-cluster-issuer",
			ClusterDomain:  "cluster.local",
		},
	}

	// ✅ Existing cert with old issuer
	existingCert := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ai-tls",
			Namespace: "default",
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: "old-secret",
			IssuerRef:  cmmeta.ObjectReference{Name: "old-issuer", Kind: "ClusterIssuer"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(aiPlatform, existingCert).
		Build()

	reconciler := &AIPlatformReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// ✅ Run reconcileCertificate (should update existing cert)
	err := reconciler.reconcileCertificate(ctx, aiPlatform)
	assert.NoError(t, err)

	// ✅ Fetch updated cert
	updatedCert := &certmanagerv1.Certificate{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "test-ai-tls", Namespace: "default"}, updatedCert)
	assert.NoError(t, err)

	assert.Equal(t, "test-ai-tls-secret", updatedCert.Spec.SecretName, "Should update SecretName")
	assert.Equal(t, "new-cluster-issuer", updatedCert.Spec.IssuerRef.Name, "Should update IssuerRef")
	expectedDNS := "test-ai.default.svc.cluster.local"
	assert.Contains(t, updatedCert.Spec.DNSNames, expectedDNS)
}
