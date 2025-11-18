# Helm Repository Setup with GitHub Pages

This guide explains how to set up a consolidated Helm repository using GitHub Pages that hosts all chart versions in one place.

## Why Use GitHub Pages?

**Current Setup (GitHub Releases):**
```
https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/
https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.1/
https://github.com/splunk/splunk-ai-operator/releases/download/v1.1.0/
```
- ❌ Different URL for each release
- ❌ Users must know exact version
- ❌ No central index

**With GitHub Pages:**
```
https://splunk.github.io/splunk-ai-operator/
```
- ✅ Single, stable URL
- ✅ Consolidated index.yaml with all versions
- ✅ Standard Helm repository experience
- ✅ Automatic updates via workflow
- ✅ Works with `helm repo add`

---

## Repository Structure

After setup, your repository will have two branches:

### `main` branch (Source):
```
helm-chart/
├── splunk-ai-operator/
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
└── splunk-ai-platform/
    ├── Chart.yaml
    ├── values.yaml
    └── templates/
```

### `gh-pages` branch (Published):
```
/  (root)
├── index.yaml                          # Consolidated index with all versions
├── splunk-ai-operator-1.0.0.tgz       # Chart packages
├── splunk-ai-operator-1.0.1.tgz
├── splunk-ai-operator-1.1.0.tgz
├── splunk-ai-platform-1.0.0.tgz
├── splunk-ai-platform-1.0.1.tgz
└── splunk-ai-platform-1.1.0.tgz
```

**Served at:** `https://splunk.github.io/splunk-ai-operator/`

---

## Setup Steps

### Step 1: Enable GitHub Pages

1. Go to repository **Settings**
2. Click **Pages** (left sidebar)
3. Under **Source**:
   - Select branch: `gh-pages`
   - Select folder: `/ (root)`
4. Click **Save**
5. ✅ GitHub Pages will be enabled

**Note:** The `gh-pages` branch will be created automatically by the workflow on first run.

### Step 2: Merge the Workflow

The workflow `publish-helm-to-pages.yml` has been created. It will:
- ✅ Automatically trigger on tag push (`v*.*.*`)
- ✅ Package your Helm charts
- ✅ Create/update `gh-pages` branch
- ✅ Generate consolidated `index.yaml`
- ✅ Publish to GitHub Pages

### Step 3: Initial Setup (First Run)

After merging your PR, you can manually trigger the workflow once to initialize:

1. Go to: https://github.com/splunk/splunk-ai-operator/actions/workflows/publish-helm-to-pages.yml
2. Click **"Run workflow"**
3. Select branch: `main`
4. Click **"Run workflow"**

This creates the `gh-pages` branch even before you have any releases.

### Step 4: Verify GitHub Pages

After the workflow runs:

1. Go to: https://splunk.github.io/splunk-ai-operator/
2. You should see the Helm repository index (or a 404 if no releases yet)
3. Check: https://splunk.github.io/splunk-ai-operator/index.yaml

**Note:** It may take 2-3 minutes after workflow completes for pages to update.

### Step 5: Update Artifact Hub

Once GitHub Pages is working:

1. Go to [Artifact Hub Control Panel](https://artifacthub.io/control-panel/repositories)
2. Edit your repository
3. Change URL to: `https://splunk.github.io/splunk-ai-operator/`
4. Click **"Update"**

Now Artifact Hub will discover all versions automatically!

---

## How It Works

### Workflow Trigger

The `publish-helm-to-pages.yml` workflow triggers when:
- You push a tag like `v1.0.0`
- You manually run the workflow

### What the Workflow Does

```
┌─────────────────────────────────────────────────┐
│  1. Checkout main branch                        │
│     Get the latest Helm charts                  │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│  2. Package charts (Helm Chart Releaser)        │
│     Creates .tgz files for each chart           │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│  3. Create/update gh-pages branch               │
│     - Adds packaged charts                      │
│     - Updates consolidated index.yaml           │
│     - Includes all previous versions            │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│  4. Deploy to GitHub Pages                      │
│     https://splunk.github.io/splunk-ai-operator │
└─────────────────────────────────────────────────┘
```

### Consolidated index.yaml Example

```yaml
apiVersion: v1
entries:
  splunk-ai-operator:
    - apiVersion: v2
      created: "2025-01-18T10:00:00Z"
      description: Kubernetes operator for AI workloads
      digest: abc123...
      name: splunk-ai-operator
      urls:
        - https://splunk.github.io/splunk-ai-operator/splunk-ai-operator-1.1.0.tgz
      version: 1.1.0
    - apiVersion: v2
      created: "2025-01-17T10:00:00Z"
      description: Kubernetes operator for AI workloads
      digest: def456...
      name: splunk-ai-operator
      urls:
        - https://splunk.github.io/splunk-ai-operator/splunk-ai-operator-1.0.1.tgz
      version: 1.0.1
    - apiVersion: v2
      created: "2025-01-16T10:00:00Z"
      description: Kubernetes operator for AI workloads
      digest: ghi789...
      name: splunk-ai-operator
      urls:
        - https://splunk.github.io/splunk-ai-operator/splunk-ai-operator-1.0.0.tgz
      version: 1.0.0
```

All versions in one index! ✅

---

## Using the Helm Repository

Once GitHub Pages is set up, users can install charts like this:

### Option 1: Add Repository (Recommended)

```bash
# Add the Helm repository
helm repo add splunk-ai https://splunk.github.io/splunk-ai-operator/

# Update repository
helm repo update

# Search for charts
helm search repo splunk-ai

# Install latest version
helm install splunk-ai-operator splunk-ai/splunk-ai-operator

# Install specific version
helm install splunk-ai-operator splunk-ai/splunk-ai-operator --version 1.0.0
```

### Option 2: Direct Install

```bash
# Install specific version directly
helm install splunk-ai-operator \
  https://splunk.github.io/splunk-ai-operator/splunk-ai-operator-1.0.0.tgz
```

---

## Benefits Over GitHub Releases Only

| Feature | GitHub Releases Only | With GitHub Pages |
|---------|---------------------|-------------------|
| **URL Stability** | ❌ Changes per version | ✅ Single URL |
| **Version Discovery** | ❌ Manual lookup | ✅ `helm search repo` |
| **helm repo add** | ❌ Doesn't work well | ✅ Full support |
| **Artifact Hub** | ⚠️ Needs URL updates | ✅ Auto-discovers |
| **User Experience** | ⚠️ Complex | ✅ Standard Helm |

---

## Dual Strategy (Recommended)

You can use **both** GitHub Releases and GitHub Pages:

### GitHub Releases:
- Individual version artifacts
- Kubernetes manifests (install.yaml)
- Detailed release notes
- GitHub Release UI

### GitHub Pages:
- Helm repository functionality
- Consolidated index
- `helm repo add` support
- Artifact Hub integration

**Workflow:** Both are updated automatically when you push a tag!

---

## Troubleshooting

### "gh-pages branch not found"

**Cause:** Workflow hasn't run yet

**Solution:**
1. Manually trigger: Actions → Publish Helm Charts → Run workflow
2. Or wait for first tag push

### "404 Not Found" on GitHub Pages

**Cause:** Pages not enabled or not deployed yet

**Solution:**
1. Check Settings → Pages → Ensure source is `gh-pages`
2. Wait 2-3 minutes after workflow completes
3. Check workflow logs for errors

### "index.yaml is empty"

**Cause:** No charts packaged yet

**Solution:**
1. Ensure charts are in `helm-chart/` directory
2. Ensure Chart.yaml is valid
3. Create a release tag to trigger packaging

### Charts not appearing in Artifact Hub

**Cause:** Artifact Hub hasn't scanned or URL is wrong

**Solution:**
1. Update Artifact Hub URL to: `https://splunk.github.io/splunk-ai-operator/`
2. Manually trigger scan in Artifact Hub Control Panel
3. Wait 5 minutes for automatic scan

---

## Maintenance

### Adding New Versions

Just create a new tag - everything is automatic:

```bash
# From GitHub UI (recommended)
Actions → Create Release Tag → Enter version

# The workflows automatically:
# 1. Build Docker images (release-package-helm.yml)
# 2. Create GitHub Release (release-package-helm.yml)
# 3. Update Helm repository (publish-helm-to-pages.yml)
```

### Updating Existing Charts

To update chart metadata without a new version:

1. Edit chart in `main` branch
2. Manually trigger `publish-helm-to-pages.yml`
3. Same version will be republished with updates

### Removing Old Versions

Generally not recommended, but if needed:

1. Go to `gh-pages` branch
2. Remove the specific `.tgz` file
3. Run: `helm repo index . --url https://splunk.github.io/splunk-ai-operator/`
4. Commit and push

---

## URLs Reference

After setup, your charts will be available at:

### Base Repository:
```
https://splunk.github.io/splunk-ai-operator/
```

### Index File:
```
https://splunk.github.io/splunk-ai-operator/index.yaml
```

### Chart Packages:
```
https://splunk.github.io/splunk-ai-operator/splunk-ai-operator-1.0.0.tgz
https://splunk.github.io/splunk-ai-operator/splunk-ai-platform-1.0.0.tgz
```

### For Artifact Hub:
```
Repository URL: https://splunk.github.io/splunk-ai-operator/
```

---

## Migration Path

If you're already using GitHub Releases:

1. ✅ Keep using GitHub Releases (nothing changes)
2. ✅ Add GitHub Pages (parallel setup)
3. ✅ Update Artifact Hub to point to Pages
4. ✅ Document both installation methods
5. ✅ Users can choose their preferred method

**No breaking changes!** Both methods coexist.

---

## Complete Setup Checklist

- [ ] Workflow `publish-helm-to-pages.yml` merged
- [ ] GitHub Pages enabled (Settings → Pages)
- [ ] Source set to `gh-pages` branch
- [ ] Workflow manually triggered once (or wait for tag)
- [ ] Verify: https://splunk.github.io/splunk-ai-operator/index.yaml
- [ ] Update Artifact Hub URL
- [ ] Test: `helm repo add splunk-ai https://splunk.github.io/splunk-ai-operator/`
- [ ] Update README with Helm repository instructions
- [ ] Done! 🎉

---

## Need Help?

- **Helm Chart Releaser**: https://github.com/helm/chart-releaser-action
- **GitHub Pages**: https://docs.github.com/en/pages
- **Helm Repository Guide**: https://helm.sh/docs/topics/chart_repository/

---

**Next Steps:**
1. Merge PR with `publish-helm-to-pages.yml`
2. Enable GitHub Pages in Settings
3. Manually trigger workflow once to initialize
4. Create v1.0.0 release
5. Verify Helm repository works
6. Update Artifact Hub URL
