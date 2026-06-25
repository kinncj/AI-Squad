#!/usr/bin/env bash
# Configure branch protection on main via gh CLI.
# Requires: gh CLI authenticated with admin/owner scope on the repo.
# Usage: bash scripts/sdlc/branch-protection.sh [--branch <name>]
set -euo pipefail

BRANCH="${1:-main}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "✗ $1 not found" >&2; exit 1; }
}
require_cmd gh
require_cmd jq

REPO=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null) || {
  echo "✗ not inside a GitHub repo — run from the project root" >&2
  exit 1
}

echo "  Configuring branch protection for ${REPO}@${BRANCH}…"

# Build the protection payload
PAYLOAD=$(jq -n \
  --arg strict "true" \
  --argjson contexts '["lint","test","sdlc-gates"]' \
  --argjson required_reviews 1 \
  '{
    required_status_checks: {
      strict: true,
      contexts: $contexts
    },
    enforce_admins: true,
    required_pull_request_reviews: {
      required_approving_review_count: $required_reviews,
      dismiss_stale_reviews: true,
      require_last_push_approval: false
    },
    restrictions: null,
    required_linear_history: false,
    allow_force_pushes: false,
    allow_deletions: false
  }')

gh api \
  "repos/${REPO}/branches/${BRANCH}/protection" \
  --method PUT \
  --input - <<< "$PAYLOAD"

echo "  ✓ branch protection applied to ${BRANCH}"
echo "    · Requires 1 approving review (stale reviews dismissed)"
echo "    · Required status checks: lint, test, sdlc-gates"
echo "    · Force-push and deletion blocked"
