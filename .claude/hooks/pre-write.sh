#!/usr/bin/env bash
# .claude/hooks/pre-write.sh
# PreToolUse[Write|Edit] — runs before Claude writes or edits a file.
# Blocks writes to app/ and tests/ when no MAPLE pipeline is active.
# Exit 2 + message to BLOCK. Exit 0 to allow.

INPUT=$(cat)
FILE=$(python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    ti = d.get('tool_input', d)
    print(ti.get('file_path', ti.get('path', '')))
except Exception:
    print('')
" <<< "$INPUT" 2>/dev/null || echo "")

[ -z "$FILE" ] && exit 0

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
REL="${FILE#$ROOT/}"

# Only guard implementation paths — docs, state, scripts, config are always allowed
case "$REL" in
  app/*|tests/*)
    ;;
  *)
    exit 0
    ;;
esac

STATE="$ROOT/.claude/state/maple.json"
if [ ! -f "$STATE" ]; then
  echo "BLOCKED: No active MAPLE pipeline. Start one before writing to '$REL'."
  echo "  /pipeline-runner implement-stories   — implement existing approved stories"
  echo "  /pipeline-runner new-ui-feature      — full UI pipeline with design gates"
  echo "  /pipeline-runner api-endpoint        — API feature pipeline"
  echo "  /pipeline-runner bugfix              — reproduce → fix → validate"
  exit 2
fi

STATUS=$(python3 -c "
import sys, json
try:
    print(json.load(open('$STATE')).get('status', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")

case "$STATUS" in
  RUNNING|PAUSED)
    exit 0
    ;;
  *)
    echo "BLOCKED: MAPLE pipeline is not active (status=${STATUS:-none}). Cannot write to '$REL' outside a running pipeline."
    echo "  Resume or start: /pipeline-runner <name>"
    exit 2
    ;;
esac
