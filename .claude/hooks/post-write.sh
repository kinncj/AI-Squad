#!/usr/bin/env bash
# .claude/hooks/post-write.sh
# PostToolUse[Write|Edit] — runs after Claude writes or edits a file.
# Receives tool input+response as JSON on stdin.
# stdout is shown to Claude as context. Exit code is informational only.

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

# ── Helper: gh available and authenticated against a GitHub remote ────────────
gh_ready() {
    command -v gh &>/dev/null || return 1
    gh auth status &>/dev/null || return 1
    git remote get-url origin 2>/dev/null | grep -q "github\.com" || return 1
}

# ── Helper: sync a story file to GitHub Issues ────────────────────────────────
sync_story() {
    local f="$1"
    gh_ready || return 0

    local issue_number
    issue_number=$(python3 -c "
import re
m = re.search(r'^issue_number:\s*(\d+)', open('$f').read(), re.MULTILINE)
print(m.group(1) if m else '')
" 2>/dev/null)

    local title
    title=$(python3 -c "
import re
m = re.search(r'^title:\s*[\"\']?(.*?)[\"\']?\s*$', open('$f').read(), re.MULTILINE)
print(m.group(1).strip('\"\'') if m else '')
" 2>/dev/null)

    [ -z "$title" ] && return 0

    if [ -z "$issue_number" ]; then
        # ── Create new issue ──────────────────────────────────────────────────
        local labels
        labels=$(python3 -c "
import re
ms = re.findall(r'^\s+- [\"\'](.*?)[\"\']\s*$', open('$f').read(), re.MULTILINE)
print(','.join(ms) if ms else 'type:feature')
" 2>/dev/null)

        local result num url
        result=$(gh issue create \
            --title "$title" \
            --body-file "$f" \
            --label "$labels" \
            --json number,url 2>/dev/null) || {
            echo "[gh-sync] WARN  could not create issue for $f (gh error)"
            return 0
        }
        num=$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin)['number'])" 2>/dev/null)
        url=$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin)['url'])"    2>/dev/null)

        # Write issue_number + issue_url back into story frontmatter
        python3 - << EOF 2>/dev/null
import re
text = open('$f').read()
text = re.sub(r'^issue_number: null', 'issue_number: $num',   text, flags=re.MULTILINE)
text = re.sub(r'^issue_url: null',    'issue_url: "$url"',    text, flags=re.MULTILINE)
open('$f', 'w').write(text)
EOF
        echo "[gh-sync] CREATED  #$num  ← $f"

    else
        # ── Update existing issue body ────────────────────────────────────────
        # Only update if the body would actually differ (avoid noise)
        local current_body new_body
        current_body=$(gh issue view "$issue_number" --json body --jq '.body' 2>/dev/null)
        new_body=$(cat "$f")
        if [ "$current_body" != "$new_body" ]; then
            gh issue edit "$issue_number" --body-file "$f" 2>/dev/null && \
                echo "[gh-sync] UPDATED  #$issue_number  ← $f"
        else
            echo "[gh-sync] SKIP    #$issue_number  no body changes"
        fi
    fi
}

# ── Story file ────────────────────────────────────────────────────────────────
if echo "$FILE" | grep -qE 'docs/stories/.+\.md$' && [[ "$FILE" != *"_template.md" ]]; then
    # 1. Validate frontmatter
    if [ -f "scripts/sdlc/validate-frontmatter.sh" ]; then
        if bash scripts/sdlc/validate-frontmatter.sh "$FILE" 2>&1; then
            echo "✓ frontmatter OK: $FILE"
        else
            echo "⚠ frontmatter issues in $FILE — fix before committing."
        fi
    fi
    # 2. GitHub sync
    sync_story "$FILE"
    exit 0
fi

# ── Source file: lint + unit tests ───────────────────────────────────────────
if echo "$FILE" | grep -qE '\.(ts|tsx|js|jsx|py|java|cs|go|rs|rb)$'; then
    if [ -f "Makefile" ]; then
        echo "Running lint..."
        make lint 2>&1 | tail -8 || echo "⚠ lint found issues"
        echo "Running unit tests..."
        make test 2>&1 | tail -15 || echo "⚠ unit tests failed — fix before moving on"
    fi
fi

exit 0
