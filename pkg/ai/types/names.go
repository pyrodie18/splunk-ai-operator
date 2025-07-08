package types

const (
	CertPath        = "/etc/certs/tls.crt"                    // Path to the TLS certificate file
	KeyPath         = "/etc/certs/tls.key"                    // Path to the TLS key file
	ApplicationPath = "ray-services/ai-platform/applications" // Path to the applications directory // FIXME TODO: remove this once we have a better way to handle multiple paths
	ArtifactsPath   = "model_artifacts"                       // Path to the artifacts directory // FIXME TODO: remove this once we have a better way to handle multiple paths
)
