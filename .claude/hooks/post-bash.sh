#!/usr/bin/env bash
# .claude/hooks/post-bash.sh
# PostToolUse[Bash] — runs after every shell command Claude executes.
# Surfaces pass/fail signals and syncs TDD/phase status to GitHub Issues.

INPUT=$(cat)

COMMAND=$(python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print(d.get('tool_input', d).get('command', ''))
except Exception:
    print('')
" <<< "$INPUT" 2>/dev/null || echo "")

EXIT_CODE=$(python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = d.get('tool_response', {})
    print(r.get('exit_code', r.get('exitCode', 0)))
except Exception:
    print(0)
" <<< "$INPUT" 2>/dev/null || echo "0")

# ── Helper: gh available + authenticated against a GitHub remote ──────────────
gh_ready() {
    command -v gh &>/dev/null || return 1
    gh auth status &>/dev/null || return 1
    git remote get-url origin 2>/dev/null | grep -q "github\.com" || return 1
}

# ── Helper: update TDD labels on in-progress issues ──────────────────────────
# Only comments when the status actually changes (RED→GREEN or first RED).
sync_tdd() {
    local passed="$1"   # "true" or "false"
    gh_ready || return 0

    # Fetch in-progress issues with their label names
    local data
    data=$(gh issue list \
        --label "in-progress" --state open \
        --json number,labels \
        --jq '.[] | "\(.number) \([.labels[].name] | join(","))"' 2>/dev/null) || return 0
    [ -z "$data" ] && return 0

    while IFS= read -r line; do
        local num labels
        num=$(echo "$line" | cut -d' ' -f1)
        labels=$(echo "$line" | cut -d' ' -f2-)
        [ -z "$num" ] && continue

        local has_red=false
        echo "$labels" | grep -q "tdd:red" && has_red=true

        if [ "$passed" = "true" ] && [ "$has_red" = "true" ]; then
            gh issue edit "$num" \
                --add-label "tdd:green" --remove-label "tdd:red" 2>/dev/null
            gh issue comment "$num" \
                --body "GREEN: tests now passing ✓" 2>/dev/null
            echo "[gh-sync] TDD RED→GREEN  #$num"

        elif [ "$passed" = "false" ] && [ "$has_red" = "false" ]; then
            gh issue edit "$num" \
                --add-label "tdd:red" --remove-label "tdd:green" 2>/dev/null
            gh issue comment "$num" \
                --body "RED: tests failing ✗ — fix before proceeding." 2>/dev/null
            echo "[gh-sync] TDD GREEN→RED  #$num"
        fi
    done <<< "$data"
}

# ── Helper: mark phase 8 complete on in-progress issues ──────────────────────
sync_phase8_done() {
    gh_ready || return 0
    local issues
    issues=$(gh issue list \
        --label "in-progress" --state open \
        --json number --jq '.[].number' 2>/dev/null) || return 0
    for num in $issues; do
        gh issue edit "$num" \
            --add-label "phase:done" \
            --remove-label "in-progress,tdd:red,tdd:green" 2>/dev/null
        gh issue comment "$num" \
            --body "Phase 8 FINAL GATE passed — \`make test-all\` green. PR ready." 2>/dev/null
        echo "[gh-sync] PHASE8 DONE  #$num"
    done
}

# ── Test commands ─────────────────────────────────────────────────────────────
if echo "$COMMAND" | grep -qE '(make test$|make test |pytest|jest|vitest|npx playwright|gradle test|mvn test|dotnet test)'; then
    if [ "$EXIT_CODE" -eq 0 ]; then
        echo "✓ Tests passed."
        sync_tdd "true"
    else
        echo "✗ Tests failed (exit $EXIT_CODE). QA gate NOT satisfied. Do not advance to next phase."
        sync_tdd "false"
    fi
fi

# ── Lint commands ─────────────────────────────────────────────────────────────
if echo "$COMMAND" | grep -qE '(make lint|eslint|pylint|flake8|golangci-lint|dotnet format)'; then
    if [ "$EXIT_CODE" -eq 0 ]; then
        echo "✓ Lint passed."
    else
        echo "✗ Lint failed (exit $EXIT_CODE). Fix before proceeding."
    fi
fi

# ── make test-all — Phase 8 gate ──────────────────────────────────────────────
if echo "$COMMAND" | grep -q 'make test-all'; then
    if [ "$EXIT_CODE" -eq 0 ]; then
        echo "✓ Phase 8 gate passed — all test layers green. Safe to open PR."
        sync_phase8_done
    else
        echo "✗ Phase 8 gate FAILED. Return to Phase 5 for the failing component."
        sync_tdd "false"
    fi
fi

# ── Design review portal auto-start at human approval gates ───────────────────
if [ -f ".claude/state/approval-pending.txt" ] && [ -f "scripts/design-review-portal.sh" ]; then
    bash scripts/design-review-portal.sh start >/dev/null 2>&1 || true
fi

exit 0
