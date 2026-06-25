---
name: gherkin-authoring
description: "Produce valid, consistently named Gherkin story files with correct IDs, timestamps, and tags. Use when writing or updating story files."
---

# SKILL: gherkin-authoring

## Purpose

Produce valid, consistently named Gherkin story files. Handles ID allocation, timestamp generation, filename construction, tag formatting, and scenario validation. Used by the product-owner agent whenever a new story is written or updated.

## Story File Naming Convention

```
<epic-slug>-<story-slug>-<timestamp>-<NNNN>.md
```

| Part | Format | Example |
|---|---|---|
| `epic-slug` | kebab-case, 2–4 words | `user-auth` |
| `story-slug` | kebab-case, 2–5 words | `reset-password` |
| `timestamp` | `YYYYMMDDHHMMSS` | `20250416143000` |
| `NNNN` | 4-digit zero-padded, sequential within epic | `0003` |

Full example: `user-auth-reset-password-20250416143000-0003.md`

## Allocate the Next Story ID

```bash
# Count existing stories for an epic to get next sequential ID
EPIC="user-auth"
NEXT_ID=$(ls docs/stories/${EPIC}-* 2>/dev/null \
  | grep -oE '\-[0-9]{4}\.md$' \
  | grep -oE '[0-9]{4}' \
  | sort -n | tail -1 \
  | python3 -c "import sys; n=sys.stdin.read().strip(); print(f'{int(n)+1:04d}' if n else '0001')")
[ -z "$NEXT_ID" ] && NEXT_ID="0001"
echo "$NEXT_ID"
```

## Generate Filename

```bash
EPIC="user-auth"
STORY="reset-password"
TS=$(date -u +"%Y%m%d%H%M%S")
ID="$NEXT_ID"   # from allocation step above
FILENAME="${EPIC}-${STORY}-${TS}-${ID}.md"
echo "$FILENAME"   # user-auth-reset-password-20250416143000-0001.md
```

## Story File Template

Every story file must contain:

```markdown
---
id: "{EPIC}-{NNNN}"
title: "{Human-readable story title}"
epic: "{epic-slug}"
priority: "high|medium|low|critical"
ui: false
adr_required: false
milestone: "{milestone name}"
labels:
  - "type:feature"
  - "priority:high"
  - "phase:discover"
issue_number: null
issue_url: null
---

## Story

**As a** {actor},
**I want** {capability},
**so that** {business outcome}.

## Acceptance Criteria

```gherkin
@story:{EPIC}-{NNNN} @epic:{epic-slug} @priority:{level}
Feature: {Feature title}

  Scenario: {scenario title}
    Given {precondition}
    When {action}
    Then {expected outcome}
```

## Definition of Done

- [ ] All Gherkin scenarios have passing step implementations
- [ ] Unit tests written and passing
- [ ] Integration tests written and passing (if applicable)
- [ ] Code reviewed and approved
- [ ] No regressions in related features
- [ ] Documentation updated (if applicable)

## ADR Links

<!-- Add links to relevant ADRs here -->
```

## Gherkin Validation Rules

Before writing a scenario, verify:

1. **Single responsibility**: each `Scenario` tests exactly one behavior.
2. **Given-When-Then order**: never skip or reorder.
3. **No implementation detail in steps**: steps describe intent, not code.
4. **Declarative steps**: `Given the user is logged in` ✓ — `Given I call POST /auth/login` ✗
5. **Tags present**: `@story:`, `@epic:`, `@priority:` required on every Feature block.
6. **No `And` as first step**: `And` is only valid after a `Given`, `When`, or `Then`.

## Background Block (shared preconditions)

```gherkin
Feature: Password Reset

  Background:
    Given the user has a verified account
    And the account is not locked

  Scenario: Successful reset request
    When the user submits their email address
    Then a reset link is sent to that address
```

Use `Background` when 2+ scenarios share the same preconditions.

## Scenario Outline (data-driven)

```gherkin
  Scenario Outline: Reject invalid email formats
    When the user submits "<email>"
    Then they see a validation error "<message>"

    Examples:
      | email        | message                  |
      | notanemail   | Invalid email format     |
      | @nodomain    | Invalid email format     |
      | a@b          | Domain must have TLD     |
```

## Tags Reference

| Tag | Required | Description |
|---|---|---|
| `@story:{id}` | Yes | Links scenario to story file |
| `@epic:{slug}` | Yes | Groups stories by epic |
| `@priority:{level}` | Yes | `critical`, `high`, `medium`, `low` |
| `@wip` | No | Work in progress — excluded from CI by default |
| `@ui` | No | Requires browser/UI driver |
| `@integration` | No | Requires external services |
| `@smoke` | No | Runs in production smoke suite |

## Failure Modes

| Condition | Action |
|---|---|
| Story ID collision (same NNNN exists) | Re-run allocation step. Never overwrite an existing file. |
| `docs/stories/` does not exist | Create it: `mkdir -p docs/stories/` |
| Scenario has no When | Invalid — split into two scenarios or add action step |
| Missing required tags | Add tags before writing the file |
| Title exceeds 72 characters | Shorten. Titles appear in issue headings and PR titles. |

## Logging

```
[gherkin] ALLOCATE  epic=user-auth  next_id=0003
[gherkin] CREATE    docs/stories/user-auth-reset-password-20250416143000-0003.md
[gherkin] VALIDATE  scenarios=3  tags=ok  structure=ok
```
