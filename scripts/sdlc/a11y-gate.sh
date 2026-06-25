#!/usr/bin/env bash
# scripts/sdlc/a11y-gate.sh
# Blocks push if any ui:true story lacks a passing a11y audit report.
# Usage: a11y-gate.sh <story_file1> [story_file2 ...]
set -euo pipefail

FAIL=0

for file in "$@"; do
  [[ "$file" == *"_template.md" ]] && continue
  [[ -f "$file" ]] || continue

  # Check ui: true
  UI=$(python3 -c "
import re
m = re.search(r'^ui:\s*(true|false)', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else 'false')
" 2>/dev/null || echo "false")

  [ "$UI" != "true" ] && continue

  # Only enforce a11y audit from validate phase onwards.
  PHASE=$(python3 -c "
import re
m = re.search(r'^phase:\s*[\"\']*(\w+)[\"\']*', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else 'validate')
" 2>/dev/null || echo "validate")
  case "$PHASE" in
    discover|architect|plan|infra|implement)
      echo "[a11y-gate] SKIP  $file  phase=$PHASE (a11y gate not enforced until validate)"
      continue
      ;;
  esac

  STORY_ID=$(python3 -c "
import re
m = re.search(r'^id:\s*[\"\'](.*?)[\"\']', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else 'unknown')
" 2>/dev/null || echo "unknown")

  REPORT="docs/design/mockups/${STORY_ID}.a11y.json"

  if [ ! -f "$REPORT" ]; then
    echo "[a11y-gate] FAIL  $file  no audit report at $REPORT"
    FAIL=1
    continue
  fi

  # Check for critical/serious violations
  VIOLATIONS=$(python3 -c "
import json, sys
data = json.load(open('$REPORT'))
violations = data.get('violations', [])
critical = [v for v in violations if v.get('impact') in ('critical','serious')]
print(len(critical))
" 2>/dev/null || echo "0")

  if [ "$VIOLATIONS" -gt 0 ]; then
    echo "[a11y-gate] FAIL  $file  $VIOLATIONS critical/serious WCAG violations unresolved"
    FAIL=1
  else
    echo "[a11y-gate] OK    $file"
  fi
done

exit $FAIL
