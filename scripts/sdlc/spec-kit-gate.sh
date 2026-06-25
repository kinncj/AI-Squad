#!/usr/bin/env bash
# scripts/sdlc/spec-kit-gate.sh
# Validates that every story file in docs/stories/ has at least one Gherkin scenario.
# Gherkin story files ARE the spec — no PROBLEM/SPEC/PLAN/TASKS files required.
# Skips spike/*, chore/* branches.
set -euo pipefail

BRANCH=$(git branch --show-current 2>/dev/null || echo "")
if echo "$BRANCH" | grep -qE '^(spike|chore)/'; then
  echo "[gherkin-gate] SKIP  branch=$BRANCH"
  exit 0
fi

STORIES=$(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null || true)

if [ -z "$STORIES" ]; then
  echo "[gherkin-gate] SKIP  no story files found"
  exit 0
fi

FAIL=0

for story in $STORIES; do
  [ -f "$story" ] || continue

  if ! grep -qE "^\s*(Scenario|Scenario Outline):" "$story" 2>/dev/null; then
    echo "[gherkin-gate] FAIL  $story — no Gherkin scenarios found (add Scenario: blocks)"
    FAIL=1
  else
    SCENARIO_COUNT=$(grep -cE "^\s*(Scenario|Scenario Outline):" "$story" 2>/dev/null || echo 0)
    echo "[gherkin-gate] OK    $story  (${SCENARIO_COUNT} scenario(s))"
  fi
done

exit $FAIL
