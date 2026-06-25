#!/usr/bin/env bash
# scripts/sdlc/rotate-logs.sh
# Rotates .claude/logs/skills.jsonl when it exceeds 10 MB.
# Keeps last 5 rotations compressed. Deletes older ones.
# Safe to run repeatedly (idempotent).
set -euo pipefail

LOG_DIR=".claude/logs"
LOG_FILE="$LOG_DIR/skills.jsonl"
MAX_BYTES=$((10 * 1024 * 1024))  # 10 MB
KEEP=5

[ -f "$LOG_FILE" ] || { echo "No log file at $LOG_FILE — nothing to rotate."; exit 0; }

size=$(wc -c < "$LOG_FILE")
if [ "$size" -lt "$MAX_BYTES" ]; then
  echo "Log size $(( size / 1024 ))KB — below 10 MB threshold, no rotation needed."
  exit 0
fi

ts=$(date +%Y%m%d-%H%M%S)
rotated="$LOG_DIR/skills.jsonl.$ts"
mv "$LOG_FILE" "$rotated"
gzip "$rotated"
touch "$LOG_FILE"
echo "Rotated $LOG_FILE → ${rotated}.gz"

# Keep only last $KEEP rotations
mapfile -t old < <(ls -t "$LOG_DIR"/skills.jsonl.*.gz 2>/dev/null | tail -n +$((KEEP + 1)))
for f in "${old[@]}"; do
  rm -f "$f"
  echo "Deleted old rotation: $f"
done
