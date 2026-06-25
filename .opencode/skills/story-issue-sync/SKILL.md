---
name: story-issue-sync
description: "Maintain bidirectional consistency between story markdown files and GitHub Issues. Use when syncing story status with GitHub."
---

# SKILL: story-issue-sync

## Purpose

Maintain bidirectional consistency between story files (`docs/stories/*.md`) and GitHub Issues. The file is authoritative for narrative and Gherkin. The issue is authoritative for labels, status, project board position, and DoD checklist state.

## Ownership Model

| Attribute | Authoritative source | Sync direction |
|---|---|---|
| Title | Story file (frontmatter `title`) | File → Issue |
| Body / Gherkin | Story file | File → Issue (on update) |
| `issue_number` | Issue (assigned at create) | Issue → File |
| `issue_url` | Issue | Issue → File |
| Labels | Issue | Issue → File (`labels` frontmatter) |
| Status / Phase | Issue labels | Issue → File (`labels` frontmatter) |
| DoD checklist | Issue body (task list) | Issue → File (on review) |
| Project board position | Project v2 | Not persisted to file |

## Inputs / Outputs

| Field | Source | Notes |
|---|---|---|
| `story_file` | `docs/stories/*.md` | Path to the story markdown file |
| `issue_number` | Issue (post-create) | Written back to story frontmatter |
| `issue_node_id` | `gh issue create --json id` | Passed to `gh-projects` skill for board add |
| `issue_url` | Issue (post-create) | Written back to story frontmatter |

## Create: File → Issue

When a new story file exists with `issue_number: null`:

```bash
STORY_FILE="docs/stories/user-auth-reset-password-20250416143000-0001.md"

# Extract frontmatter fields
TITLE=$(python3 -c "
import sys, re
text = open('$STORY_FILE').read()
m = re.search(r'^title:\s*[\"\'](.*?)[\"\']', text, re.MULTILINE)
print(m.group(1) if m else '')
")

LABELS=$(python3 -c "
import sys, re
text = open('$STORY_FILE').read()
m = re.findall(r'^\s+- [\"\'](.*?)[\"\']\s*$', text, re.MULTILINE)
print(','.join(m))
")

MILESTONE=$(python3 -c "
import sys, re
text = open('$STORY_FILE').read()
m = re.search(r'^milestone:\s*[\"\'](.*?)[\"\']', text, re.MULTILINE)
print(m.group(1) if m else '')
")

# Create the issue
RESULT=$(gh issue create \
  --title "$TITLE" \
  --body-file "$STORY_FILE" \
  --label "$LABELS" \
  --milestone "$MILESTONE" \
  --json number,url,id)

ISSUE_NUMBER=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin)['number'])")
ISSUE_URL=$(echo "$RESULT"    | python3 -c "import sys,json; print(json.load(sys.stdin)['url'])")

# Write back to story frontmatter
python3 - <<EOF
import re
text = open('$STORY_FILE').read()
text = re.sub(r'^issue_number: null', f'issue_number: $ISSUE_NUMBER', text, flags=re.MULTILINE)
text = re.sub(r'^issue_url: null', f'issue_url: "$ISSUE_URL"', text, flags=re.MULTILINE)
open('$STORY_FILE', 'w').write(text)
EOF

echo "[story-sync] CREATED  #$ISSUE_NUMBER  ← $STORY_FILE"
```

## Update: File → Issue (body changed)

When story Gherkin or acceptance criteria are edited:

```bash
ISSUE_NUMBER=42
STORY_FILE="docs/stories/user-auth-reset-password-20250416143000-0001.md"

gh issue edit "$ISSUE_NUMBER" \
  --body-file "$STORY_FILE"

echo "[story-sync] BODY_UPDATE  #$ISSUE_NUMBER  ← $STORY_FILE"
```

## Sync: Issue → File (labels changed)

When labels on the issue change (e.g., after PO updates priority or phase advances):

```bash
ISSUE_NUMBER=42
STORY_FILE="docs/stories/user-auth-reset-password-20250416143000-0001.md"

# Fetch current labels from issue
CURRENT_LABELS=$(gh issue view "$ISSUE_NUMBER" \
  --json labels \
  --jq '[.labels[].name] | join(",")')

# Rewrite labels block in frontmatter
python3 - <<EOF
import re
text = open('$STORY_FILE').read()
label_lines = '\n'.join(f'  - "{l}"' for l in "$CURRENT_LABELS".split(',') if l)
text = re.sub(
    r'^labels:\n(  - .*\n)+',
    f'labels:\n{label_lines}\n',
    text,
    flags=re.MULTILINE
)
open('$STORY_FILE', 'w').write(text)
EOF

echo "[story-sync] LABELS_SYNC  #$ISSUE_NUMBER  → $STORY_FILE"
```

## Detect Drift (file vs issue)

Run before any update to check whether file and issue are in sync:

```bash
ISSUE_NUMBER=42
STORY_FILE="docs/stories/user-auth-reset-password-20250416143000-0001.md"

FILE_TITLE=$(python3 -c "
import re
m = re.search(r'^title:\s*[\"\'](.*?)[\"\']', open('$STORY_FILE').read(), re.MULTILINE)
print(m.group(1) if m else '')
")

ISSUE_TITLE=$(gh issue view "$ISSUE_NUMBER" --json title --jq '.title')

if [ "$FILE_TITLE" != "$ISSUE_TITLE" ]; then
  echo "[story-sync] DRIFT  title mismatch: file='$FILE_TITLE'  issue='$ISSUE_TITLE'"
  # File is authoritative for title — update the issue
  gh issue edit "$ISSUE_NUMBER" --title "$FILE_TITLE"
fi
```

## Batch Sync (all stories)

```bash
find docs/stories -name "*.md" ! -name "_template.md" | while read -r f; do
  NUMBER=$(python3 -c "
import re
m = re.search(r'^issue_number:\s*(\d+)', open('$f').read(), re.MULTILINE)
print(m.group(1) if m else '')
")
  if [ -z "$NUMBER" ]; then
    echo "[story-sync] NEEDS_CREATE  $f"
  else
    echo "[story-sync] HAS_ISSUE    $f  → #$NUMBER"
  fi
done
```

## Failure Modes

| Condition | Action |
|---|---|
| Story has `issue_number: null` | Create the issue. Never skip. |
| Issue number in file but issue is closed | Log warning. Do not reopen automatically — escalate to human. |
| Title drift detected | File wins. Update issue title. |
| Labels on issue unknown to label set | Log: `[story-sync] UNKNOWN_LABEL {name}`. Do not remove. |
| Story file missing after issue exists | Log warning. Issue is not deleted. Human decides. |

## Logging

```
[story-sync] CREATED       #42  ← docs/stories/user-auth-reset-0001.md
[story-sync] BODY_UPDATE   #42  ← docs/stories/user-auth-reset-0001.md
[story-sync] LABELS_SYNC   #42  → docs/stories/user-auth-reset-0001.md
[story-sync] DRIFT         #42  title mismatch — issue updated
[story-sync] SKIP          #42  no changes detected
```
