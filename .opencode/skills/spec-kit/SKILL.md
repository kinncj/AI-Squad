---
name: spec-kit
description: "Write a Gherkin story file to docs/stories/ for a feature. The story IS the spec. No intermediate PROBLEM/SPEC/PLAN/TASKS artifacts."
---

# SKILL: spec-kit

## Purpose

Produce a complete Gherkin story file in `docs/stories/` before any implementation begins. One invocation = one story file. The story file is the minimum spec — no additional artifacts required.

## State Machine

```
feature description → story draft → human approval → DISCOVER
```

No step may begin until the previous one completes. Approval = human sets `status: approved` in the story frontmatter, or confirms verbally.

## Story File Location

```
docs/stories/{epic-slug}-{feature-slug}.md   # with epic context
docs/stories/{feature-slug}.md               # standalone feature
```

## Story File Template

```markdown
---
id: "{feature-slug}"
title: "{Feature Title}"
epic: "{epic-slug or null}"
priority: "high|medium|low"
ui: false
adr_required: false
phase: discover
labels:
  - "type:feature"
  - "priority:{priority}"
status: draft
---

## Story

**As a** {user role},
**I want** {what they want},
**so that** {business outcome}.

## Acceptance Criteria

```gherkin
@story:{feature-slug} @priority:{priority}
Feature: {Feature Title}

  Scenario: {happy path title}
    Given {precondition}
    When {action}
    Then {outcome}

  Scenario: {failure or edge case title}
    Given {precondition}
    When {action}
    Then {outcome}
```

## Definition of Done

- [ ] All Gherkin scenarios have passing step implementations
- [ ] Unit tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated if user-facing
```

## Validate a Story File

```bash
validate_story() {
  local file="$1"

  if [ ! -f "$file" ]; then
    echo "[spec-kit] MISSING  $file"
    return 1
  fi

  if ! grep -qE "^\s*(Scenario|Scenario Outline):" "$file"; then
    echo "[spec-kit] FAIL  $file — no Gherkin scenarios found"
    return 1
  fi

  STATUS=$(grep -m1 "^status:" "$file" | awk '{print $2}' | tr -d '"')
  if [ "$STATUS" != "approved" ]; then
    echo "[spec-kit] BLOCKED  $file  status=$STATUS  required=approved"
    return 1
  fi

  echo "[spec-kit] OK  $file  status=approved"
}

# Example
validate_story "docs/stories/user-auth-reset-password.md" || exit 1
```

## Check All Stories Are Approved

```bash
FAIL=0
STORIES=$(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null || true)

if [ -z "$STORIES" ]; then
  echo "[spec-kit] SKIP  no story files found"
  exit 0
fi

for story in $STORIES; do
  validate_story "$story" || FAIL=1
done

exit $FAIL
```

## Scenario Count Guidelines

| Feature complexity | Minimum scenarios |
|---|---|
| Simple CRUD | 2 (happy path + not-found/validation failure) |
| Auth / permissions | 3 (success + unauthorized + invalid input) |
| Multi-step workflow | 1 per step + 1 end-to-end |
| UI component | 2 (renders correctly + interaction) |

Do not invent scenarios not implied by the feature description. Mark unknown cases as `TODO` and flag for human review.

## Skip Conditions

```bash
BRANCH=$(git branch --show-current)
if echo "$BRANCH" | grep -qE '^(spike|chore)/'; then
  echo "[spec-kit] SKIP  branch=$BRANCH — spike/chore exempt"
  exit 0
fi
```

## Failure Modes

| Condition | Action |
|---|---|
| No feature description given | Ask the human before writing anything |
| Story has zero `Scenario:` blocks | Refuse to emit. Require at least one scenario. |
| Story `status: rejected` | Surface rejection reason. Do not advance. |
| Duplicate story file exists | Confirm with human before overwriting. |

## Logging

```
[spec-kit] WRITE   docs/stories/user-auth-reset-password.md
[spec-kit] HALT    awaiting human approval
[spec-kit] OK      docs/stories/user-auth-reset-password.md  status=approved
[spec-kit] SKIP    spike/* branch — spec-kit not required
```
