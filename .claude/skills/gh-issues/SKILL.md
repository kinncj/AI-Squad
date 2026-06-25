---
name: gh-issues
description: "Create, read, update, and link GitHub Issues as story artifacts throughout the story lifecycle. Use when managing issues for stories."
---

# SKILL: gh-issues

## Purpose

Create, read, update, and link GitHub Issues as first-class story artifacts. Agents use this skill to manage issue lifecycle — from PO story creation through QA closure — without human intervention.

## Inputs

| Field | Source | Example |
|---|---|---|
| `title` | story frontmatter | `"User can reset password"` |
| `body_file` | story file path | `docs/stories/auth-reset-0001.md` |
| `labels` | story frontmatter + phase | `"type:feature,priority:high,phase:discover"` |
| `milestone` | `project.config.yaml` | `"v1.0"` |
| `assignee` | orchestrator context | `"@me"` |
| `issue_number` | previously created issue | `42` |

## Outputs

| Field | Description |
|---|---|
| `issue_number` | integer, use for all subsequent operations |
| `issue_url` | full GitHub URL for logging |
| `issue_node_id` | GraphQL node ID, needed for Projects v2 card add |

## Create an Issue

```bash
# Capture issue number and URL
RESULT=$(gh issue create \
  --title "{title}" \
  --body-file "{body_file}" \
  --label "{labels}" \
  --milestone "{milestone}" \
  --json number,url,id)

ISSUE_NUMBER=$(echo "$RESULT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['number'])")
ISSUE_URL=$(echo "$RESULT"    | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['url'])")
ISSUE_NODE=$(echo "$RESULT"   | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['id'])")
```

## View an Issue

```bash
gh issue view {issue_number} --json number,title,state,labels,body
```

## Edit Labels and Assignee

```bash
# Add label, remove old phase label
gh issue edit {issue_number} \
  --add-label "phase:architect" \
  --remove-label "phase:discover"

# Assign to current actor
gh issue edit {issue_number} --add-assignee "@me"
```

## Add a Comment

```bash
gh issue comment {issue_number} \
  --body "{message}"
```

Comment conventions by role:

| Event | Template |
|---|---|
| Phase start | `"Phase {N} {PHASE_NAME}: starting. Artifact target: {path}"` |
| Phase complete | `"Phase {N} {PHASE_NAME}: complete. Artifacts: {path}"` |
| TDD red | `"RED: Failing test at {test_path} — {failure_message}"` |
| TDD green | `"GREEN: {test_count} tests passing."` |
| Blocked | `"BLOCKED: {agent} failed 3x on {task}. Human intervention required."` |

## Close an Issue

```bash
# Close with final validation summary
gh issue close {issue_number} \
  --comment "All acceptance criteria passing. Validation: {summary}"
```

## List Issues

```bash
# All open issues for current phase
gh issue list --label "phase:implement" --state open --json number,title,labels

# Blocked issues
gh issue list --label "blocked" --state open
```

## Link Issues (parent / blocks)

GitHub Issues has no native parent field — use body references and labels:

```bash
# In the child issue body, add:
# "Closes #{parent_number}" → auto-closes parent when child closes
# "Part of #{epic_issue_number}"

# For blocking relationships, comment on the blocked issue:
gh issue comment {blocked_number} \
  --body "Blocked by #{blocker_number} — {reason}"
gh issue edit {blocked_number} --add-label "blocked"
```

## Failure Modes

| Condition | Action |
|---|---|
| `gh` not authenticated | Stop. Output: `gh auth login required`. Do not retry. |
| Issue already exists with same title | Check with `gh issue list --search "{title}" --state all`. Skip create if found; return existing number. |
| Label does not exist | Create label first via `gh label create`. See `gh-labels-milestones` skill. |
| Milestone not found | Create milestone first: `gh api repos/{owner}/{repo}/milestones -f title="{name}"`. |
| Rate limit hit | Wait 60 seconds, retry once. If second attempt fails, stop and log. |

## Logging

Always log after every `gh issue` mutation:

```
[gh-issues] CREATE  #42  "User can reset password"  labels=[type:feature,priority:high]
[gh-issues] COMMENT #42  phase:implement start
[gh-issues] CLOSE   #42  all criteria passing
```

Format: `[gh-issues] {ACTION} #{number} {description}`
