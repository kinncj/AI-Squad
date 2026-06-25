#!/usr/bin/env bash
# scripts/sdlc/design-approved-gate.sh
# Blocks push if any ui:true story has unapproved wireframe or mockup.
# Usage: design-approved-gate.sh <story_file1> [story_file2 ...]
set -euo pipefail

FAIL=0

check_approved() {
  local file="$1" label="$2"
  if [ ! -f "$file" ]; then
    echo "[design-gate] FAIL  $file missing: $label"
    return 1
  fi
  STATUS=$(python3 -c "
import re
m = re.search(r'^status:\s*(\w+)', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else 'draft')
" 2>/dev/null || echo "draft")
  if [ "$STATUS" != "approved" ]; then
    echo "[design-gate] FAIL  $label  status=$STATUS (requires: approved)"
    return 1
  fi
  return 0
}

for file in "$@"; do
  [[ "$file" == *"_template.md" ]] && continue
  [[ -f "$file" ]] || continue

  UI=$(python3 -c "
import re
m = re.search(r'^ui:\s*(true|false)', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else 'false')
" 2>/dev/null || echo "false")
  [ "$UI" != "true" ] && continue

  # Only enforce design approval from implement phase onwards.
  # During discover/architect/plan/infra the design sub-pipeline hasn't run yet.
  PHASE=$(python3 -c "
import re
m = re.search(r'^phase:\s*[\"\']*(\w+)[\"\']*', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else 'implement')
" 2>/dev/null || echo "implement")
  case "$PHASE" in
    discover|architect|plan|infra)
      echo "[design-gate] SKIP  $file  phase=$PHASE (design gate not enforced until implement)"
      continue
      ;;
  esac

  STORY_ID=$(python3 -c "
import re
m = re.search(r'^id:\s*[\"\'](.*?)[\"\']', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else 'unknown')
" 2>/dev/null || echo "unknown")

  WF="docs/design/wireframes/${STORY_ID}.wireframe.md"
  MK="docs/design/mockups/${STORY_ID}.mockup.md"

  check_approved "$WF" "wireframe:$STORY_ID" || FAIL=1
  check_approved "$MK" "mockup:$STORY_ID"    || FAIL=1

  [ "$FAIL" -eq 0 ] && echo "[design-gate] OK    $file"
done

exit $FAIL
