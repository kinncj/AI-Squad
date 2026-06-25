#!/usr/bin/env bash
# .claude/hooks/pre-bash.sh
# PreToolUse[Bash] — runs before every shell command Claude executes.
# Receives tool input as JSON on stdin.
# Exit 2 + stdout message to BLOCK the command.
# Exit 0 to allow it through.

INPUT=$(cat)
COMMAND=$(python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print(d.get('tool_input', d).get('command', ''))
except Exception:
    print('')
" <<< "$INPUT" 2>/dev/null || echo "")

# ── Gate: block git commit / push if SDLC gates fail ─────────────────────────
if echo "$COMMAND" | grep -qE '^git (commit|push)'; then
    FAIL=0
    MESSAGES=()

    # Spec-Kit gate
    if [ -f "scripts/sdlc/spec-kit-gate.sh" ]; then
        OUT=$(bash scripts/sdlc/spec-kit-gate.sh 2>&1) || { FAIL=1; MESSAGES+=("$OUT"); }
    fi

    # Story frontmatter
    if [ -f "scripts/sdlc/validate-frontmatter.sh" ]; then
        FILES=$(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null | tr '\n' ' ')
        if [ -n "$FILES" ]; then
            OUT=$(bash scripts/sdlc/validate-frontmatter.sh $FILES 2>&1) || { FAIL=1; MESSAGES+=("$OUT"); }
        fi
    fi

    # Design-approved gate
    if [ -f "scripts/sdlc/design-approved-gate.sh" ]; then
        FILES=$(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null | tr '\n' ' ')
        if [ -n "$FILES" ]; then
            OUT=$(bash scripts/sdlc/design-approved-gate.sh $FILES 2>&1) || { FAIL=1; MESSAGES+=("$OUT"); }
        fi
    fi

    # A11y gate
    if [ -f "scripts/sdlc/a11y-gate.sh" ]; then
        FILES=$(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null | tr '\n' ' ')
        if [ -n "$FILES" ]; then
            OUT=$(bash scripts/sdlc/a11y-gate.sh $FILES 2>&1) || { FAIL=1; MESSAGES+=("$OUT"); }
        fi
    fi

    if [ "$FAIL" -ne 0 ]; then
        echo "BLOCKED: SDLC gate(s) failed — resolve before committing."
        for msg in "${MESSAGES[@]}"; do
            echo "$msg"
        done
        exit 2
    fi
fi

# ── Gate: no raw secrets about to be staged ──────────────────────────────────
if echo "$COMMAND" | grep -qE '^git (add|commit)'; then
    if git diff --cached --name-only 2>/dev/null | xargs grep -lE "(API_KEY|SECRET|PASSWORD|TOKEN)\s*=\s*['\"][^'\"]{8,}" 2>/dev/null | grep -q .; then
        echo "BLOCKED: Possible secrets in staged files. Review before committing."
        exit 2
    fi
fi

exit 0
