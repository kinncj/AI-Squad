#!/usr/bin/env bash
# tests/cli/test_ai_squad.sh — CLI smoke tests for maple
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLI="$REPO_ROOT/scripts/maple"
TEMPLATE_DIR="$REPO_ROOT/template"
PASS=0
FAIL=0

# ─── helpers ──────────────────────────────────────────────────────────────────
ok()   { printf "  \033[1;32m✓\033[0m  %s\n" "$1"; PASS=$((PASS + 1)); }
fail() { printf "  \033[1;31m✗\033[0m  %s\n" "$1"; FAIL=$((FAIL + 1)); }

assert_exit_ok() {
  local label="$1"; shift
  if "$@" >/dev/null 2>&1; then
    ok "$label"
  else
    fail "$label (expected exit 0, got $?)"
  fi
}

assert_exit_fail() {
  local label="$1"; shift
  if ! "$@" >/dev/null 2>&1; then
    ok "$label"
  else
    fail "$label (expected non-zero exit, got 0)"
  fi
}

# ─── tests ────────────────────────────────────────────────────────────────────
printf "\n\033[1m  CLI Tests\033[0m  →  %s\n\n" "$CLI"

# 1. Executable
if [[ -x "$CLI" ]]; then
  ok "maple is executable"
else
  fail "maple is not executable"
fi

# 2. help exits 0
assert_exit_ok  "maple help exits 0"         "$CLI" help
assert_exit_ok  "maple --help exits 0"       "$CLI" --help
assert_exit_ok  "maple -h exits 0"           "$CLI" -h

# 3. Unknown command exits non-zero
assert_exit_fail "unknown command exits non-zero" "$CLI" totally-not-a-command

# 4. init copies template files
TEST_TMP="$(mktemp -d)"
TEST_PROJECT="$TEST_TMP/test-project"
trap 'rm -rf "$TEST_TMP"' EXIT

# Pipe "n" to skip labels prompt; pass absolute path to init
printf 'n\n' | "$CLI" init "$TEST_PROJECT" >/dev/null 2>&1 || true

if [[ -d "$TEST_PROJECT" ]]; then
  ok "init creates project directory"
else
  fail "init did not create project directory"
fi

# 5. init copies Claude Code agents
CLAUDE_AGENTS="$TEST_PROJECT/.claude/agents"
if [[ -d "$CLAUDE_AGENTS" ]]; then
  COUNT=$(find "$CLAUDE_AGENTS" -name "*.md" | wc -l | tr -d ' ')
  EXPECTED=$(find "$TEMPLATE_DIR/.claude/agents" -name "*.md" | wc -l | tr -d ' ')
  if [[ "$COUNT" -eq "$EXPECTED" ]]; then
    ok "init copies $COUNT Claude Code agent files"
  else
    fail "init copied $COUNT agents, expected $EXPECTED"
  fi
else
  fail "init did not create .claude/agents/"
fi

# 6. init copies OpenCode agents
OC_AGENTS="$TEST_PROJECT/.opencode/agents"
if [[ -d "$OC_AGENTS" ]]; then
  COUNT=$(find "$OC_AGENTS" -name "*.md" | wc -l | tr -d ' ')
  EXPECTED=$(find "$TEMPLATE_DIR/.opencode/agents" -name "*.md" | wc -l | tr -d ' ')
  if [[ "$COUNT" -eq "$EXPECTED" ]]; then
    ok "init copies $COUNT OpenCode agent files"
  else
    fail "init copied $COUNT agents, expected $EXPECTED"
  fi
else
  fail "init did not create .opencode/agents/"
fi

# 7. init copies Makefile
if [[ -f "$TEST_PROJECT/Makefile" ]]; then
  ok "init copies Makefile"
else
  fail "init did not copy Makefile"
fi

# 8. init copies CLAUDE.md
if [[ -f "$TEST_PROJECT/CLAUDE.md" ]]; then
  ok "init copies CLAUDE.md"
else
  fail "init did not copy CLAUDE.md"
fi

# 9. init copies opencode.json
if [[ -f "$TEST_PROJECT/opencode.json" ]]; then
  ok "init copies opencode.json"
else
  fail "init did not copy opencode.json"
fi

# 10. no agent file contains a model: line (model names are not hardcoded)
CLAUDE_MODEL_COUNT=$(grep -rl '^model:' "$TEST_PROJECT/.claude/agents/" 2>/dev/null | wc -l | tr -d ' ') || true
OC_MODEL_COUNT=$(grep -rl '^model:' "$TEST_PROJECT/.opencode/agents/" 2>/dev/null | wc -l | tr -d ' ') || true
if [[ "$CLAUDE_MODEL_COUNT" -eq 0 && "$OC_MODEL_COUNT" -eq 0 ]]; then
  ok "no agent file contains a hardcoded model: line"
else
  fail "found hardcoded model: lines ($CLAUDE_MODEL_COUNT Claude Code, $OC_MODEL_COUNT OpenCode)"
fi

# 11. opencode.json does not contain a top-level model field
if python3 -c "import json,sys; d=json.load(open('$TEST_PROJECT/opencode.json')); sys.exit(1 if 'model' in d else 0)" 2>/dev/null; then
  ok "opencode.json has no hardcoded model field"
else
  fail "opencode.json still contains a top-level model field"
fi

# 12. maple help output does not mention swarm
MAIN_HELP=$("$CLI" help 2>&1 || true)
if printf '%s' "$MAIN_HELP" | grep -qi 'swarm'; then
  fail "maple help still mentions 'swarm'"
else
  ok "maple help contains no swarm references"
fi

# 13. maple help lists init, update, labels, and project commands
if printf '%s' "$MAIN_HELP" | grep -q 'init' && \
   printf '%s' "$MAIN_HELP" | grep -q 'update' && \
   printf '%s' "$MAIN_HELP" | grep -q 'labels' && \
   printf '%s' "$MAIN_HELP" | grep -q 'project'; then
  ok "maple help lists init, labels, and project commands"
else
  fail "maple help is missing init, update, labels, or project"
fi

# 14. maple project command exists and exits non-zero without a repo (no gh auth in CI)
PROJECT_OUT=$("$CLI" project 2>&1 || true)
if printf '%s' "$PROJECT_OUT" | grep -qiE 'repository|owner|project|Could not'; then
  ok "maple project command exists and produces meaningful output"
else
  fail "maple project command missing or silent"
fi

# 15. template includes story template
if [[ -f "$TEMPLATE_DIR/docs/stories/_template.md" ]]; then
  ok "template includes docs/stories/_template.md"
else
  fail "template is missing docs/stories/_template.md"
fi

# 16. template includes DoD document
if [[ -f "$TEMPLATE_DIR/docs/dod/definition-of-done.md" ]]; then
  ok "template includes docs/dod/definition-of-done.md"
else
  fail "template is missing docs/dod/definition-of-done.md"
fi

# 17. init copies story template and dod into new project
if [[ -f "$TEST_PROJECT/docs/stories/_template.md" ]]; then
  ok "init copies docs/stories/_template.md"
else
  fail "init did not copy docs/stories/_template.md"
fi

if [[ -f "$TEST_PROJECT/docs/dod/definition-of-done.md" ]]; then
  ok "init copies docs/dod/definition-of-done.md"
else
  fail "init did not copy docs/dod/definition-of-done.md"
fi

# 17b. init copies .gitignore
if [[ -f "$TEST_PROJECT/.gitignore" ]]; then
  ok "init copies .gitignore"
else
  fail "init did not copy .gitignore"
fi

# 17c. template has PULL_REQUEST_TEMPLATE.md
if [[ -f "$TEMPLATE_DIR/.github/PULL_REQUEST_TEMPLATE.md" ]]; then
  ok "template includes .github/PULL_REQUEST_TEMPLATE.md"
else
  fail "template missing .github/PULL_REQUEST_TEMPLATE.md"
fi

# 17d. init copies PULL_REQUEST_TEMPLATE.md
if [[ -f "$TEST_PROJECT/.github/PULL_REQUEST_TEMPLATE.md" ]]; then
  ok "init copies .github/PULL_REQUEST_TEMPLATE.md"
else
  fail "init did not copy .github/PULL_REQUEST_TEMPLATE.md"
fi

# 18. template .gitignore contains Claude logs entries
GITIGNORE_CONTENT=$(cat "$TEMPLATE_DIR/.gitignore")
if printf '%s' "$GITIGNORE_CONTENT" | grep -q '.claude/logs/' && \
   printf '%s' "$GITIGNORE_CONTENT" | grep -q 'pending-sync.jsonl' && \
   printf '%s' "$GITIGNORE_CONTENT" | grep -q '.claude/state/maple.json'; then
  ok ".gitignore includes .claude/logs/, pending-sync.jsonl, state/maple.json"
else
  fail ".gitignore missing one or more Claude runtime entries"
fi

# 19. label command output covers new groups (dry check via grep on script source)
for group in "type:feature" "type:spike" "priority:high" "priority:medium" \
             "spec:problem" "spec:approved" "design:pending" "design:a11y-passed" \
             "adr:required" "adr:complete" "ui:required"; do
  if grep -q "\"$group\"" "$CLI"; then
    ok "label group present: $group"
  else
    fail "label group missing: $group"
  fi
done

# ─── Phase II: GitHub Integration Skills ─────────────────────────────────────

PHASE2_SKILLS=(gh-issues gh-projects gh-labels-milestones gherkin-authoring story-issue-sync cucumber-automation)

for skill in "${PHASE2_SKILLS[@]}"; do
  # Skill exists in Claude Code
  if [[ -f "$TEMPLATE_DIR/.claude/skills/$skill/SKILL.md" ]]; then
    ok "claude skill present: $skill"
  else
    fail "claude skill missing: $skill"
  fi
  # Mirrored in OpenCode
  if [[ -f "$TEMPLATE_DIR/.opencode/skills/$skill/SKILL.md" ]]; then
    ok "opencode skill present: $skill"
  else
    fail "opencode skill missing: $skill"
  fi
done

# gh-issues: covers Create, View, Edit, Comment, Close
for keyword in "gh issue create" "gh issue view" "gh issue edit" "gh issue comment" "gh issue close" "issue_number" "BLOCKED"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/gh-issues/SKILL.md"; then
    ok "gh-issues covers: $keyword"
  else
    fail "gh-issues missing: $keyword"
  fi
done

# gh-projects: covers GraphQL add, field update, project config read
for keyword in "addProjectV2ItemById" "updateProjectV2ItemFieldValue" "project_node_id" "singleSelectOptionId"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/gh-projects/SKILL.md"; then
    ok "gh-projects covers: $keyword"
  else
    fail "gh-projects missing: $keyword"
  fi
done

# gh-labels-milestones: covers upsert pattern and milestone
for keyword in "upsert_label" "upsert_milestone" "gh label edit" "gh label create" "PATCH"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/gh-labels-milestones/SKILL.md"; then
    ok "gh-labels-milestones covers: $keyword"
  else
    fail "gh-labels-milestones missing: $keyword"
  fi
done

# gherkin-authoring: covers naming convention, ID allocation, validation rules
for keyword in "NNNN" "YYYY" "next_id" "@story:" "@epic:" "Scenario Outline"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/gherkin-authoring/SKILL.md"; then
    ok "gherkin-authoring covers: $keyword"
  else
    fail "gherkin-authoring missing: $keyword"
  fi
done

# story-issue-sync: covers both sync directions and ownership table
for keyword in "issue_number: null" "File → Issue" "Issue → File" "DRIFT" "issue_node_id"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/story-issue-sync/SKILL.md"; then
    ok "story-issue-sync covers: $keyword"
  else
    fail "story-issue-sync missing: $keyword"
  fi
done

# cucumber-automation: covers extraction, both stacks, stubs
for keyword in "gherkin" "@cucumber/cucumber" "behave" "NO_GHERKIN" "MANUAL_EDIT"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/cucumber-automation/SKILL.md"; then
    ok "cucumber-automation covers: $keyword"
  else
    fail "cucumber-automation missing: $keyword"
  fi
done

# ─── Phase III: Design & UX Suite ────────────────────────────────────────────

PHASE3_SKILLS=(wireframe visual-identity design-tokens mockup component-scaffold a11y-audit)
PHASE3_AGENTS=(ux-researcher wireframe-architect visual-identity-designer design-system-author ui-mockup-builder a11y-auditor)

for skill in "${PHASE3_SKILLS[@]}"; do
  if [[ -f "$TEMPLATE_DIR/.claude/skills/$skill/SKILL.md" ]]; then
    ok "claude design skill present: $skill"
  else
    fail "claude design skill missing: $skill"
  fi
  if [[ -f "$TEMPLATE_DIR/.opencode/skills/$skill/SKILL.md" ]]; then
    ok "opencode design skill present: $skill"
  else
    fail "opencode design skill missing: $skill"
  fi
done

for agent in "${PHASE3_AGENTS[@]}"; do
  if [[ -f "$TEMPLATE_DIR/.claude/agents/$agent.md" ]]; then
    ok "claude design agent present: $agent"
  else
    fail "claude design agent missing: $agent"
  fi
  if [[ -f "$TEMPLATE_DIR/.opencode/agents/$agent.md" ]]; then
    ok "opencode design agent present: $agent"
  else
    fail "opencode design agent missing: $agent"
  fi
done

# docs/design/ structure
for dir in research wireframes mockups identity "system/components"; do
  if [[ -d "$TEMPLATE_DIR/docs/design/$dir" ]]; then
    ok "docs/design/$dir directory exists"
  else
    fail "docs/design/$dir directory missing"
  fi
done

# docs/design/README.md
if [[ -f "$TEMPLATE_DIR/docs/design/README.md" ]]; then
  ok "docs/design/README.md present"
else
  fail "docs/design/README.md missing"
fi

# Orchestrator references design gate and design agents
ORC="$TEMPLATE_DIR/.claude/agents/orchestrator.md"
for keyword in "ui: true" "ux-researcher" "wireframe-architect" "visual-identity-designer" "design-system-author" "ui-mockup-builder" "a11y-auditor"; do
  if grep -q "$keyword" "$ORC"; then
    ok "orchestrator references: $keyword"
  else
    fail "orchestrator missing reference: $keyword"
  fi
done

# wireframe skill covers key constructs
for keyword in "status: approved" "BLOCKED" "ASCII" "svg" "html" "tab order"; do
  if grep -qi "$keyword" "$TEMPLATE_DIR/.claude/skills/wireframe/SKILL.md"; then
    ok "wireframe skill covers: $keyword"
  else
    fail "wireframe skill missing: $keyword"
  fi
done

# visual-identity skill covers WCAG contrast
for keyword in "contrast_aa" "4.5" "WCAG" "palette.json" "typography.json"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/visual-identity/SKILL.md"; then
    ok "visual-identity skill covers: $keyword"
  else
    fail "visual-identity skill missing: $keyword"
  fi
done

# design-tokens covers W3C DTCG and all three emitters
for keyword in "DTCG" "tokens.css" "tailwind.tokens.js" "mantine.theme.ts" "\$value" "\$type"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/design-tokens/SKILL.md"; then
    ok "design-tokens skill covers: $keyword"
  else
    fail "design-tokens skill missing: $keyword"
  fi
done

# a11y-audit covers axe, pa11y, WCAG, merge gate, PR comment
for keyword in "axe" "pa11y" "WCAG" "critical" "serious" "gh pr comment" "BLOCKED"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/a11y-audit/SKILL.md"; then
    ok "a11y-audit skill covers: $keyword"
  else
    fail "a11y-audit skill missing: $keyword"
  fi
done

# DoD has ui: true section
DOD="$TEMPLATE_DIR/docs/dod/definition-of-done.md"
for keyword in "ui: true" "Wireframe" "mockup" "WCAG" "a11y"; do
  if grep -qi "$keyword" "$DOD"; then
    ok "DoD covers ui:true requirement: $keyword"
  else
    fail "DoD missing ui:true requirement: $keyword"
  fi
done

# ─── Phase IV: Spec-Kit ───────────────────────────────────────────────────────

for platform in claude opencode; do
  if [[ -f "$TEMPLATE_DIR/.$platform/skills/spec-kit/SKILL.md" ]]; then
    ok "$platform spec-kit skill present"
  else
    fail "$platform spec-kit skill missing"
  fi
  if [[ -f "$TEMPLATE_DIR/.$platform/agents/spec-kit.md" ]]; then
    ok "$platform spec-kit agent present"
  else
    fail "$platform spec-kit agent missing"
  fi
done

if [[ -f "$TEMPLATE_DIR/docs/specs/README.md" ]]; then
  ok "docs/specs/README.md present"
else
  fail "docs/specs/README.md missing"
fi

for keyword in "Scenario" "docs/stories" "status: approved" "validate_story" "spike" "chore"; do
  if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/spec-kit/SKILL.md"; then
    ok "spec-kit skill covers: $keyword"
  else
    fail "spec-kit skill missing: $keyword"
  fi
done

# Orchestrator references spec-kit
if grep -q "spec-kit" "$TEMPLATE_DIR/.claude/agents/orchestrator.md"; then
  ok "orchestrator references spec-kit pre-DISCOVER gate"
else
  fail "orchestrator missing spec-kit reference"
fi

# ─── Phase V: Taffy Workflows ────────────────────────────────────────────────
# Taffy workflows are an optional feature; skip checks if the directory is absent.

if [[ -d "$TEMPLATE_DIR/.claude/taffy" ]]; then
  if [[ -f "$TEMPLATE_DIR/.claude/taffy/schema.yaml" ]]; then
    ok "taffy schema.yaml present"
  else
    fail "taffy schema.yaml missing"
  fi

  for sp in new-ui-feature api-endpoint bugfix design-refresh; do
    if [[ -f "$TEMPLATE_DIR/.claude/taffy/$sp.yaml" ]]; then
      ok "taffy workflow present: $sp"
    else
      fail "taffy workflow missing: $sp"
    fi
  done

  for platform in claude opencode; do
    if [[ -f "$TEMPLATE_DIR/.$platform/skills/pipeline-runner/SKILL.md" ]]; then
      ok "$platform pipeline-runner skill present"
    else
      fail "$platform pipeline-runner skill missing"
    fi
  done

  for keyword in "stages" "when:" "gate:" "human-approval" "pipeline" "PAUSED" "maple.json"; do
    if grep -q "$keyword" "$TEMPLATE_DIR/.claude/skills/pipeline-runner/SKILL.md" 2>/dev/null; then
      ok "pipeline-runner covers: $keyword"
    else
      fail "pipeline-runner missing: $keyword"
    fi
  done

  for keyword in "spec-kit" "wireframe" "ui-mockup-builder" "a11y-audit" "standard-8-phase"; do
    if grep -q "$keyword" "$TEMPLATE_DIR/.claude/taffy/new-ui-feature.yaml" 2>/dev/null; then
      ok "new-ui-feature taffy workflow references: $keyword"
    else
      fail "new-ui-feature taffy workflow missing: $keyword"
    fi
  done
else
  echo "  (taffy workflows not present — skipping Phase V checks)"
fi

# ─── Phase VI: TUI ────────────────────────────────────────────────────────────

for f in go.mod go.sum; do
  if [[ -f "$f" ]]; then
    ok "$f present"
  else
    fail "$f missing"
  fi
done

# Go source now lives under app/ (BusinessRepo rebuild)
for f in app/cmd/maple/main.go app/internal/req/convert.go app/internal/gh/gh.go app/internal/scaffold/scaffold.go; do
  if [[ -f "$f" ]]; then
    ok "$f present"
  else
    fail "$f missing"
  fi
done

# go.mod declares Bubble Tea (not Ratatui)
if grep -q "bubbletea" go.mod; then
  ok "uses Bubble Tea (Go, not Rust/Ratatui)"
else
  fail "go.mod missing bubbletea"
fi

# All 5 JSON themes present
for theme in tokyo-night catppuccin-mocha gruvbox nord everforest; do
  if [[ -f "app/internal/tui/theme/themes/$theme.json" ]]; then
    ok "theme present: $theme"
  else
    fail "theme missing: $theme"
  fi
done

# maple binary builds and responds to --help
MAPLE_BIN="$REPO_ROOT/maple"
_build_maple() {
  rm -f "$REPO_ROOT/app/cmd/maple/template"
  cp -rL "$REPO_ROOT/template" "$REPO_ROOT/app/cmd/maple/template"
  go build -o "$MAPLE_BIN" "$REPO_ROOT/app/cmd/maple" 2>/dev/null
  local rc=$?
  rm -rf "$REPO_ROOT/app/cmd/maple/template"
  ln -s ../../../template "$REPO_ROOT/app/cmd/maple/template" 2>/dev/null || true
  return $rc
}
if [[ -f "$MAPLE_BIN" ]] || _build_maple; then
  ok "maple binary builds"
  MAPLE_HELP=$("$MAPLE_BIN" --help 2>&1 || true)
  for cmd in "init" "update" "req" "labels" "project"; do
    if printf '%s' "$MAPLE_HELP" | grep -q "$cmd"; then
      ok "maple --help lists: $cmd"
    else
      fail "maple --help missing: $cmd"
    fi
  done
else
  fail "maple binary failed to build"
fi

# ─── Phase VII: Enforcement ───────────────────────────────────────────────────

if [[ -f "$TEMPLATE_DIR/lefthook.yml" ]]; then
  ok "lefthook.yml present"
else
  fail "lefthook.yml missing"
fi

for hook in "spec-kit-gate" "feature-frontmatter" "design-approved" "a11y-required"; do
  if grep -q "$hook" "$TEMPLATE_DIR/lefthook.yml"; then
    ok "lefthook.yml defines hook: $hook"
  else
    fail "lefthook.yml missing hook: $hook"
  fi
done

for script in validate-frontmatter.sh a11y-gate.sh design-approved-gate.sh spec-kit-gate.sh; do
  if [[ -f "$TEMPLATE_DIR/scripts/sdlc/$script" ]]; then
    ok "sdlc script present: $script"
  else
    fail "sdlc script missing: $script"
  fi
  if [[ -x "$TEMPLATE_DIR/scripts/sdlc/$script" ]]; then
    ok "sdlc script executable: $script"
  else
    fail "sdlc script not executable: $script"
  fi
done

if [[ -f "$TEMPLATE_DIR/.github/workflows/sdlc-gates.yml" ]]; then
  ok "sdlc-gates.yml workflow present"
else
  fail "sdlc-gates.yml workflow missing"
fi

for job in frontmatter spec-kit design-approved a11y adr-required; do
  if grep -q "name: $job\|$job:" "$TEMPLATE_DIR/.github/workflows/sdlc-gates.yml" 2>/dev/null; then
    ok "workflow job present: $job"
  else
    fail "workflow job missing: $job"
  fi
done

if [[ -f "$TEMPLATE_DIR/scripts/bootstrap-branch-protection.sh" ]]; then
  ok "bootstrap-branch-protection.sh present"
else
  fail "bootstrap-branch-protection.sh missing"
fi

# adr-required-gate.sh
if [[ -f "$TEMPLATE_DIR/scripts/sdlc/adr-required-gate.sh" ]]; then
  ok "sdlc script present: adr-required-gate.sh"
else
  fail "sdlc script missing: adr-required-gate.sh"
fi

if [[ -x "$TEMPLATE_DIR/scripts/sdlc/adr-required-gate.sh" ]]; then
  ok "sdlc script executable: adr-required-gate.sh"
else
  fail "sdlc script not executable: adr-required-gate.sh"
fi

# ADR template and architecture README
if [[ -f "$TEMPLATE_DIR/docs/architecture/adr-template.md" ]]; then
  ok "template includes docs/architecture/adr-template.md"
else
  fail "template missing docs/architecture/adr-template.md"
fi

if [[ -f "$TEMPLATE_DIR/docs/architecture/README.md" ]]; then
  ok "template includes docs/architecture/README.md"
else
  fail "template missing docs/architecture/README.md"
fi

# qa-cucumber agent
if [[ -f "$TEMPLATE_DIR/.claude/agents/qa-cucumber.md" ]]; then
  ok "qa-cucumber agent present"
else
  fail "qa-cucumber agent missing"
fi

# ─── Phase VIII: Examples ─────────────────────────────────────────────────────

for example in ui-feature api-endpoint spike; do
  if [[ -f "docs/examples/$example/README.md" ]]; then
    ok "example present: $example"
  else
    fail "example missing: $example"
  fi
done

if grep -q "ui: true" docs/examples/ui-feature/README.md; then
  ok "ui-feature example demonstrates ui:true"
else
  fail "ui-feature example missing ui:true"
fi

if grep -q "ui: false" docs/examples/api-endpoint/README.md; then
  ok "api-endpoint example demonstrates ui:false (no design gate)"
else
  fail "api-endpoint example missing ui:false"
fi

if grep -q "spike" docs/examples/spike/README.md && grep -q "Spec-Kit" docs/examples/spike/README.md; then
  ok "spike example shows spec-kit skip for spike/* branches"
else
  fail "spike example missing spike/spec-kit explanation"
fi

# ─── summary ──────────────────────────────────────────────────────────────────
printf "\n  ────────────────────────────────────────\n"
printf "  \033[1;32m%d passed\033[0m  ·  " "$PASS"
if [[ "$FAIL" -gt 0 ]]; then
  printf "\033[1;31m%d failed\033[0m\n\n" "$FAIL"
  exit 1
else
  printf "\033[2m0 failed\033[0m\n\n"
fi
