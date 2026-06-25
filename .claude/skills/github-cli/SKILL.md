---
name: github-cli
description: "Comprehensive gh CLI reference — issues, tasks, PRs, repos, branches, Actions, search, and project board. Use whenever interacting with GitHub."
---

# SKILL: GitHub CLI

## Repo Context (always run first)

```bash
# Who am I, what repo am I in
gh auth status
gh repo view --json name,owner,defaultBranchRef,url,isPrivate

# Open issues + PRs at a glance
gh issue list --state open --limit 20 --json number,title,labels,assignees
gh pr list --state open --json number,title,headRefName,statusCheckRollup
```

---

## Issues as Tasks

### Create
```bash
# Feature story
gh issue create \
  --title "feat: {title}" \
  --body-file docs/stories/{slug}/Story.md \
  --label "type:feature,priority:high,phase:discover" \
  --milestone "v1.0" \
  --assignee "@me"

# Bug
gh issue create \
  --title "fix: {title}" \
  --body "## Steps to reproduce\n{steps}\n\n## Expected\n{expected}\n\n## Actual\n{actual}" \
  --label "type:bug,priority:high"

# Task (sub-work, no story file)
gh issue create \
  --title "task: {title}" \
  --body "{description}" \
  --label "type:task" \
  --assignee "@me"
```

### Read
```bash
gh issue view {number}
gh issue view {number} --json number,title,state,body,labels,assignees,comments,milestone

# Search
gh issue list --search "label:blocked" --state open
gh issue list --search "assignee:@me is:open"
gh issue list --search "phase:implement" --label "tdd:red"
gh issue list --state open --label "type:feature" --milestone "v1.0"
```

### Update
```bash
# Phase transition
gh issue edit {number} \
  --add-label "phase:architect" \
  --remove-label "phase:discover"

# Assign / unassign
gh issue edit {number} --add-assignee "@me"
gh issue edit {number} --remove-assignee "{login}"

# Change milestone
gh issue edit {number} --milestone "v2.0"

# Block / unblock
gh issue edit {number} --add-label "blocked"
gh issue edit {number} --remove-label "blocked"
```

### Comment
```bash
gh issue comment {number} --body "{message}"

# Phase lifecycle comments (standard format)
gh issue comment {number} --body "Phase {N} {NAME}: starting. Target artifacts: {paths}"
gh issue comment {number} --body "Phase {N} {NAME}: complete. Artifacts: {paths}"
gh issue comment {number} --body "RED: Failing test at {path} — {message}"
gh issue comment {number} --body "GREEN: {count} tests passing."
gh issue comment {number} --body "BLOCKED: {agent} failed 3x on {task}. Human required."
```

### Close
```bash
gh issue close {number} --comment "Acceptance criteria passing. Validation: {summary}"
gh issue close {number} --reason "not planned"
```

---

## Pull Requests

```bash
# Create PR (always draft first)
gh pr create \
  --title "feat: {description}" \
  --body "## Summary\n{summary}\n\nCloses #{issue_number}" \
  --base main \
  --draft

# Mark ready when all gates pass
gh pr ready {number}

# View PR + CI status
gh pr view {number} --json number,title,state,reviews,statusCheckRollup,files

# Request review
gh pr edit {number} --add-reviewer "{login}"

# Check CI on PR
gh pr checks {number}

# Merge (squash — preferred)
gh pr merge {number} --squash --subject "feat: {description} (#{issue_number})" --delete-branch

# Review
gh pr review {number} --approve --body "LGTM"
gh pr review {number} --request-changes --body "{feedback}"
```

---

## Branches

```bash
# List branches
gh api repos/{owner}/{repo}/branches --jq '.[].name'

# Create via API (when not in git context)
gh api repos/{owner}/{repo}/git/refs \
  -f ref="refs/heads/feat/{slug}" \
  -f sha="$(gh api repos/{owner}/{repo}/git/ref/heads/main --jq '.object.sha')"

# Delete remote branch
gh api -X DELETE repos/{owner}/{repo}/git/refs/heads/feat/{slug}

# Branch protection status
gh api repos/{owner}/{repo}/branches/main/protection
```

---

## Actions / CI

```bash
# List recent runs
gh run list --limit 10
gh run list --workflow ci.yml --branch main --limit 5

# Watch a run live
gh run watch {run-id}

# View failed run details
gh run view {run-id} --log-failed

# Trigger workflow manually
gh workflow run {workflow-file} --ref main

# Re-run failed jobs only
gh run rerun {run-id} --failed

# Download artifact
gh run download {run-id} --name {artifact-name} --dir ./artifacts
```

---

## Repo Inspection

```bash
# Full repo metadata
gh repo view --json name,owner,description,topics,visibility,defaultBranchRef,\
licenseInfo,stargazerCount,forkCount,createdAt,updatedAt

# Collaborators
gh api repos/{owner}/{repo}/collaborators --jq '.[].login'

# Repo settings
gh api repos/{owner}/{repo} --jq '{private:.private,hasIssues:.has_issues,hasWiki:.has_wiki}'

# Topics
gh repo edit --add-topic "{topic}"

# Clone URL
gh repo view --json sshUrl,url

# Releases
gh release list --limit 5
gh release view {tag}
```

---

## Labels & Milestones

```bash
# Bootstrap MAPLE labels (idempotent)
bash scripts/bootstrap-labels.sh

# List labels
gh label list

# Create label
gh label create "type:task" --color "0075ca" --description "Work item"

# Create milestone
gh api repos/{owner}/{repo}/milestones \
  -f title="v1.0" \
  -f description="First release" \
  -f due_on="2026-06-01T00:00:00Z"

# List milestones
gh api repos/{owner}/{repo}/milestones --jq '.[] | {number:.number,title:.title,open:.open_issues}'
```

---

## Search

```bash
# Search code in repo
gh search code "{query}" --repo {owner}/{repo} --json path,url

# Search issues across GitHub
gh search issues "{query}" --repo {owner}/{repo} --json number,title,state

# Search PRs
gh search prs "{query}" --repo {owner}/{repo} --state open
```

---

## Issue Lifecycle (MAPLE Standard)

| Event | Label added | Label removed | Comment |
|---|---|---|---|
| Story created | `phase:discover` | — | "Phase 1 DISCOVER: story created." |
| Architect done | `phase:architect` | `phase:discover` | "Phase 2 ARCHITECT: design complete." |
| Plan done | `phase:plan` | `phase:architect` | "Phase 3 PLAN: plan.md ready." |
| Infra done | `phase:infra` | `phase:plan` | "Phase 4 INFRA: containers healthy." |
| Implement start | `phase:implement,in-progress` | `phase:infra` | "Phase 5 IMPLEMENT: TDD loop starting." |
| TDD red | `tdd:red` | — | "RED: failing test at {path}" |
| TDD green | `tdd:green` | `tdd:red` | "GREEN: {n} tests passing." |
| Validate pass | `phase:validate,validated` | `phase:implement` | "Phase 6 VALIDATE: 100% pass." |
| Document done | `phase:document` | `phase:validate` | "Phase 7 DOCUMENT: docs complete." |
| PR created | `ready-for-review` | `phase:document` | "Phase 8: PR #{pr} created." |
| Blocked | `blocked` | — | "BLOCKED: {agent} failed 3x. Human needed." |
| Unblocked | — | `blocked` | "Unblocked: {resolution}" |

---

## Failure Modes

| Condition | Action |
|---|---|
| Not authenticated | `gh auth login` — do not retry ops |
| Label missing | Create via `gh label create` before using |
| Milestone missing | Create via `gh api milestones` before using |
| Rate limited | Wait 60s, retry once; if fails again, stop |
| Issue already exists | `gh issue list --search "{title}" --state all` — return existing number |
| `gh` not found | Stop. Output: "gh CLI not available. Install from https://cli.github.com" |
