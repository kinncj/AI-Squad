#!/usr/bin/env bash
# scripts/bootstrap-branch-protection.sh
# Sets branch protection rules for main using gh CLI.
# Run once after repo creation.
# Usage: ./scripts/bootstrap-branch-protection.sh [owner/repo]
set -euo pipefail

REPO="${1:-$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null)}"
[ -z "$REPO" ] && { echo "Usage: $0 owner/repo"; exit 1; }

echo "Bootstrapping branch protection for: $REPO (branch: main)"

gh api "repos/$REPO/branches/main/protection" \
  -X PUT \
  -H "Accept: application/vnd.github+json" \
  --input - <<'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "frontmatter",
      "spec-kit",
      "design-approved",
      "a11y"
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1,
    "dismiss_stale_reviews": true
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_linear_history": true
}
EOF

echo "Branch protection set on main."
echo "Required checks: frontmatter, spec-kit, design-approved, a11y"
