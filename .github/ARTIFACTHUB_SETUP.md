# Artifact Hub Setup Guide

This guide explains how to register your Helm charts on Artifact Hub.

## What is Artifact Hub?

[Artifact Hub](https://artifacthub.io/) is a centralized registry for Kubernetes packages (Helm charts, Operators, etc.). It's like "npm for Kubernetes" - users can discover and install your charts easily.

**Benefits:**
- 🔍 Increased discoverability
- 📊 Usage statistics
- 🔒 Security scanning results
- 📝 Automatic documentation
- 🏷️ Version tracking
- ⭐ Community ratings

---

## Prerequisites

Before you begin:
- ✅ Helm charts must be available in a GitHub Release
- ✅ Charts must be packaged (`.tgz` files)
- ✅ Charts must have `Chart.yaml` with proper metadata
- ✅ A GitHub account with access to the repository

---

## Step-by-Step Setup

### Step 1: Create Artifact Hub Account

1. Go to https://artifacthub.io/
2. Click **"Sign in"** (top right)
3. Choose **"Sign in with GitHub"**
4. Authorize Artifact Hub to access your GitHub account
5. Complete your profile

### Step 2: Add Your Repository

Once logged in:

1. Click your **profile icon** (top right) → **"Control Panel"**
2. In the left sidebar, click **"Repositories"**
3. Click **"Add repository"** button
4. Fill in the form:

#### Repository Details:

```yaml
Kind: Helm charts
Name: splunk-ai-operator
Display name: Splunk AI Operator
URL: https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/
```

**Important:** The URL should point to where your Helm chart releases are stored. Since you're using GitHub Releases:
```
https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/
```

Or if you create a consolidated index across all releases:
```
https://splunk.github.io/splunk-ai-operator/
```

#### Optional Fields:

```yaml
Description: Kubernetes operator for managing AI workloads using standardized CRDs
Official: ✅ (check this if you're from Splunk)
Scanner disabled: ☐ (leave unchecked for security scanning)
```

5. Click **"Add"**

### Step 3: Verify Repository

After adding:

1. Artifact Hub will scan your repository (takes 1-5 minutes)
2. Go to **"Repositories"** in Control Panel
3. Check the status - should show **"Success"** with a green checkmark
4. If there are errors, click on the repository to see details

### Step 4: View Your Charts

Once verified:

1. Go to https://artifacthub.io/
2. Search for "splunk-ai-operator"
3. You should see your charts listed!

Example URL:
```
https://artifacthub.io/packages/helm/splunk-ai-operator/splunk-ai-operator
```

---

## Repository URL Options

You have two main options for hosting your Helm charts:

### Option 1: GitHub Releases (Current Setup) ✅

**URL Format:**
```
https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/
```

**Pros:**
- ✅ Already set up (you're using this)
- ✅ No additional hosting needed
- ✅ Works with current workflow

**Cons:**
- ⚠️ URL changes with each version
- ⚠️ Need to update Artifact Hub for each release

**Best for:** Getting started quickly

### Option 2: GitHub Pages (Recommended for Multiple Versions)

**URL Format:**
```
https://splunk.github.io/splunk-ai-operator/
```

**Pros:**
- ✅ Single, stable URL
- ✅ Hosts multiple versions
- ✅ Automatic updates via workflow

**Cons:**
- ⚠️ Requires additional setup

**Best for:** Production use with multiple versions

---

## Setting Up GitHub Pages (Optional but Recommended)

If you want a stable URL that works across all versions:

### 1. Create gh-pages branch workflow

Add this to `.github/workflows/publish-helm-charts.yml`:

```yaml
name: Publish Helm Charts to GitHub Pages

on:
  push:
    tags:
      - 'v*.*.*'

permissions:
  contents: write

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Configure Git
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"

      - name: Install Helm
        uses: azure/setup-helm@v4
        with:
          version: v3.14.0

      - name: Run chart-releaser
        uses: helm/chart-releaser-action@v1.6.0
        with:
          charts_dir: helm-chart
        env:
          CR_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
```

### 2. Enable GitHub Pages

1. Go to repository **Settings**
2. Navigate to **Pages** (left sidebar)
3. Under **Source**, select:
   - Branch: `gh-pages`
   - Folder: `/ (root)`
4. Click **Save**
5. Wait 2-3 minutes for deployment

### 3. Update Artifact Hub

Go back to Artifact Hub Control Panel and update your repository URL to:
```
https://splunk.github.io/splunk-ai-operator/
```

---

## Updating Artifact Hub After Each Release

### If using GitHub Releases URL:

After each release, you need to update the URL in Artifact Hub:

1. Go to Artifact Hub Control Panel
2. Click on your repository
3. Click **"Edit"**
4. Update URL to new version: `https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.1/`
5. Click **"Update"**

### If using GitHub Pages:

No action needed! Artifact Hub automatically discovers new versions.

---

## Adding the Badge to README

Once your repository is verified on Artifact Hub:

### Get Your Badge

1. Go to your package page: https://artifacthub.io/packages/helm/splunk-ai-operator/splunk-ai-operator
2. Click the **"Badge"** button (top right)
3. Copy the Markdown code

### Add to README.md

```markdown
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/splunk-ai-operator)](https://artifacthub.io/packages/search?repo=splunk-ai-operator)
```

Place it in the "Release & Version" section of your README.

---

## Troubleshooting

### "Repository not found" error

**Cause:** URL is incorrect or charts are not accessible

**Solution:**
1. Verify URL in browser - it should download `index.yaml`
2. Check that GitHub Release is public
3. Ensure chart files are uploaded to release

### "Invalid index.yaml" error

**Cause:** Helm repository index is malformed

**Solution:**
1. Check that `helm repo index` was run during release
2. Verify `index.yaml` is included in GitHub Release artifacts
3. Validate index.yaml structure:
   ```bash
   curl https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/index.yaml
   ```

### Charts not appearing

**Cause:** Artifact Hub hasn't scanned yet

**Solution:**
1. Wait 5 minutes for automatic scan
2. Or manually trigger: Control Panel → Repositories → Click repository → "Check now"

### "Authentication required" error

**Cause:** Repository is private

**Solution:**
- GitHub Releases must be public for Artifact Hub to access them
- Check repository visibility settings

---

## Maintenance

### Keeping Charts Updated

**With GitHub Releases:**
- Update Artifact Hub URL after each release
- Or switch to GitHub Pages for automatic updates

**With GitHub Pages:**
- Charts auto-update when you push new releases
- Artifact Hub scans every few hours
- Manual refresh: Control Panel → "Check now"

### Monitoring

Check your Artifact Hub dashboard periodically:
- **Views**: How many people viewed your charts
- **Installs**: Estimate based on pulls (if available)
- **Security**: Any vulnerabilities found
- **Versions**: All published versions

---

## Example: Complete Setup

Here's what the final setup looks like:

### 1. Files in Repository:
```
helm-chart/
├── splunk-ai-operator/
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── templates/
│   └── artifacthub-repo.yml  ← Added
└── splunk-ai-platform/
    ├── Chart.yaml
    ├── values.yaml
    ├── templates/
    └── artifacthub-repo.yml  ← Added
```

### 2. After v1.0.0 Release:
- GitHub Release created with:
  - `splunk-ai-operator-1.0.0.tgz`
  - `splunk-ai-platform-1.0.0.tgz`
  - `index.yaml`

### 3. Artifact Hub Configuration:
```yaml
Repository: splunk-ai-operator
URL: https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/
Kind: Helm charts
Status: ✅ Success
```

### 4. Result:
- Charts discoverable at: https://artifacthub.io/
- Users can find via search
- Installation instructions auto-generated
- Badge working in README

---

## Quick Start (TL;DR)

1. **Create metadata files** (done! ✅)
   - `helm-chart/splunk-ai-operator/artifacthub-repo.yml`
   - `helm-chart/splunk-ai-platform/artifacthub-repo.yml`

2. **Merge PR and create v1.0.0 release**
   - Workflow will include artifacthub-repo.yml in charts

3. **Register on Artifact Hub:**
   - Sign in with GitHub
   - Add repository: https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/
   - Wait for verification

4. **Add badge to README**
   - Get from Artifact Hub package page
   - Add to "Release & Version" section

5. **Done!** 🎉

---

## Need Help?

- **Artifact Hub Docs**: https://artifacthub.io/docs
- **Helm Chart Repository Guide**: https://helm.sh/docs/topics/chart_repository/
- **GitHub Issues**: Report problems in this repository

---

**Next Steps:**
1. Commit the `artifacthub-repo.yml` files
2. Merge your PR
3. Create v1.0.0 release
4. Register on Artifact Hub
5. Add badge to README
