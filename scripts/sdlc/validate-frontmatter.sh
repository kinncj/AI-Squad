#!/usr/bin/env bash
# scripts/sdlc/validate-frontmatter.sh
# Validates required frontmatter fields in story files.
# Usage: validate-frontmatter.sh <file1> [file2 ...]
set -euo pipefail

REQUIRED_FIELDS=(id title epic priority ui adr_required labels)
FAIL=0

for file in "$@"; do
  [[ "$file" == *"_template.md" ]] && continue
  [[ -f "$file" ]] || continue

  for field in "${REQUIRED_FIELDS[@]}"; do
    if ! grep -q "^${field}:" "$file" && ! grep -q "^${field}: " "$file"; then
      echo "[frontmatter] FAIL  $file  missing field: $field"
      FAIL=1
    fi
  done

  # id must not be null
  ID=$(python3 -c "
import re, sys
m = re.search(r'^id:\s*[\"\'](.*?)[\"\']', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else '')
" 2>/dev/null || echo "")
  if [ -z "$ID" ]; then
    echo "[frontmatter] FAIL  $file  id is empty or null"
    FAIL=1
  fi

  # priority must be one of critical|high|medium|low
  PRIORITY=$(python3 -c "
import re, sys
m = re.search(r'^priority:\s*[\"\'](.*?)[\"\']', open('$file').read(), re.MULTILINE)
print(m.group(1) if m else '')
" 2>/dev/null || echo "")
  if ! echo "$PRIORITY" | grep -qE "^(critical|high|medium|low)$"; then
    echo "[frontmatter] FAIL  $file  priority='$PRIORITY' must be critical|high|medium|low"
    FAIL=1
  fi

  [ "$FAIL" -eq 0 ] && echo "[frontmatter] OK    $file"
done

exit $FAIL
