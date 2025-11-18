#!/usr/bin/env bash

# Script to close irrelevant Dependabot issues with explanations
# This script closes GitHub Actions update issues since we pin to commit hashes
# and closes the invalid Golang issue

set -euo pipefail

# Check for GitHub CLI
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed"
    echo "Install with: brew install gh"
    exit 1
fi

# Check authentication
if ! gh auth status &> /dev/null; then
    echo "Error: Not authenticated with GitHub"
    echo "Run: gh auth login"
    exit 1
fi

echo "Closing irrelevant Dependabot issues..."
echo ""

# Issues to close with specific reasons
declare -A ISSUES_TO_CLOSE=(
    # GitHub Actions issues - we pin to commit hashes
    ["41"]="docker/setup-buildx-action"
    ["40"]="actions/setup-go"
    ["37"]="actions/checkout"
    ["36"]="aws-actions/amazon-ecr-login"
    ["35"]="actions/setup-python"
    ["33"]="actions/attest-build-provenance"
    ["32"]="helm/chart-testing-action"
    ["31"]="docker/login-action"
    ["30"]="docker/build-push-action"
    ["29"]="peter-evans/create-pull-request"

    # Invalid Golang issue
    ["34"]="golang (invalid - incorrect version)"
)

COMMENT_ACTIONS="Closing this issue because we pin GitHub Actions to specific commit hashes for supply chain security (see [pinned actions policy](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#using-third-party-actions)).

**Why we pin to commit hashes:**
- Protects against tag hijacking
- Ensures reproducible builds
- Follows SLSA best practices
- See our [security policy](../blob/main/.github/workflows/) for details

**What we do instead:**
- Pin actions to commit SHAs (e.g., \`uses: actions/checkout@08eba0b2...\`)
- Review updates manually when needed
- Update hashes in controlled manner

This automated version bump is not applicable to our workflow. We'll review action updates as part of our regular security review process.

Closed by automation script. Related to Dependabot configuration update in #XXX."

COMMENT_GOLANG="Closing this issue because it appears to be based on incorrect version information.

**Current state:**
- Our \`.env\` file specifies: \`GO_VERSION=1.23.0\`
- The issue suggests bumping from 1.24 to 1.25
- We are not on Go 1.24

**Our Go version policy:**
- We follow the Go release schedule
- We upgrade Go versions in controlled releases
- Current version (1.23.0) is tested and validated

This appears to be a false positive or misconfiguration in Dependabot. We'll upgrade Go as part of our regular release cycle when appropriate.

Closed by automation script. Related to Dependabot configuration update in #XXX."

# Function to close an issue
close_issue() {
    local issue_number=$1
    local issue_name=$2
    local comment=$3

    echo "Processing issue #${issue_number}: ${issue_name}"

    # Add comment
    echo "  Adding comment..."
    gh issue comment "${issue_number}" --body "${comment}" \
        --repo splunk/splunk-ai-operator || {
        echo "  ⚠️  Failed to add comment to #${issue_number}"
        return 1
    }

    # Close issue
    echo "  Closing issue..."
    gh issue close "${issue_number}" \
        --repo splunk/splunk-ai-operator \
        --reason "not planned" || {
        echo "  ⚠️  Failed to close #${issue_number}"
        return 1
    }

    echo "  ✅ Closed #${issue_number}"
    echo ""
}

# Close GitHub Actions issues
echo "Closing GitHub Actions update issues..."
for issue_num in 41 40 37 36 35 33 32 31 30 29; do
    issue_name="${ISSUES_TO_CLOSE[$issue_num]}"
    close_issue "$issue_num" "$issue_name" "$COMMENT_ACTIONS"
    sleep 2  # Rate limiting
done

# Close invalid Golang issue
echo "Closing invalid Golang issue..."
close_issue "34" "${ISSUES_TO_CLOSE[34]}" "$COMMENT_GOLANG"

echo ""
echo "================================================"
echo "Summary:"
echo "- Closed 11 GitHub Actions update issues"
echo "- Closed 1 invalid Golang issue"
echo "- Total: 12 issues closed"
echo ""
echo "Remaining issues (still valid):"
echo "- #49: controller-runtime (HIGH PRIORITY)"
echo "- #46-45-42: Kubernetes core (HIGH PRIORITY)"
echo "- #48: Azure SDK (MEDIUM)"
echo "- #47: testify (LOW)"
echo "- #44: cert-manager (MEDIUM)"
echo "- #43: sigs.k8s.io/yaml (LOW)"
echo "- #39: prometheus (LOW)"
echo "- #38: go-logr (LOW)"
echo ""
echo "Next steps:"
echo "1. Review tracking issue for post-v0.1.0 updates"
echo "2. Consider batching K8s updates into one PR"
echo "3. Updated Dependabot config will prevent future noise"
echo "================================================"
