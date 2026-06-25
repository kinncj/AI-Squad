---
name: gh-labels-milestones
description: "Bootstrap and sync GitHub labels and milestones idempotently. Use when setting up a new repository or syncing label definitions."
---

# SKILL: gh-labels-milestones

## Purpose

Bootstrap and sync GitHub labels and milestones idempotently. Labels and milestones are created if absent; existing ones are updated if color or description differs. Never delete — only add or update.

## Label Bootstrap

Use this at project start (`maple labels`) and in CI to guarantee label state.

```bash
# Idempotent label create-or-update
upsert_label() {
  local name="$1" color="$2" description="$3"
  if gh label list --json name --jq '.[].name' | grep -qx "$name"; then
    gh label edit "$name" --color "$color" --description "$description"
  else
    gh label create "$name" --color "$color" --description "$description"
  fi
}
```

## Label Groups

### Phase Labels (pipeline position)

```bash
upsert_label "phase:discover"   "0075ca" "Phase 1: Discovery"
upsert_label "phase:architect"  "0075ca" "Phase 2: Architecture"
upsert_label "phase:plan"       "0075ca" "Phase 3: Planning"
upsert_label "phase:infra"      "0075ca" "Phase 4: Infrastructure"
upsert_label "phase:implement"  "0075ca" "Phase 5: Implementation"
upsert_label "phase:validate"   "0075ca" "Phase 6: Validation"
upsert_label "phase:document"   "0075ca" "Phase 7: Documentation"
upsert_label "phase:done"       "0075ca" "Phase 8: Complete"
```

### Type Labels (work classification)

```bash
upsert_label "type:feature"     "0052cc" "New capability"
upsert_label "type:bug"         "d73a4a" "Defect"
upsert_label "type:spike"       "e4e669" "Research / time-boxed exploration"
upsert_label "type:chore"       "ededed" "Non-functional maintenance"
upsert_label "type:refactor"    "ededed" "Code restructuring, no behavior change"
upsert_label "type:docs"        "0075ca" "Documentation only"
```

### Priority Labels (MoSCoW)

```bash
upsert_label "priority:critical" "b60205" "Must ship — blocks launch"
upsert_label "priority:high"     "e11d48" "Must have"
upsert_label "priority:medium"   "f97316" "Should have"
upsert_label "priority:low"      "84cc16" "Could have"
upsert_label "priority:wontfix"  "ffffff" "Won't have this cycle"
```

### Spec / Story Kit Labels

```bash
upsert_label "spec:problem"     "7057ff" "Problem statement written"
upsert_label "spec:approved"    "7057ff" "Spec approved by PO"
upsert_label "spec:in-review"   "7057ff" "Spec under review"
```

### Design Labels

```bash
upsert_label "design:pending"      "fbca04" "Awaiting design"
upsert_label "design:in-progress"  "fbca04" "Design work active"
upsert_label "design:approved"     "fbca04" "Design approved"
upsert_label "design:a11y-passed"  "fbca04" "Accessibility review passed"
```

### ADR Labels

```bash
upsert_label "adr:required"    "5319e7" "ADR must be written before implementation"
upsert_label "adr:in-progress" "5319e7" "ADR being authored"
upsert_label "adr:complete"    "5319e7" "ADR accepted"
upsert_label "adr:rejected"    "5319e7" "ADR rejected — see comments"
```

### UI / Accessibility Labels

```bash
upsert_label "ui:required"     "fef2c0" "Has UI surface — design review required"
upsert_label "ui:in-progress"  "fef2c0" "UI implementation active"
upsert_label "ui:complete"     "fef2c0" "UI complete and reviewed"
```

### Status Labels

```bash
upsert_label "in-progress" "0052cc" "Work started"
upsert_label "blocked"     "b60205" "Blocked — needs human"
upsert_label "validated"   "0e8a16" "All tests pass"
upsert_label "tdd:red"     "d73a4a" "Failing test written"
upsert_label "tdd:green"   "0e8a16" "Tests passing"
```

## Milestone Bootstrap

```bash
# Idempotent milestone create
upsert_milestone() {
  local title="$1" due="$2" description="$3"
  EXISTING=$(gh api "repos/{owner}/{repo}/milestones" \
    --jq ".[] | select(.title == \"$title\") | .number" 2>/dev/null)
  if [ -n "$EXISTING" ]; then
    gh api "repos/{owner}/{repo}/milestones/$EXISTING" \
      -X PATCH \
      -f title="$title" \
      -f due_on="${due}T00:00:00Z" \
      -f description="$description"
  else
    gh api "repos/{owner}/{repo}/milestones" \
      -X POST \
      -f title="$title" \
      -f due_on="${due}T00:00:00Z" \
      -f description="$description"
  fi
}

# Example usage
upsert_milestone "v1.0" "2025-12-31" "Initial release"
```

## Assign Labels to an Issue

```bash
# Add labels (idempotent — gh ignores if already present)
gh issue edit {issue_number} \
  --add-label "type:feature,priority:high,phase:discover"

# Remove a specific label
gh issue edit {issue_number} --remove-label "phase:discover"
```

## Failure Modes

| Condition | Action |
|---|---|
| Label name collision (wrong color) | `gh label edit` — update in place |
| Milestone already exists | Patch via API — do not create duplicate |
| `gh` not authenticated | Stop immediately. Do not retry. |
| Label with special characters | URL-encode the label name in API calls |

## Logging

```
[gh-labels] UPSERT  label="type:feature"    color=0052cc
[gh-labels] SKIP    label="priority:high"   (unchanged)
[gh-milestones] CREATE  "v1.0"  due=2025-12-31
[gh-milestones] SKIP    "v1.0"  already exists
```
