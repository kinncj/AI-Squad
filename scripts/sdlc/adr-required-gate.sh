#!/usr/bin/env bash
# adr-required-gate.sh — block push when a story has adr_required: true
# but no corresponding ADR file exists in docs/architecture/.
#
# Passing condition:
#   For every story file where `adr_required: true`, at least one file
#   matching docs/architecture/*.md (excluding adr-template.md and README.md)
#   must exist.
#
# The check is intentionally loose: it doesn't match ADR to story by name.
# It enforces the presence of at least one ADR, not the mapping.
# More precise enforcement belongs in CI, not pre-push hooks.

set -euo pipefail

STORIES=$(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null || true)
if [ -z "$STORIES" ]; then
  echo "[adr-gate] SKIP  no story files found"
  exit 0
fi

ADR_REQUIRED=0
for story in $STORIES; do
  if grep -qiE "^adr_required:\s*true" "$story" 2>/dev/null; then
    ADR_REQUIRED=1
    echo "[adr-gate] adr_required=true in $story"
  fi
done

if [ "$ADR_REQUIRED" -eq 0 ]; then
  echo "[adr-gate] OK  no stories require an ADR"
  exit 0
fi

# Check that at least one ADR file exists (not template, not README)
ADR_FILES=$(find docs/architecture -name "*.md" \
  ! -name "adr-template.md" \
  ! -name "README.md" 2>/dev/null | wc -l | tr -d ' ')

if [ "$ADR_FILES" -eq 0 ]; then
  echo "[adr-gate] FAIL  one or more stories have adr_required: true but no ADR files found in docs/architecture/"
  echo "[adr-gate] ACTION: copy docs/architecture/adr-template.md → docs/architecture/NNNN-decision-title.md and fill it in"
  exit 1
fi

echo "[adr-gate] OK  adr_required stories present and $ADR_FILES ADR file(s) found"
exit 0
