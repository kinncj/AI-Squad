---
name: a11y-audit
description: "Run WCAG 2.2 Level AA accessibility audits against generated UI. Use when a story has ui:true or when asked to audit accessibility."
---

# SKILL: a11y-audit

## Purpose

Run WCAG 2.2 Level AA accessibility audits against generated UI. Uses axe-core (via `@axe-core/cli`) or pa11y as available. Posts findings as PR comments. Blocks merge on AA violations. Required for every story with `ui: true`.

## Target awareness

When `design.target` (`project.config.yaml`, default `web`) is `tui`, skip the browser tooling (axe/pa11y)
and the preview URL. Instead evaluate the terminal a11y checklist from `docs/design/design-targets.md`
and write the findings to `docs/design/mockups/<id>.a11y.json` using the SAME object shape the web path
produces — an object with `violations[]`, where each violation has `id`, `impact`
(`critical|serious|moderate|minor`), `description`, `help`, and `nodes[]` (`{target:[...], failureSummary}`).
The gate (`scripts/sdlc/a11y-gate.sh`) reads only `violations[].impact`, so matching this shape keeps it
unchanged. Map checklist failures to impacts: keyboard-reachable→critical, focus-visible→serious,
color-contrast→serious, color-only-signaling→serious, no-color-support→moderate, min-width-resize→moderate.
The WCAG criteria, tool-detection, axe/pa11y, and PR-comment sections below apply to `web` targets.

## WCAG 2.2 AA — Minimum Requirements

| Criterion | Level | What to check |
|---|---|---|
| 1.1.1 Non-text Content | A | All `<img>` have `alt`. Icons used as UI controls have labels. |
| 1.3.1 Info and Relationships | A | Semantic HTML: headings, lists, tables, landmarks used correctly. |
| 1.4.3 Contrast (minimum) | AA | Normal text ≥ 4.5:1. Large text ≥ 3:1. |
| 1.4.11 Non-text Contrast | AA | UI components and focus indicators ≥ 3:1 against background. |
| 2.1.1 Keyboard | A | All interactive elements reachable and operable via keyboard. |
| 2.4.3 Focus Order | A | Tab order is logical and follows visual layout. |
| 2.4.7 Focus Visible | AA | Focus indicator visible on all interactive elements. |
| 3.3.1 Error Identification | A | Errors identified in text, not color alone. |
| 3.3.2 Labels or Instructions | A | All inputs have labels. |
| 4.1.2 Name, Role, Value | A | All UI components have accessible name, role, and state. |

## Tool Detection

```bash
detect_tool() {
  if command -v axe &>/dev/null; then
    echo "axe"
  elif command -v pa11y &>/dev/null; then
    echo "pa11y"
  elif npx --yes @axe-core/cli --version &>/dev/null 2>&1; then
    echo "axe-npx"
  else
    echo "none"
  fi
}
TOOL=$(detect_tool)
```

## Run with axe-core CLI

```bash
URL="${1:-http://localhost:3000}"  # preview URL or Storybook story URL
STORY_ID="${2:-unknown}"
REPORT="docs/design/mockups/${STORY_ID}.a11y.json"

mkdir -p docs/design/mockups

axe "$URL" \
  --stdout \
  --tags wcag2a,wcag2aa,wcag21aa,wcag22aa \
  > "$REPORT"

VIOLATIONS=$(python3 -c "import json,sys; d=json.load(open('$REPORT')); print(len(d.get('violations',[])))")
echo "[a11y-audit] axe  $URL  violations=$VIOLATIONS"
```

## Run with pa11y

```bash
URL="${1:-http://localhost:3000}"
STORY_ID="${2:-unknown}"
REPORT="docs/design/mockups/${STORY_ID}.a11y.json"

pa11y "$URL" \
  --standard WCAG2AA \
  --reporter json \
  > "$REPORT" 2>&1

ISSUES=$(python3 -c "import json,sys; d=json.load(open('$REPORT')); print(len(d) if isinstance(d,list) else 0)")
echo "[a11y-audit] pa11y  $URL  issues=$ISSUES"
```

## Parse Results and Classify

```python
import json, sys

report_path = sys.argv[1]
data = json.load(open(report_path))

# axe format
violations = data.get("violations", [])
passes     = data.get("passes", [])

critical = [v for v in violations if v["impact"] in ("critical", "serious")]
moderate = [v for v in violations if v["impact"] == "moderate"]
minor    = [v for v in violations if v["impact"] == "minor"]

print(f"CRITICAL/SERIOUS: {len(critical)}")
print(f"MODERATE:         {len(moderate)}")
print(f"MINOR:            {len(minor)}")
print(f"PASSES:           {len(passes)}")

# Merge gate: block on critical + serious
if critical:
    print("\nMERGE BLOCKED — resolve the following before merging:")
    for v in critical:
        print(f"  [{v['impact'].upper()}] {v['id']}: {v['description']}")
        for node in v.get("nodes", [])[:2]:
            print(f"    → {node.get('target', ['?'])[0]}: {node.get('failureSummary','')[:120]}")
    sys.exit(1)
```

## Post Findings as PR Comment

```bash
STORY_ID="auth-reset-0001"
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" --json number --jq '.[0].number')
REPORT="docs/design/mockups/${STORY_ID}.a11y.json"

# Build comment body
COMMENT=$(python3 - <<'EOF'
import json, sys

data = json.load(open(sys.argv[1]))
violations = data.get("violations", [])

if not violations:
    print("## ✅ A11y Audit Passed\n\nNo WCAG 2.2 AA violations found.")
    sys.exit(0)

lines = [f"## ⚠️ A11y Audit: {len(violations)} violation(s) found\n"]
for v in violations[:10]:
    impact = v.get("impact","").upper()
    lines.append(f"### [{impact}] {v['id']}")
    lines.append(f"_{v['description']}_\n")
    for node in v.get("nodes",[])[:2]:
        selector = ', '.join(node.get("target",[]))
        lines.append(f"- `{selector}`")
        lines.append(f"  {node.get('failureSummary','')[:200]}")
    lines.append("")

if len(violations) > 10:
    lines.append(f"_...and {len(violations)-10} more. See full report in `docs/design/mockups/`._")

print('\n'.join(lines))
EOF
"$REPORT")

gh pr comment "$PR_NUMBER" --body "$COMMENT"
echo "[a11y-audit] PR_COMMENT  #$PR_NUMBER  violations=$(echo "$COMMENT" | grep -c '\[CRITICAL\]\|\[SERIOUS\]' || echo 0)"
```

## Merge Gate Check

Called from `scripts/sdlc/a11y-gate.sh` (added in Phase VII):

```bash
STORY_FILE="$1"
UI=$(python3 -c "
import re
m = re.search(r'^ui:\s*(true|false)', open('$STORY_FILE').read(), re.MULTILINE)
print(m.group(1) if m else 'false')
")

if [ "$UI" != "true" ]; then
  echo "[a11y-audit] SKIP  ui:false story — no audit required"
  exit 0
fi

STORY_ID=$(python3 -c "
import re
m = re.search(r'^id:\s*[\"\'](.*?)[\"\']', open('$STORY_FILE').read(), re.MULTILINE)
print(m.group(1) if m else 'unknown')
")
REPORT="docs/design/mockups/${STORY_ID}.a11y.json"

if [ ! -f "$REPORT" ]; then
  echo "[a11y-audit] FAIL  no audit report found for $STORY_ID — run audit before merging"
  exit 1
fi

python3 scripts/sdlc/parse-a11y-report.py "$REPORT"
```

## Manual Audit Checklist (no tool available)

When `TOOL=none`, perform a structured manual check:

```markdown
## Manual A11y Checklist — {story_id}

### Perceivable
- [ ] All images have meaningful `alt` text (or `alt=""` for decorative)
- [ ] Color is not the sole means of conveying information
- [ ] Normal text contrast ≥ 4.5:1 (check with browser DevTools)
- [ ] Large text contrast ≥ 3:1

### Operable
- [ ] Tab through the entire form/component with keyboard only
- [ ] All actions reachable without mouse
- [ ] No keyboard trap
- [ ] Focus indicator visible on every interactive element

### Understandable
- [ ] Form inputs have visible labels (not just placeholders)
- [ ] Errors described in text, not color alone
- [ ] Error messages are specific and actionable

### Robust
- [ ] Correct semantic elements: `<button>`, `<input>`, `<label>`
- [ ] ARIA attributes only where native HTML is insufficient
- [ ] Works with browser zoom at 200%
```

## Failure Modes

| Condition | Action |
|---|---|
| No tool available | Fall back to manual checklist. Log `TOOL=none — manual audit required`. |
| Preview URL not reachable | Log `URL_UNREACHABLE`. Do not skip audit — escalate. |
| `ui: false` story | Skip audit. Log `SKIP — ui:false`. |
| Critical violations found | Post PR comment. Exit non-zero. Block merge. |
| Report already exists and is current | Skip re-run. Log `SKIP — current report exists`. |

## Logging

```
[a11y-audit] RUN      http://localhost:6006/... story=auth-reset-0001  tool=axe
[a11y-audit] RESULT   violations=0  passes=24  WCAG2AA=PASS
[a11y-audit] RESULT   violations=3  critical=1  WCAG2AA=FAIL
[a11y-audit] PR_COMMENT  #42  violations=1
[a11y-audit] SKIP     ui:false story — audit not required
[a11y-audit] BLOCKED  merge blocked — 1 critical violation unresolved
```
