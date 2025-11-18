# Bill of Materials (BOM) Implementation Summary

## Overview

Implemented comprehensive Bill of Materials (BOM) and Software Bill of Materials (SBOM) generation for the Splunk AI Operator to support supply chain security, compliance, and transparency requirements.

## What Was Implemented

### 1. BOM Generation Script (`scripts/generate-bom.sh`)

A shell script that extracts all container images managed by the operator and generates BOM in multiple formats:

**Features:**
- Reads image references from `.env` file
- Extracts operator version and managed images
- Generates three output formats:
  - **Text** (`.txt`) - Human-readable format
  - **YAML** (`.yaml`) - Kubernetes-friendly format
  - **JSON** (`.json`) - CycloneDX standard format
- Includes metadata: model versions, Ray version, timestamps
- Provides verification and security scanning instructions

**Usage:**
```bash
./scripts/generate-bom.sh <VERSION> <OUTPUT_DIR>
```

**Example Output:**
```bash
./scripts/generate-bom.sh 0.1.0 dist/
# Creates:
# - dist/bom-v0.1.0.txt
# - dist/bom-v0.1.0.yaml
# - dist/bom-v0.1.0.json
```

### 2. Makefile Integration

Added `generate-bom` target to Makefile:

```makefile
.PHONY: generate-bom
generate-bom: ## Generate Bill of Materials (BOM) for release
	@echo "Generating Bill of Materials..."
	@mkdir -p dist
	@./scripts/generate-bom.sh $(VERSION) dist
	@echo "✅ BOM generated in dist/ directory"
```

**Usage:**
```bash
make generate-bom VERSION=0.1.0
```

### 3. Release Workflow Updates

Updated `.github/workflows/release-package-helm.yml` to automatically generate and publish BOM/SBOM artifacts:

**New Workflow Steps:**

#### a. Generate Custom BOM
```yaml
- name: Generate Bill of Materials (BOM)
  run: |
    VERSION="${{ steps.version.outputs.version }}"
    make generate-bom VERSION=$VERSION
    cat dist/bom-v$VERSION.txt
```

#### b. Install Syft for SBOM
```yaml
- name: Install Syft for SBOM generation
  uses: anchore/sbom-action/download-syft@ab5d7b5f48981941c4c5d6bf33aeb98fe3bae38c # v0.15.10
```

#### c. Generate SBOM
```yaml
- name: Generate SBOM for operator image
  run: |
    VERSION="${{ steps.version.outputs.version }}"
    OPERATOR_IMAGE="ghcr.io/splunk/splunk-ai-operator:v$VERSION"

    # Generate SBOM in multiple formats
    syft "$OPERATOR_IMAGE" -o cyclonedx-json=dist/sbom-operator-v$VERSION.cyclonedx.json
    syft "$OPERATOR_IMAGE" -o spdx-json=dist/sbom-operator-v$VERSION.spdx.json
    syft "$OPERATOR_IMAGE" -o syft-json=dist/sbom-operator-v$VERSION.syft.json
```

#### d. Include in Release Assets
All BOM/SBOM files are now attached to GitHub releases:
- `bom-v0.1.0.txt`
- `bom-v0.1.0.yaml`
- `bom-v0.1.0.json`
- `sbom-operator-v0.1.0.cyclonedx.json`
- `sbom-operator-v0.1.0.spdx.json`
- `sbom-operator-v0.1.0.syft.json`

### 4. Documentation

Created comprehensive documentation: `docs/bill-of-materials.md`

**Covers:**
- Overview of BOM/SBOM artifacts
- Available formats and use cases
- How to access and download BOM/SBOM files
- Verification and security scanning procedures
- Compliance use cases (SLSA, SSDF, EO 14028)
- Integration with CI/CD pipelines
- Best practices
- Policy enforcement examples

## Images Tracked in BOM

The BOM tracks all container images deployed by the operator:

1. **Operator Image**
   - `ghcr.io/splunk/splunk-ai-operator:vX.Y.Z`

2. **Managed Images** (from `.env`):
   - `RELATED_IMAGE_SPLUNK_ENTERPRISE` - Splunk Enterprise
   - `RELATED_IMAGE_RAY_HEAD` - Ray head node (ML runtime)
   - `RELATED_IMAGE_RAY_WORKER` - Ray worker nodes (GPU-enabled)
   - `RELATED_IMAGE_WEAVIATE` - Weaviate vector database
   - `RELATED_IMAGE_SAIA_API` - SAIA API service
   - `RELATED_IMAGE_POST_INSTALL_HOOK` - Post-installation hooks
   - `RELATED_IMAGE_FLUENT_BIT` - Fluent Bit logging

3. **Dependency Versions**:
   - `MODEL_VERSION` - ML model version
   - `RAY_VERSION` - Ray framework version

## BOM Format Examples

### Text Format (bom-v0.1.0.txt)
```text
================================================================================
Bill of Materials (BOM)
Splunk AI Operator v0.1.0
Generated: 2025-11-18T19:34:23Z
================================================================================

OPERATOR IMAGE
--------------
ghcr.io/splunk/splunk-ai-operator:v0.1.0

MANAGED CONTAINER IMAGES
------------------------
splunk-enterprise:        splunk/splunk:9.2.3
ray-head:                 example.ecr.aws.com/ray/ray-head:build-5
...
```

### YAML Format (bom-v0.1.0.yaml)
```yaml
apiVersion: v1
kind: BillOfMaterials
metadata:
  name: splunk-ai-operator
  version: 0.1.0
  timestamp: 2025-11-18T19:34:23Z
spec:
  operatorImage: ghcr.io/splunk/splunk-ai-operator:v0.1.0
  containerImages:
    - name: splunk-enterprise
      image: splunk/splunk:9.2.3
    ...
```

### JSON/CycloneDX Format (bom-v0.1.0.json)
```json
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.4",
  "version": 1,
  "metadata": {
    "timestamp": "2025-11-18T19:34:23Z",
    "component": {
      "type": "application",
      "name": "splunk-ai-operator",
      "version": "0.1.0"
    }
  },
  "components": [...]
}
```

## Compliance Benefits

### Supply Chain Security
- **SLSA Level 2+**: Build provenance attestations + SBOM
- **SSDF Compliance**: Software component inventory
- **EO 14028**: SBOM for critical software

### Vulnerability Management
- Continuous monitoring with SBOM-based scanners
- Integration with Grype, Trivy, Dependency-Track
- Automated CVE detection

### License Compliance
- License information in SBOM
- Organizational policy enforcement
- Audit trail for dependencies

## Integration Points

### CI/CD Integration
```yaml
# Download and scan SBOM in pipeline
- name: Scan SBOM
  run: |
    curl -LO https://github.com/splunk/splunk-ai-operator/releases/download/v0.1.0/sbom-operator-v0.1.0.cyclonedx.json
    grype sbom:./sbom-operator-v0.1.0.cyclonedx.json
```

### Vulnerability Scanning
```bash
# Scan operator image
trivy image ghcr.io/splunk/splunk-ai-operator:v0.1.0

# Scan using SBOM
grype sbom:./sbom-operator-v0.1.0.cyclonedx.json
```

### Policy Enforcement
```bash
# Verify all images from approved registries
./verify-approved-registries.sh bom-v0.1.0.yaml
```

## Files Created/Modified

### New Files
1. `scripts/generate-bom.sh` - BOM generation script
2. `docs/bill-of-materials.md` - Comprehensive documentation
3. `BOM_IMPLEMENTATION.md` - This summary document

### Modified Files
1. `Makefile` - Added `generate-bom` target
2. `.github/workflows/release-package-helm.yml` - Added BOM/SBOM generation steps

## Testing

Successfully tested BOM generation:
```bash
./scripts/generate-bom.sh 0.1.0 /tmp/bom-test

# Output:
✅ Generated BOM files:
   - /tmp/bom-test/bom-v0.1.0.json (CycloneDX format)
   - /tmp/bom-test/bom-v0.1.0.yaml (Kubernetes-friendly YAML)
   - /tmp/bom-test/bom-v0.1.0.txt (Human-readable text)

Summary of images included in v0.1.0:
----------------------------------------
operator:                 ghcr.io/splunk/splunk-ai-operator:v0.1.0
splunk-enterprise:        splunk/splunk:9.2.3
ray-head:                 example.ecr.aws.com/ray/ray-head:build-5
ray-worker:               example.ecr.aws.com/ray/ray-worker-gpu:build-6
weaviate:                 semitechnologies/weaviate:stable-v1.28-007846a
fluent-bit:               fluent/fluent-bit:1.9.6
...
----------------------------------------
Total images: 8
```

## Next Steps

### For Maintainers

1. **Keep `.env` file updated** with image versions:
   ```bash
   # Update image tags when dependencies change
   RELATED_IMAGE_RAY_HEAD=new-registry/ray-head:build-7
   ```

2. **Review BOM before release**:
   ```bash
   make generate-bom VERSION=0.2.0
   cat dist/bom-v0.2.0.txt
   ```

3. **Monitor vulnerability alerts** from SBOM scanners

### For Users

1. **Download BOM/SBOM** from GitHub releases
2. **Scan for vulnerabilities** before deployment
3. **Integrate with security tools** (Dependency-Track, etc.)
4. **Enforce organizational policies** using BOM

## Security Scanning Examples

### Scan All Managed Images
```bash
# Extract images from BOM and scan
grep "image:" bom-v0.1.0.yaml | awk '{print $3}' | while read image; do
  echo "Scanning: $image"
  trivy image "$image"
done
```

### Verify Image Digests
```bash
# Pull and inspect to get digest
docker pull ghcr.io/splunk/splunk-ai-operator:v0.1.0
docker inspect ghcr.io/splunk/splunk-ai-operator:v0.1.0 --format='{{.RepoDigests}}'
```

### Verify Build Attestations
```bash
# Using GitHub CLI
gh attestation verify oci://ghcr.io/splunk/splunk-ai-operator:v0.1.0 --owner splunk

# Using cosign
cosign verify-attestation \
  --type slsaprovenance \
  --certificate-identity-regexp="^https://github.com/splunk/splunk-ai-operator" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  ghcr.io/splunk/splunk-ai-operator:v0.1.0
```

## Resources

- **Documentation**: `docs/bill-of-materials.md`
- **Script**: `scripts/generate-bom.sh`
- **Makefile**: `make generate-bom VERSION=X.Y.Z`
- **Workflow**: `.github/workflows/release-package-helm.yml`

## Standards Compliance

- ✅ **CycloneDX 1.4** - Industry-standard BOM format
- ✅ **SPDX 2.3** - Software Package Data Exchange
- ✅ **SLSA** - Supply chain Levels for Software Artifacts
- ✅ **NTIA Minimum Elements** - SBOM baseline requirements
- ✅ **SSDF** - Secure Software Development Framework

## Summary

The BOM implementation provides comprehensive visibility into all software components and container images used by the Splunk AI Operator, supporting:

- **Security**: Vulnerability scanning, attestation verification
- **Compliance**: SLSA, SSDF, EO 14028 requirements
- **Transparency**: Clear inventory of dependencies
- **Automation**: Integrated into release workflow
- **Flexibility**: Multiple formats for different tools

Every release now includes complete BOM and SBOM artifacts, enabling organizations to meet security and compliance requirements while maintaining visibility into their software supply chain.
