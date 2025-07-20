package v1

const (
	// TotalWorker concurrent workers to reconcile
	TotalWorker int = 1
)

type SecretSourceType string

const (
    SecretSourceKubernetes SecretSourceType = "kubernetes"
    SecretSourceVault      SecretSourceType = "vault"
)