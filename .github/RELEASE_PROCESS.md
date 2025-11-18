# Release Process

This document describes how to create releases for the Splunk AI Operator.

## Overview

The release process is **fully automated** via GitHub Actions. You do not need to create tags locally.

## Creating a Release

### Step 1: Trigger the Release Workflow

1. Go to: https://github.com/splunk/splunk-ai-operator/actions/workflows/create-release-tag.yml
2. Click **"Run workflow"** button (top right)
3. Fill in the form:
   - **Use workflow from**: Select `main` (default)
   - **Release version**: Enter version number (e.g., `1.0.0` or `v1.0.0`)
   - **Mark as pre-release**: Check if this is a pre-release (beta, rc, etc.)
4. Click **"Run workflow"** button (green)

### Step 2: Monitor Progress

The workflow will automatically:
1. ✅ Validate the version format
2. ✅ Check if tag already exists
3. ✅ Create an annotated git tag
4. ✅ Push tag to GitHub
5. ✅ Trigger the release workflow automatically

**Monitor:**
- [View Create Tag Workflow](https://github.com/splunk/splunk-ai-operator/actions/workflows/create-release-tag.yml)
- [View Release Workflow](https://github.com/splunk/splunk-ai-operator/actions/workflows/release-package-helm.yml)

### Step 3: Verify Release

After both workflows complete (5-10 minutes):

1. **Check Release:**
   - Go to: https://github.com/splunk/splunk-ai-operator/releases
   - Verify your release appears with all artifacts

2. **Check Docker Images:**
   - GHCR: https://github.com/splunk/splunk-ai-operator/pkgs/container/splunk-ai-operator
   - Docker Hub: https://hub.docker.com/r/splunk/splunk-ai-operator/tags

3. **Test Installation:**
   ```bash
   # Test kubectl installation
   kubectl apply -f https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/install-v1.0.0.yaml

   # Test Helm installation
   helm install splunk-ai-operator \
     https://github.com/splunk/splunk-ai-operator/releases/download/v1.0.0/splunk-ai-operator-1.0.0.tgz

   # Test Docker pull
   docker pull ghcr.io/splunk/splunk-ai-operator:v1.0.0
   docker pull splunk/splunk-ai-operator:v1.0.0
   ```

---

## Version Format

### Semantic Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
```

**Format Rules:**
- `X.Y.Z` - Standard release (e.g., `1.0.0`)
- `X.Y.Z-prerelease` - Pre-release (e.g., `1.0.0-beta.1`, `1.0.0-rc.2`)
- `X.Y.Z+build` - Build metadata (e.g., `1.0.0+20250118`)

**Examples:**

✅ **Valid:**
- `1.0.0`
- `v1.0.0` (automatically normalized)
- `1.0.0-beta.1`
- `1.0.0-rc.1`
- `1.0.0-alpha`
- `2.1.3+build.123`
- `1.0.0-beta.1+build.456`

❌ **Invalid:**
- `1.0` (missing patch version)
- `v1` (missing minor and patch)
- `1.0.0.0` (too many components)
- `latest` (not a version)
- `main` (not a version)

### Version Increment Guidelines

**MAJOR** (`X.0.0`):
- Breaking API changes
- Incompatible CRD changes
- Major feature overhaul

**MINOR** (`0.Y.0`):
- New features (backward compatible)
- New CRD fields (with defaults)
- Deprecations (with warnings)

**PATCH** (`0.0.Z`):
- Bug fixes
- Documentation updates
- Security patches

**PRERELEASE** (`-suffix`):
- `-alpha.N` - Early testing, may have bugs
- `-beta.N` - Feature complete, testing needed
- `-rc.N` - Release candidate, production-like

---

## What Gets Released

When you create a tag `v1.0.0`, the automated workflow creates:

### Docker Images:
```
ghcr.io/splunk/splunk-ai-operator:v1.0.0
ghcr.io/splunk/splunk-ai-operator:1.0.0  (without v prefix)
ghcr.io/splunk/splunk-ai-operator:latest (if on main)

splunk/splunk-ai-operator:v1.0.0
splunk/splunk-ai-operator:1.0.0
splunk/splunk-ai-operator:latest (if on main)
```

### Release Artifacts:
```
install-v1.0.0.yaml                 - Kubernetes manifests (CRDs + Operator)
splunk-ai-operator-1.0.0.tgz        - Operator Helm chart
splunk-ai-platform-1.0.0.tgz        - Platform Helm chart
index.yaml                           - Helm repository index
```

### GitHub Release:
- Auto-generated release notes from commits
- Links to all artifacts
- Installation instructions
- Container image references

---

## Troubleshooting

### "Tag already exists"

**Error:**
```
❌ Tag v1.0.0 already exists on remote
```

**Solution:**
1. Choose a different version number, or
2. Delete the existing tag:
   ```bash
   # Delete from GitHub (requires admin access)
   git push --delete origin v1.0.0

   # Delete locally
   git tag -d v1.0.0
   ```
3. Delete the GitHub Release if it exists:
   - Go to: https://github.com/splunk/splunk-ai-operator/releases
   - Find the release → Delete

### "Invalid version format"

**Error:**
```
❌ Invalid version format: 1.0
```

**Solution:**
Use semantic versioning format: `X.Y.Z`
- ✅ `1.0.0` (correct)
- ❌ `1.0` (missing patch)

### "Workflow failed to create release"

**Check:**
1. View workflow logs:
   - [Create Tag Workflow](https://github.com/splunk/splunk-ai-operator/actions/workflows/create-release-tag.yml)
   - [Release Workflow](https://github.com/splunk/splunk-ai-operator/actions/workflows/release-package-helm.yml)

2. Common issues:
   - **Docker Hub authentication**: Check `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` secrets
   - **Permissions**: Ensure workflow has `contents: write` permission
   - **Build failures**: Check if tests pass on main branch

### "Docker images not appearing"

**Check:**
1. GHCR: https://github.com/orgs/splunk/packages?repo_name=splunk-ai-operator
2. Docker Hub: https://hub.docker.com/r/splunk/splunk-ai-operator/tags

**Note:** Images may take 5-10 minutes to appear after workflow completes.

---

## Release Checklist

Before creating a release:

### Pre-Release (1 week before):
- [ ] All PRs for release merged to `main`
- [ ] All CI/CD checks passing on `main`
- [ ] Documentation updated
- [ ] CHANGELOG.md updated with release notes
- [ ] Breaking changes documented (if MAJOR version)
- [ ] Migration guide written (if needed)

### Release Day:
- [ ] Pull latest `main`: `git checkout main && git pull`
- [ ] Run release workflow via GitHub UI
- [ ] Monitor workflow completion (5-10 min)
- [ ] Verify release on GitHub
- [ ] Test installation (kubectl, Helm, Docker)
- [ ] Verify Docker images on GHCR and Docker Hub
- [ ] Update release notes (edit GitHub Release if needed)

### Post-Release:
- [ ] Announce release (Slack, email, etc.)
- [ ] Update documentation website (if applicable)
- [ ] Monitor GitHub issues for problems
- [ ] Create hotfix tag if critical bugs found

---

## Emergency Rollback

If a release has critical issues:

### Option 1: Quick Hotfix
```bash
# Let CI create the patch tag
# Use GitHub UI workflow: create-release-tag.yml
# Version: 1.0.1 (increment patch)
```

### Option 2: Mark Release as Pre-release
1. Go to release page
2. Edit release
3. Check "Set as a pre-release"
4. Update description with warning

### Option 3: Delete Release (Last Resort)
1. Delete GitHub Release
2. Delete tag: `git push --delete origin v1.0.0`
3. Delete Docker images (if possible)
4. Communicate to users

---

## Examples

### Creating v1.0.0 (First GA Release):

1. Go to: https://github.com/splunk/splunk-ai-operator/actions/workflows/create-release-tag.yml
2. Click "Run workflow"
3. Enter:
   - Version: `1.0.0`
   - Pre-release: ❌ (unchecked)
4. Wait ~10 minutes
5. Check: https://github.com/splunk/splunk-ai-operator/releases/tag/v1.0.0

### Creating v1.1.0-beta.1 (Pre-release):

1. Go to workflow
2. Click "Run workflow"
3. Enter:
   - Version: `1.1.0-beta.1`
   - Pre-release: ✅ (checked)
4. Release will be marked as "Pre-release" on GitHub

### Creating v1.0.1 (Hotfix):

1. Merge hotfix PR to `main`
2. Go to workflow
3. Enter:
   - Version: `1.0.1`
   - Pre-release: ❌
4. Patch release created automatically

---

## FAQ

### Q: Can I create tags locally?
**A:** You can, but it's not recommended. Use the GitHub Actions workflow for consistency.

### Q: Can I delete a tag after releasing?
**A:** Yes, but avoid this in production. Users may already be using that version.

### Q: What if the workflow fails?
**A:** Check the logs, fix the issue, then retry with the same version (after deleting the tag if it was created).

### Q: Can I release from a branch other than main?
**A:** The workflow is configured for `main` only. For special releases, modify the workflow temporarily.

### Q: How do I release a hotfix?
**A:** Same process, just increment the patch version (e.g., `1.0.0` → `1.0.1`).

### Q: Can I edit release notes after creation?
**A:** Yes! Go to the release page on GitHub and click "Edit release".

---

## Additional Resources

- [Semantic Versioning Specification](https://semver.org/)
- [GitHub Releases Documentation](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [Helm Chart Versioning Best Practices](https://helm.sh/docs/topics/charts/#charts-and-versioning)
- [Container Image Tagging Best Practices](https://docs.docker.com/develop/dev-best-practices/)

---

**Questions?** Open an issue or ask in team Slack channel.
