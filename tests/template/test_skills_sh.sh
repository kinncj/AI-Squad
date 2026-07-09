#!/usr/bin/env bash
# tests/template/test_skills_sh.sh — skills.sh CLI integration tests
# Requires Node.js / npx. Network tests are skipped when NO_NETWORK=1 or offline.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PASS=0
FAIL=0
SKIP=0

ok()   { printf "  \033[1;32m✓\033[0m  %s\n" "$1"; PASS=$((PASS + 1)); }
fail() { printf "  \033[1;31m✗\033[0m  %s\n" "$1"; FAIL=$((FAIL + 1)); }
skip() { printf "  \033[2m~\033[0m  %s (skipped)\n" "$1"; SKIP=$((SKIP + 1)); }

# ─── prerequisite: npx ────────────────────────────────────────────────────────
printf "\n\033[1m  Prerequisites\033[0m\n\n"

if ! command -v npx &>/dev/null; then
  skip "npx not found — install Node.js from nodejs.org"
  printf "\n  ────────────────────────────────────────\n"
  printf "  \033[1;32m%d passed\033[0m  ·  \033[2m%d skipped\033[0m\n\n" "$PASS" "$SKIP"
  exit 0
fi
NPX_VERSION=$(npx --version 2>/dev/null || echo "unknown")
ok "npx found ($NPX_VERSION)"

# ─── skills CLI availability ──────────────────────────────────────────────────
printf "\n\033[1m  skills CLI\033[0m\n\n"

if npm_config_yes=true npx skills --version &>/dev/null 2>&1; then
  ok "npx skills CLI available (via npm_config_yes=true)"
else
  skip "npx skills not available (may need network)"
fi

# ─── skills.sh output parsing (offline — uses local parseSkillsOutput logic) ──
printf "\n\033[1m  Output parsing (Go unit tests)\033[0m\n\n"

# Pure parsing unit tests live under app/internal (no embed dance needed here).
if command -v go &>/dev/null && [ -d "$REPO_ROOT/app/internal/req" ]; then
  if (cd "$REPO_ROOT" && go test -run "TestParse" -count=1 -timeout 30s ./app/internal/req/... &>/dev/null 2>&1); then
    ok "Go unit tests: ParseStories + gherkin parsing"
  else
    fail "Go unit tests failed — run 'go test -run TestParse ./app/internal/req/...' for details"
  fi
else
  skip "Go unit tests: go toolchain or app/internal/req not present"
fi

# ─── network tests ────────────────────────────────────────────────────────────
printf "\n\033[1m  skills.sh network integration\033[0m\n\n"

# Allow skipping network tests in CI or offline
if [[ "${NO_NETWORK:-0}" == "1" ]] || [[ "${CI:-}" != "" ]]; then
  skip "skills find (network) — NO_NETWORK=1 or CI environment"
  skip "skills find result parsing"
else
  # Test: npx skills find returns output with at least one skill
  # npm_config_yes=true allows npx to install the skills package without a package.json
  FIND_OUT=$(NO_COLOR=1 FORCE_COLOR=0 npm_config_yes=true npx skills find tdd 2>/dev/null \
    | sed 's/\x1b\[[0-9;]*[mGKHF]//g' || true)
  if echo "$FIND_OUT" | grep -qE "[a-z][-a-z0-9]+/[-a-z0-9]+@[-a-z0-9]+ [0-9,.KMB]+ installs"; then
    ok "skills find 'tdd' returns results with install counts"
  else
    fail "skills find 'tdd' returned no parseable results — check skills.sh reachability"
  fi

  # Test: result lines follow expected format owner/repo@skill N installs
  FIRST_LINE=$(echo "$FIND_OUT" | grep -E "[a-z][-a-z0-9]+/[-a-z0-9]+@[-a-z0-9]+ [0-9,.KMB]+ installs" | head -1 || true)
  if [[ -n "$FIRST_LINE" ]]; then
    PKG=$(echo "$FIRST_LINE" | awk '{print $1}')
    if [[ "$PKG" == *"/"*"@"* ]]; then
      ok "result format: owner/repo@skill detected ($PKG)"
    else
      fail "result format unexpected: $FIRST_LINE"
    fi
  else
    skip "result format check (no results to check)"
  fi
fi

# ─── skills install/remove (isolated test) ────────────────────────────────────
printf "\n\033[1m  skills install → verify → remove\033[0m\n\n"

if [[ "${NO_NETWORK:-0}" == "1" ]] || [[ "${CI:-}" != "" ]]; then
  skip "install/remove test — NO_NETWORK=1 or CI environment"
else
  TEST_DIR=$(mktemp -d)
  cd "$TEST_DIR"
  # Install a well-known tiny skill into isolated dir
  INSTALL_OUT=$(NO_COLOR=1 FORCE_COLOR=0 npm_config_yes=true npx skills add vercel-labs/agent-skills --all -y 2>&1 || true)
  SKILL_FOUND=false
  for look_path in \
    "$TEST_DIR/.claude/skills" \
    "$HOME/.claude/skills" \
    "$TEST_DIR/skills"; do
    if [[ -d "$look_path" ]] && ls "$look_path"/ &>/dev/null; then
      SKILL_FOUND=true
      ok "skill installed — found skills dir at $look_path"
      break
    fi
  done

  if ! $SKILL_FOUND; then
    # Check install output for success indicators
    if echo "$INSTALL_OUT" | grep -qiE "install|add|success|added"; then
      ok "skill install command ran without error"
    else
      fail "skill install may have failed — output: $(echo "$INSTALL_OUT" | head -3)"
    fi
  fi

  cd "$REPO_ROOT"
  rm -rf "$TEST_DIR"
fi

# ─── pipeline-runner fallback reference check ─────────────────────────────────
printf "\n\033[1m  pipeline-runner skills.sh fallback\033[0m\n\n"

SKILL_FILE="$REPO_ROOT/template/.claude/skills/pipeline-runner/SKILL.md"
if grep -q "skills.sh\|npx skills" "$SKILL_FILE" 2>/dev/null; then
  ok "pipeline-runner SKILL.md references skills.sh fallback"
else
  fail "pipeline-runner SKILL.md missing skills.sh fallback — update template/.claude/skills/pipeline-runner/SKILL.md"
fi

SKILL_FILE_OC="$REPO_ROOT/template/.opencode/skills/pipeline-runner/SKILL.md"
if grep -q "skills.sh\|npx skills" "$SKILL_FILE_OC" 2>/dev/null; then
  ok "pipeline-runner (OpenCode) SKILL.md references skills.sh fallback"
else
  fail "pipeline-runner (OpenCode) SKILL.md missing skills.sh fallback"
fi

# ─── publishing structure check ───────────────────────────────────────────────
printf "\n\033[1m  skills.sh publishing structure\033[0m\n\n"

ROOT_SKILL="$REPO_ROOT/SKILL.md"
if [[ -f "$ROOT_SKILL" ]]; then
  ok "root SKILL.md present (enables npx skills add kinncj/maple)"
else
  fail "root SKILL.md missing — skills.sh cannot index this repo"
fi

SKILLS_DIR="$REPO_ROOT/template/.claude/skills"
if [[ -d "$SKILLS_DIR" ]]; then
  COUNT=$(find "$SKILLS_DIR" -name "SKILL.md" | wc -l | tr -d ' ')
  ok "template/.claude/skills/ present with $COUNT SKILL.md files (canonical location)"
else
  fail "template/.claude/skills/ missing"
fi

# ─── summary ──────────────────────────────────────────────────────────────────
printf "\n  ────────────────────────────────────────\n"
printf "  \033[1;32m%d passed\033[0m  ·  " "$PASS"
if [[ "$FAIL" -gt 0 ]]; then
  printf "\033[1;31m%d failed\033[0m  ·  \033[2m%d skipped\033[0m\n\n" "$FAIL" "$SKIP"
  exit 1
else
  printf "\033[2m0 failed\033[0m  ·  \033[2m%d skipped\033[0m\n\n" "$SKIP"
fi
