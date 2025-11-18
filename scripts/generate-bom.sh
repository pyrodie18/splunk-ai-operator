#!/usr/bin/env bash

# Generate Bill of Materials (BOM) for Splunk AI Operator
# This script extracts all Docker images used by the operator and its dependencies
# Usage: ./scripts/generate-bom.sh [VERSION]

set -euo pipefail

VERSION="${1:-unknown}"
OUTPUT_DIR="${2:-.}"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Output files
BOM_JSON="${OUTPUT_DIR}/bom-v${VERSION}.json"
BOM_YAML="${OUTPUT_DIR}/bom-v${VERSION}.yaml"
BOM_TXT="${OUTPUT_DIR}/bom-v${VERSION}.txt"

echo "Generating Bill of Materials for Splunk AI Operator v${VERSION}"
echo "Output directory: ${OUTPUT_DIR}"

# Source .env file to get image versions
if [ -f .env ]; then
    # shellcheck disable=SC1091
    set -a
    source .env
    set +a
else
    echo "Warning: .env file not found, using defaults"
fi

# Operator image (from kustomization.yaml or parameter)
OPERATOR_IMAGE="ghcr.io/splunk/splunk-ai-operator:v${VERSION}"

# Extract images from environment variables
declare -A IMAGES=(
    ["operator"]="${OPERATOR_IMAGE}"
    ["splunk-enterprise"]="${RELATED_IMAGE_SPLUNK_ENTERPRISE:-splunk/splunk:9.2.3}"
    ["ray-head"]="${RELATED_IMAGE_RAY_HEAD:-unknown}"
    ["ray-worker"]="${RELATED_IMAGE_RAY_WORKER:-unknown}"
    ["weaviate"]="${RELATED_IMAGE_WEAVIATE:-semitechnologies/weaviate:stable-v1.28-007846a}"
    ["saia-api"]="${RELATED_IMAGE_SAIA_API:-unknown}"
    ["post-install-hook"]="${RELATED_IMAGE_POST_INSTALL_HOOK:-unknown}"
    ["fluent-bit"]="${RELATED_IMAGE_FLUENT_BIT:-fluent/fluent-bit:1.9.6}"
)

# Additional metadata
MODEL_VERSION="${MODEL_VERSION:-unknown}"
RAY_VERSION="${RAY_VERSION:-unknown}"

# Generate JSON BOM
cat > "${BOM_JSON}" <<EOF
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.4",
  "version": 1,
  "metadata": {
    "timestamp": "${TIMESTAMP}",
    "component": {
      "type": "application",
      "name": "splunk-ai-operator",
      "version": "${VERSION}",
      "description": "Splunk AI Operator - Kubernetes Operator for AI Platform"
    },
    "properties": [
      {
        "name": "model_version",
        "value": "${MODEL_VERSION}"
      },
      {
        "name": "ray_version",
        "value": "${RAY_VERSION}"
      }
    ]
  },
  "components": [
EOF

# Add components to JSON
FIRST=true
for name in "${!IMAGES[@]}"; do
    image="${IMAGES[$name]}"
    if [ "$FIRST" = true ]; then
        FIRST=false
    else
        echo "," >> "${BOM_JSON}"
    fi

    # Extract image parts
    if [[ "$image" == *":"* ]]; then
        image_name="${image%:*}"
        image_tag="${image##*:}"
    else
        image_name="$image"
        image_tag="latest"
    fi

    cat >> "${BOM_JSON}" <<COMPONENT
    {
      "type": "container",
      "name": "${name}",
      "version": "${image_tag}",
      "purl": "pkg:docker/${image_name}@${image_tag}",
      "properties": [
        {
          "name": "image",
          "value": "${image}"
        }
      ]
    }
COMPONENT
done

cat >> "${BOM_JSON}" <<EOF

  ]
}
EOF

# Generate YAML BOM
cat > "${BOM_YAML}" <<EOF
apiVersion: v1
kind: BillOfMaterials
metadata:
  name: splunk-ai-operator
  version: ${VERSION}
  timestamp: ${TIMESTAMP}
spec:
  operatorImage: ${OPERATOR_IMAGE}
  dependencies:
    modelVersion: ${MODEL_VERSION}
    rayVersion: ${RAY_VERSION}
  containerImages:
EOF

for name in "${!IMAGES[@]}"; do
    image="${IMAGES[$name]}"
    cat >> "${BOM_YAML}" <<YAML_IMAGE
    - name: ${name}
      image: ${image}
YAML_IMAGE
done

# Generate human-readable text BOM
cat > "${BOM_TXT}" <<EOF
================================================================================
Bill of Materials (BOM)
Splunk AI Operator v${VERSION}
Generated: ${TIMESTAMP}
================================================================================

OPERATOR IMAGE
--------------
${OPERATOR_IMAGE}

MANAGED CONTAINER IMAGES
------------------------
EOF

for name in "${!IMAGES[@]}"; do
    if [ "$name" != "operator" ]; then
        printf "%-25s %s\n" "${name}:" "${IMAGES[$name]}" >> "${BOM_TXT}"
    fi
done

cat >> "${BOM_TXT}" <<EOF

DEPENDENCY VERSIONS
-------------------
Model Version:      ${MODEL_VERSION}
Ray Version:        ${RAY_VERSION}

VERIFICATION
------------
To verify image digests:
  docker pull <image> --platform linux/amd64
  docker inspect <image> --format='{{.RepoDigests}}'

SECURITY
--------
For security scanning, use:
  grype <image>
  trivy image <image>

================================================================================
EOF

echo "✅ Generated BOM files:"
echo "   - ${BOM_JSON} (CycloneDX format)"
echo "   - ${BOM_YAML} (Kubernetes-friendly YAML)"
echo "   - ${BOM_TXT} (Human-readable text)"

# Print summary
echo ""
echo "Summary of images included in v${VERSION}:"
echo "----------------------------------------"
for name in "${!IMAGES[@]}"; do
    printf "%-25s %s\n" "${name}:" "${IMAGES[$name]}"
done
echo "----------------------------------------"
echo "Total images: ${#IMAGES[@]}"
