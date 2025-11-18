# OCI Helm Registry Guide

This project uses **OCI registries** to distribute Helm charts, providing a modern, clean approach without storing binary `.tgz` files in git.

## What is OCI for Helm?

Starting with Helm 3.8.0, Helm charts can be stored in OCI (Open Container Initiative) registries, the same infrastructure used for Docker images.

**Traditional Helm Repository:**
```
helm repo add splunk-ai https://example.com/charts/
helm install splunk-ai-operator splunk-ai/splunk-ai-operator
```

**OCI Registry (Modern):**
```bash
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 1.0.0
```

---

## Benefits

### No Binary Files in Git ✅
- Charts stored in GHCR (GitHub Container Registry)
- No `.tgz` files committed to any branch
- Git repository stays clean

### Unified Infrastructure ✅
- Same registry for Docker images + Helm charts
- Single authentication system
- Consistent access controls

### Better Security ✅
- Immutable like container images
- Support for signing (cosign)
- Vulnerability scanning
- Access control via GitHub permissions

### Modern Standard ✅
- Helm 3.8+ native support
- Industry trend (CNCF recommended)
- Used by major projects (Kubernetes, Argo CD, etc.)

---

## Chart Locations

After each release, charts are available at:

### Operator Chart:
```
oci://ghcr.io/splunk/charts/splunk-ai-operator
```

### Platform Chart:
```
oci://ghcr.io/splunk/charts/splunk-ai-platform
```

### Versioned:
```
oci://ghcr.io/splunk/charts/splunk-ai-operator:1.0.0
oci://ghcr.io/splunk/charts/splunk-ai-operator:1.0.1
oci://ghcr.io/splunk/charts/splunk-ai-platform:1.0.0
```

---

## Installation Guide

### Prerequisites

**Check Helm Version:**
```bash
helm version --short
# Need: v3.8.0 or later
```

**Upgrade if needed:**
```bash
# macOS
brew upgrade helm

# Linux
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Windows
choco upgrade kubernetes-helm
```

---

## Installing Charts

### Option 1: Specific Version (Recommended)

```bash
# Install operator chart
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 1.0.0 \
  --namespace splunk-ai-operator-system \
  --create-namespace

# Install platform chart
helm install splunk-ai-platform \
  oci://ghcr.io/splunk/charts/splunk-ai-platform \
  --version 1.0.0
```

### Option 2: Latest Version

```bash
# Helm will pull the latest version
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator
```

### Option 3: With Custom Values

```bash
# Create values file
cat > my-values.yaml <<EOF
image:
  tag: v1.0.0
replicaCount: 2
EOF

# Install with custom values
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 1.0.0 \
  --values my-values.yaml
```

---

## Common Operations

### List Available Versions

```bash
# Use GitHub Container Registry UI
open https://github.com/splunk/splunk-ai-operator/pkgs/container/charts%2Fsplunk-ai-operator

# Or use oras CLI
oras discover ghcr.io/splunk/charts/splunk-ai-operator
```

### Pull Chart Locally

```bash
# Pull chart package
helm pull oci://ghcr.io/splunk/charts/splunk-ai-operator --version 1.0.0

# This creates: splunk-ai-operator-1.0.0.tgz
```

### Show Chart Information

```bash
# Show chart metadata
helm show chart oci://ghcr.io/splunk/charts/splunk-ai-operator --version 1.0.0

# Show values
helm show values oci://ghcr.io/splunk/charts/splunk-ai-operator --version 1.0.0

# Show all (chart + values + readme)
helm show all oci://ghcr.io/splunk/charts/splunk-ai-operator --version 1.0.0
```

### Template Chart Locally

```bash
# Render templates without installing
helm template my-release \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 1.0.0
```

### Upgrade Existing Installation

```bash
# Upgrade to newer version
helm upgrade splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 1.0.1
```

### Uninstall

```bash
helm uninstall splunk-ai-operator --namespace splunk-ai-operator-system
```

---

## Authentication

### Public Charts (No Auth Needed)

If charts are public, no authentication required:
```bash
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator
```

### Private Charts (Auth Required)

If charts are private:

```bash
# Login to GHCR
echo $GITHUB_TOKEN | helm registry login ghcr.io -u USERNAME --password-stdin

# Then install
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator

# Logout
helm registry logout ghcr.io
```

**Create GitHub Token:**
1. Go to: https://github.com/settings/tokens
2. Generate new token (classic)
3. Select scopes: `read:packages`
4. Use as `$GITHUB_TOKEN`

---

## Comparison

### Traditional Helm Repository vs OCI

| Feature | Traditional Repo | OCI Registry |
|---------|-----------------|--------------|
| **Storage** | HTTP server + index.yaml | Container registry |
| **Files in Git** | Often .tgz committed | ❌ None |
| **helm repo add** | ✅ Yes | ❌ No (use direct URL) |
| **Versioning** | index.yaml | OCI tags |
| **Security** | Basic auth | OCI auth + signing |
| **Infrastructure** | Separate hosting | Reuse container registry |
| **Standard** | Original Helm | Modern (Helm 3.8+) |

---

## For CI/CD

### GitHub Actions Example

```yaml
- name: Install Helm
  uses: azure/setup-helm@v4
  with:
    version: v3.14.0

- name: Install from OCI
  run: |
    helm install splunk-ai-operator \
      oci://ghcr.io/splunk/charts/splunk-ai-operator \
      --version 1.0.0 \
      --wait
```

### Kubernetes Job Example

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: helm-install
spec:
  template:
    spec:
      containers:
      - name: helm
        image: alpine/helm:3.14.0
        command:
          - helm
          - install
          - splunk-ai-operator
          - oci://ghcr.io/splunk/charts/splunk-ai-operator
          - --version
          - "1.0.0"
      restartPolicy: Never
```

---

## Troubleshooting

### "Error: failed to authorize"

**Cause:** Authentication required or missing permissions

**Solution:**
```bash
# For private charts, login first
helm registry login ghcr.io -u USERNAME

# For public charts, ensure chart is published
open https://github.com/orgs/splunk/packages
```

### "Error: chart not found"

**Cause:** Chart not yet published or wrong URL

**Solution:**
1. Check chart exists: https://github.com/orgs/splunk/packages
2. Verify URL format: `oci://ghcr.io/splunk/charts/splunk-ai-operator`
3. Check version exists

### "Error: Helm version too old"

**Cause:** Helm < 3.8.0

**Solution:**
```bash
# Upgrade Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm version --short
```

### "Error: failed to pull"

**Cause:** Network issue or chart not public

**Solution:**
1. Check internet connection
2. Verify chart visibility in GHCR
3. Try with authentication

---

## Migration from Traditional Repo

If users were using traditional Helm repo:

### Before (Traditional):
```bash
helm repo add splunk-ai https://example.com/charts
helm repo update
helm install splunk-ai-operator splunk-ai/splunk-ai-operator
```

### After (OCI):
```bash
# No helm repo add needed!
helm install splunk-ai-operator \
  oci://ghcr.io/splunk/charts/splunk-ai-operator \
  --version 1.0.0
```

**Notes:**
- No `helm repo add` command needed
- Specify version explicitly (best practice)
- Can still use GitHub Releases as fallback

---

## Also Available: GitHub Releases

For users who prefer traditional approach or can't use Helm 3.8+:

```bash
# Download from GitHub Release
helm install splunk-ai-operator \
  https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/splunk-ai-operator-1.0.0.tgz
```

Both methods supported! Choose what works best for you.

---

## FAQ

### Q: Do I need `helm repo add`?
**A:** No! With OCI, you use the direct `oci://` URL.

### Q: Can I still use traditional Helm repo?
**A:** Charts are also available in GitHub Releases as `.tgz` files for compatibility.

### Q: Is OCI more secure?
**A:** Yes! OCI registries support signing, scanning, and better access controls.

### Q: What Helm version do I need?
**A:** Helm 3.8.0 or later (released April 2022).

### Q: Where are charts stored?
**A:** In GHCR (GitHub Container Registry) at `ghcr.io/splunk/charts/`.

### Q: Can I see all versions?
**A:** Yes, visit: https://github.com/splunk/splunk-ai-operator/pkgs/container/charts%2Fsplunk-ai-operator

### Q: Is this the recommended way?
**A:** Yes! OCI is the modern standard for Helm 3.8+.

---

## Resources

- **Helm OCI Documentation**: https://helm.sh/docs/topics/registries/
- **CNCF OCI Artifacts**: https://github.com/opencontainers/artifacts
- **GitHub Container Registry**: https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry
- **Helm Best Practices**: https://helm.sh/docs/chart_best_practices/

---

**Questions?** Open an issue or refer to the main documentation.
