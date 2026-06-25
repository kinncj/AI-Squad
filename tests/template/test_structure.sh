#!/usr/bin/env bash
# tests/template/test_structure.sh — template directory structure validation
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMPLATE="$REPO_ROOT/template"
PASS=0
FAIL=0

# ─── helpers ──────────────────────────────────────────────────────────────────
ok()   { printf "  \033[1;32m✓\033[0m  %s\n" "$1"; PASS=$((PASS + 1)); }
fail() { printf "  \033[1;31m✗\033[0m  %s\n" "$1"; FAIL=$((FAIL + 1)); }

assert_file() {
  local rel="$1"
  if [[ -f "$TEMPLATE/$rel" ]]; then
    ok "template/$rel exists"
  else
    fail "template/$rel missing"
  fi
}

assert_dir() {
  local rel="$1"
  if [[ -d "$TEMPLATE/$rel" ]]; then
    ok "template/$rel/ exists"
  else
    fail "template/$rel/ missing"
  fi
}

assert_count_gte() {
  local label="$1" actual="$2" min="$3"
  if [[ "$actual" -ge "$min" ]]; then
    ok "$label: $actual (min $min)"
  else
    fail "$label: $actual (min $min required)"
  fi
}

assert_count_eq() {
  local label="$1" actual="$2" expected="$3"
  if [[ "$actual" -eq "$expected" ]]; then
    ok "$label: $actual"
  else
    fail "$label: $actual (expected $expected)"
  fi
}

# ─── tests ────────────────────────────────────────────────────────────────────
printf "\n\033[1m  Template Structure Tests\033[0m  →  %s\n\n" "$TEMPLATE"

# Required root files
assert_file "Makefile"
assert_file "CLAUDE.md"
assert_file "AGENTS.md"
assert_file "opencode.json"
assert_file "docker-compose.test.yml"
assert_file "playwright.config.ts"

# Required directories
assert_dir ".claude/agents"
assert_dir ".claude/commands"
assert_dir ".claude/skills"
assert_dir ".opencode/agents"
assert_dir ".opencode/commands"
assert_dir ".opencode/skills"
assert_dir "infra/scripts"
# Agent counts — must be equal between platforms (use min check to stay flexible)
CLAUDE_AGENT_COUNT=$(find "$TEMPLATE/.claude/agents" -name "*.md" | wc -l | tr -d ' ')
OC_AGENT_COUNT=$(find "$TEMPLATE/.opencode/agents" -name "*.md" | wc -l | tr -d ' ')
assert_count_gte "Claude Code agent count" "$CLAUDE_AGENT_COUNT" 27
assert_count_gte "OpenCode agent count"    "$OC_AGENT_COUNT"    27

if [[ "$CLAUDE_AGENT_COUNT" -eq "$OC_AGENT_COUNT" ]]; then
  ok "Agent counts are mirrored (${CLAUDE_AGENT_COUNT} each)"
else
  fail "Agent counts differ: Claude Code=${CLAUDE_AGENT_COUNT} OpenCode=${OC_AGENT_COUNT}"
fi

# Skills — at least 17
CLAUDE_SKILL_COUNT=$(find "$TEMPLATE/.claude/skills" | wc -l | tr -d ' ')
OC_SKILL_COUNT=$(find "$TEMPLATE/.opencode/skills" | wc -l | tr -d ' ')
# subtract 1 for the directory itself
CLAUDE_SKILL_COUNT=$((CLAUDE_SKILL_COUNT - 1))
OC_SKILL_COUNT=$((OC_SKILL_COUNT - 1))
assert_count_gte "Claude Code skill count" "$CLAUDE_SKILL_COUNT" 17
assert_count_gte "OpenCode skill count"    "$OC_SKILL_COUNT"    17

# Commands — at least 5
CLAUDE_CMD_COUNT=$(find "$TEMPLATE/.claude/commands" -name "*.md" | wc -l | tr -d ' ')
OC_CMD_COUNT=$(find "$TEMPLATE/.opencode/commands" -name "*.md" | wc -l | tr -d ' ')
assert_count_gte "Claude Code command count" "$CLAUDE_CMD_COUNT" 5
assert_count_gte "OpenCode command count"    "$OC_CMD_COUNT"    5

# ─── summary ──────────────────────────────────────────────────────────────────
printf "\n  ────────────────────────────────────────\n"
printf "  \033[1;32m%d passed\033[0m  ·  " "$PASS"
if [[ "$FAIL" -gt 0 ]]; then
  printf "\033[1;31m%d failed\033[0m\n\n" "$FAIL"
  exit 1
else
  printf "\033[2m0 failed\033[0m\n\n"
fi
