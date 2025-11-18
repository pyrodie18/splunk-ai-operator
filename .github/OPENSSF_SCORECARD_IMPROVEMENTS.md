# OpenSSF Scorecard Improvements Guide

This document outlines improvements to achieve a high OpenSSF Scorecard score.

## What is OpenSSF Scorecard?

OpenSSF Scorecard is an automated security tool that checks for security best practices in open source projects. Scores range from 0-10 for each check.

**Scorecard Badge**: Once the workflow runs, you can add this badge to your README.md:
```markdown
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/splunk/splunk-ai-operator/badge)](https://securityscorecards.dev/viewer/?uri=github.com/splunk/splunk-ai-operator)
```

---

## Current Status

✅ **Already Implemented:**
- Dependabot configured
- CodeQL analysis enabled
- SECURITY.md present
- GitHub Actions workflows use commit SHAs (partially)
- Branch protection (needs configuration)

---

## High Priority Improvements

### 1. Pin GitHub Actions by SHA ⚠️ (Security Check)

**Current:** Using version tags like `@v4`, `@v5`
**Required:** Pin to full commit SHA

**Why:** Prevents supply chain attacks where action tags can be moved to malicious code.

**Example Fix:**
```yaml
# Before (current)
- uses: actions/checkout@v4

# After (recommended)
- uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11  # v4.1.1
```

**How to implement:**
1. Use a tool like Dependabot to manage action versions
2. Or manually look up SHAs: https://github.com/actions/checkout/releases

**Files to update:**
- `.github/workflows/main.yml`
- `.github/workflows/main-build-image.yml`
- `.github/workflows/main-check-formatting.yml`
- `.github/workflows/main-unit-tests.yml`
- `.github/workflows/main-vulnerability-scan.yml`
- `.github/workflows/helm-lint-test.yml`
- `.github/workflows/release-package-helm.yml`
- `.github/workflows/codeql-analysis.yml`
- `.github/workflows/scorecard.yml`

**Note:** This is a trade-off between security and maintainability. Dependabot can automatically update pinned SHAs.

---

### 2. Enable Branch Protection Rules ✅ (Required)

**Status:** Needs configuration in GitHub UI

**Required settings for `main` branch:**
```
Settings → Branches → Add branch protection rule → main

✅ Require pull request reviews before merging (1 approval)
✅ Require status checks to pass before merging:
   - check-formatting
   - unit-tests
   - build-image
   - helm-lint-test
✅ Require branches to be up to date before merging
✅ Include administrators
✅ Restrict who can push to matching branches (optional)
❌ Allow force pushes: NO
❌ Allow deletions: NO
```

**Impact:** Prevents direct pushes to main, ensures code review and CI passes.

---

### 3. Enable Security Policy ✅ (Already Done)

✅ SECURITY.md exists and is well-documented

---

### 4. Add CODEOWNERS File (Code Review Check)

Create `.github/CODEOWNERS`:
```
# Default owners for everything in the repo
*       @splunk/splunk-ai-operator-maintainers

# Require security team review for security-related changes
SECURITY.md                     @splunk/security-team
.github/workflows/*.yml         @splunk/devops-team

# Require docs team review for documentation
/docs/**                        @splunk/docs-team
*.md                            @splunk/docs-team
```

**Note:** Replace team names with actual GitHub teams or individual usernames.

---

### 5. Enable Required Signed Commits (Optional, Advanced)

**Status:** Not currently enforced

**How to enable:**
```
Settings → Branches → main → Require signed commits
```

**Impact:** Ensures all commits are cryptographically signed with GPG keys.

**Trade-off:** Adds friction for contributors who don't have GPG keys set up.

---

## Medium Priority Improvements

### 6. Token Permissions in Workflows ✅ (Mostly Done)

**Status:** Already using explicit permissions in workflows

**Example (already implemented):**
```yaml
permissions:
  contents: read
  packages: write
  security-events: write
  id-token: write
  attestations: write
```

✅ Good practice already followed!

---

### 7. Fuzzing (Optional for Go projects)

**Status:** Not implemented

**Recommendation:** Add OSS-Fuzz integration if project handles untrusted input.

**Example:** Create `.github/workflows/fuzz.yml`:
```yaml
name: Fuzz Testing
on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  fuzz:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: '.env'
      - name: Run fuzz tests
        run: go test -fuzz=. -fuzztime=1m ./...
```

**Note:** Only add if you have fuzz tests written.

---

### 8. SAST (Static Application Security Testing) ✅ (Done)

✅ CodeQL already configured and running

---

### 9. Dependency Update Tool ✅ (Done)

✅ Dependabot already configured for:
- Go modules
- GitHub Actions
- Docker base images

---

## Quick Wins (Low Effort, High Impact)

### 10. Add Security Scanning Badge to README

Add to top of README.md:
```markdown
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/splunk/splunk-ai-operator/badge)](https://securityscorecards.dev/viewer/?uri=github.com/splunk/splunk-ai-operator)
```

### 11. Enable Private Vulnerability Reporting

```
Settings → Security → Enable private vulnerability reporting
```

This allows security researchers to privately report vulnerabilities.

### 12. Add Security Contacts

In `SECURITY.md`, ensure contact information is clear:
```markdown
## Security Contact

- Email: security@splunk.com
- Private vulnerability reporting: Use GitHub's private reporting feature
```

---

## Scorecard Checks Explained

| Check | Current Status | Priority | Action Required |
|-------|---------------|----------|-----------------|
| **Binary-Artifacts** | ✅ Pass | Low | None - no binaries in repo |
| **Branch-Protection** | ⚠️ Needs Config | HIGH | Configure in GitHub Settings |
| **CI-Tests** | ✅ Pass | Low | Already have CI |
| **CII-Best-Practices** | ❌ Not registered | Medium | Register at bestpractices.coreinfrastructure.org |
| **Code-Review** | ✅ Pass | Low | Using PRs |
| **Contributors** | ✅ Pass | Low | Multiple contributors |
| **Dangerous-Workflow** | ✅ Pass | Low | Workflows are safe |
| **Dependency-Update-Tool** | ✅ Pass | Low | Dependabot configured |
| **Fuzzing** | ❌ Not implemented | Low | Optional - add if needed |
| **License** | ✅ Pass | Low | Apache 2.0 present |
| **Maintained** | ✅ Pass | Low | Recent commits |
| **Pinned-Dependencies** | ⚠️ Partial | HIGH | Pin GitHub Actions by SHA |
| **Packaging** | ✅ Pass | Low | Published to GHCR |
| **SAST** | ✅ Pass | Low | CodeQL running |
| **Security-Policy** | ✅ Pass | Low | SECURITY.md present |
| **Signed-Releases** | ❌ Not signed | Medium | Add artifact signing |
| **Token-Permissions** | ✅ Pass | Low | Explicit permissions used |
| **Vulnerabilities** | ✅ Pass | Low | No known vulnerabilities |
| **Webhooks** | ✅ Pass | Low | No webhooks configured |

---

## Implementation Priority

### Week 1 (Quick Wins):
1. ✅ Add scorecard.yml workflow (done)
2. Configure branch protection in GitHub UI
3. Add OpenSSF Scorecard badge to README
4. Enable private vulnerability reporting

### Week 2 (Security Hardening):
5. Pin GitHub Actions by commit SHA (or enable Dependabot updates)
6. Add CODEOWNERS file
7. Register for CII Best Practices badge

### Week 3+ (Optional Enhancements):
8. Enable signed commits (if required)
9. Add fuzzing tests (if handling untrusted input)
10. Add release signing with cosign

---

## Monitoring Your Score

Once the scorecard workflow runs:

1. **View in GitHub:**
   - Go to Security → Code scanning → Scorecard results

2. **Public Dashboard:**
   - Visit: https://securityscorecards.dev/viewer/?uri=github.com/splunk/splunk-ai-operator

3. **Badge:**
   - Add badge to README to show your score publicly

---

## Expected Score After Improvements

| Category | Current | After Quick Wins | After Full Implementation |
|----------|---------|------------------|---------------------------|
| Quick estimate | ~6.5/10 | ~8.0/10 | ~9.5/10 |

**Note:** Perfect 10/10 is rare and often not practical. Scores of 8+ are considered excellent.

---

## Resources

- OpenSSF Scorecard Docs: https://github.com/ossf/scorecard
- Best Practices Badge: https://bestpractices.coreinfrastructure.org/
- GitHub Security Guides: https://docs.github.com/en/code-security
- Dependabot for Actions: https://docs.github.com/en/code-security/dependabot/working-with-dependabot/keeping-your-actions-up-to-date-with-dependabot

---

**Next Steps:**
1. Merge the scorecard.yml workflow
2. Configure branch protection (requires GitHub admin)
3. Wait for scorecard to run (weekly + on push to main)
4. Review results and implement high-priority items
5. Add badge to README once score is acceptable
