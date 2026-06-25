---
name: gh-projects
description: "Manage GitHub Projects v2 board: add issues, update field values, and query board state. Use when managing project boards."
---

# SKILL: gh-projects

## Purpose

Manage GitHub Projects v2 board operations: add issues as cards, update field values (status, type, ADR required), and query board state. Uses `gh project` CLI where available; falls back to `gh api graphql` for operations the CLI does not expose.

## Inputs

| Field | Source | Example |
|---|---|---|
| `project_number` | `project.config.yaml` | `3` |
| `project_node_id` | `project.config.yaml` | `"PVT_kwDO..."` |
| `issue_node_id` | `gh-issues` skill output | `"I_kwDO..."` |
| `issue_number` | `gh-issues` skill output | `42` |
| `field_name` | project field name | `"Status"` |
| `field_value` | target option name | `"In Progress"` |

## Read project config

```bash
PROJECT_NUMBER=$(python3 -c "
import sys
for line in open('project.config.yaml'):
    if 'project_number:' in line:
        print(line.split(':')[1].strip())
        break
")

PROJECT_NODE=$(python3 -c "
import sys
for line in open('project.config.yaml'):
    if 'project_node_id:' in line:
        print(line.split(':', 1)[1].strip())
        break
")
```

## Add an Issue to the Project Board

```bash
# CLI (preferred)
gh project item-add "$PROJECT_NUMBER" \
  --owner "{owner}" \
  --url "https://github.com/{owner}/{repo}/issues/{issue_number}"

# GraphQL fallback (when CLI unavailable or for automation)
gh api graphql -f query='
  mutation($project: ID!, $content: ID!) {
    addProjectV2ItemById(input: {projectId: $project, contentId: $content}) {
      item { id }
    }
  }' \
  -f project="$PROJECT_NODE" \
  -f content="{issue_node_id}" \
  --jq '.data.addProjectV2ItemById.item.id'
```

Save the returned `item_id` — required for field updates.

## Get Field and Option IDs

Field updates require the field's node ID and the option's node ID (for single-select fields).

```bash
FIELDS=$(gh api graphql -f query='
  query($project: ID!) {
    node(id: $project) {
      ... on ProjectV2 {
        fields(first: 20) {
          nodes {
            ... on ProjectV2Field { id name }
            ... on ProjectV2SingleSelectField {
              id name
              options { id name }
            }
          }
        }
      }
    }
  }' \
  -f project="$PROJECT_NODE")

# Extract field ID by name
FIELD_ID=$(echo "$FIELDS" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for f in d['data']['node']['fields']['nodes']:
    if f.get('name') == '{field_name}':
        print(f['id'])
        break
")

# Extract option ID by value (single-select fields)
OPTION_ID=$(echo "$FIELDS" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for f in d['data']['node']['fields']['nodes']:
    if f.get('name') == '{field_name}':
        for opt in f.get('options', []):
            if opt['name'] == '{field_value}':
                print(opt['id'])
                break
")
```

## Update a Single-Select Field (Status, Type, ADR Required)

```bash
gh api graphql -f query='
  mutation($project: ID!, $item: ID!, $field: ID!, $option: String!) {
    updateProjectV2ItemFieldValue(input: {
      projectId: $project
      itemId: $item
      fieldId: $field
      value: { singleSelectOptionId: $option }
    }) {
      projectV2Item { id }
    }
  }' \
  -f project="$PROJECT_NODE" \
  -f item="{item_id}" \
  -f field="$FIELD_ID" \
  -f option="$OPTION_ID"
```

## Update a Text Field (Epic, Specialist)

```bash
gh api graphql -f query='
  mutation($project: ID!, $item: ID!, $field: ID!, $value: String!) {
    updateProjectV2ItemFieldValue(input: {
      projectId: $project
      itemId: $item
      fieldId: $field
      value: { text: $value }
    }) {
      projectV2Item { id }
    }
  }' \
  -f project="$PROJECT_NODE" \
  -f item="{item_id}" \
  -f field="$FIELD_ID" \
  -f value="{text_value}"
```

## Standard Field Updates by Pipeline Phase

| Phase | Field | Value |
|---|---|---|
| Discover | Status | `Todo` |
| Architect | Status | `In Progress` |
| Implement | Status | `In Progress` |
| Validate | Status | `In Review` |
| Done | Status | `Done` |

## Query Board State

```bash
# List all items with status
gh project item-list "$PROJECT_NUMBER" \
  --owner "{owner}" \
  --format json \
  --jq '.items[] | {id, title: .content.title, status: .fieldValues[] | select(.field.name=="Status") | .name}'
```

## Failure Modes

| Condition | Action |
|---|---|
| `project_node_id` missing from config | Run `maple project` to bootstrap. Stop until resolved. |
| Item already on board | Query before adding. `gh project item-list` and check content URL. Skip if present. |
| Field ID not found | Re-fetch fields. Field names are case-sensitive. |
| Option ID not found | List available options from field metadata. Do not guess. |
| GraphQL rate limit | Wait 60 seconds. Retry once. Log and stop on second failure. |

## Logging

```
[gh-projects] ADD    #42  → project #{project_number}  item_id={id}
[gh-projects] UPDATE #42  field=Status  value="In Progress"
[gh-projects] SKIP   #42  already on board
```
