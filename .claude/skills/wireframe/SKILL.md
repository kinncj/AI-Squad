---
name: wireframe
description: "Generate low-fidelity wireframes (ASCII, SVG, or HTML) from user story files. Use when creating wireframes for UI stories."
---

# SKILL: wireframe

## Purpose

Generate low-fidelity wireframes from user story files. Output is deterministic — given the same story and layout hints, the same wireframe structure is produced. Three output formats: ASCII (default), SVG, HTML. Human approval is required before the wireframe feeds downstream mockup work.

## Inputs

| Field | Source | Example |
|---|---|---|
| `story_file` | path to story markdown | `docs/stories/auth-reset-0001.md` |
| `ui_components` | derived from story Gherkin | form, button, error message |
| `stack` | `project.config.yaml` | `react-mantine` |

## Outputs

The required files depend on `design.target` in `project.config.yaml` (default `web`) — see Target awareness:

| File | Location | Targets |
|---|---|---|
| `<story-id>.wireframe.md` | `docs/design/wireframes/` | web + tui — ASCII layout + approval metadata |
| `<story-id>.wireframe.excalidraw` | `docs/design/wireframes/` | web + tui — editable Excalidraw diagram |
| `<story-id>.wireframe.html` | `docs/design/wireframes/` | web only — browser-previewable static wireframe |

A wireframe stage that produces fewer than the files required for the active target is **incomplete**. Do not PAUSE or mark DONE without them.

## Target awareness

`design.target` decides which files are required:

- **web** — `<id>.wireframe.md` (ASCII) + `.html` (preview) + `.excalidraw`.
- **tui** — `<id>.wireframe.md` (ASCII/box-drawing) + `.excalidraw`. No `.html`.

Use box-drawing primitives for tui layouts, label panes/overlays/status bar, and include the
keybinding legend and focus order inline in the `.md`. The Excalidraw generator below applies to both
targets; the HTML generator is web only.

## ASCII Wireframe Primitives

Use these consistently across all wireframes:

```
┌─────────────────────────────────┐   ← container / card
│  [Label]  [Input Field      ]   │   ← label + text input
│  [Button: Primary Action    ]   │   ← primary button
│  [Button: Secondary]            │   ← secondary button
│  ○ Option A  ○ Option B         │   ← radio group
│  ☐ Checkbox label               │   ← checkbox
│  ▼ Dropdown / Select            │   ← select / combobox
│  ──────────────────────         │   ← divider
│  ⚠ Error message text           │   ← validation error
│  ✓ Success confirmation         │   ← success state
└─────────────────────────────────┘

[Nav: Logo | Item 1 | Item 2 | CTA]  ← navigation bar
[ Sidebar  ][     Main Content    ]  ← two-column layout
[  Col 1  ][  Col 2  ][  Col 3  ]   ← three-column grid
[         Full-width Banner         ]← hero / header band
```

## ASCII Wireframe — Example (Password Reset)

```
┌──────────────────────────────────────┐
│            Reset Password            │
│                                      │
│  Email                               │
│  [                              ]    │
│                                      │
│  ⚠ No account found for this email  │  ← error state
│                                      │
│  [Button: Send Reset Link       ]    │
│                                      │
│  ← Back to Login                     │
└──────────────────────────────────────┘
```

## Generate Wireframe Files

For each story, produce all three output files. Run these steps:

```bash
STORY_FILE="docs/stories/auth-reset-password-20250416143000-0001.md"
STORY_ID=$(python3 -c "
import re
m = re.search(r'^id:\s*[\"\'](.*?)[\"\']', open('$STORY_FILE').read(), re.MULTILINE)
print(m.group(1) if m else 'unknown')
")
mkdir -p docs/design/wireframes
```

**Step 1 — Write `${STORY_ID}.wireframe.md`** (ASCII layout + approval metadata):

```markdown
---
story_id: "{story_id}"
story_file: "{story_file}"
status: draft          # draft | approved | rejected
approved_by: null
approved_at: null
---

## Wireframe: {story title}

### Default state
{ASCII wireframe}

### Error state
{ASCII wireframe — validation error}

### Success state
{ASCII wireframe — confirmation}

### Interaction Notes

- Tab order: {list of focusable elements in tab sequence}
- Primary action: {describe}
- Error handling: {describe visible error states}

### Approval

- [ ] Approved by product owner
- [ ] Approved by UX lead (if applicable)
```

**Step 2 — Write `${STORY_ID}.wireframe.html`** (see HTML template below)

**Step 3 — Write `${STORY_ID}.wireframe.excalidraw`** (see Excalidraw template below)

After writing the files required for the active target, verify they exist (web: md, html, excalidraw; tui: md, excalidraw):

```bash
ls docs/design/wireframes/${STORY_ID}.wireframe.md
ls docs/design/wireframes/${STORY_ID}.wireframe.excalidraw
# web target also:
ls docs/design/wireframes/${STORY_ID}.wireframe.html 2>/dev/null || true
```

If any required file is missing, produce it before continuing.

## HTML Wireframe (web target only)

For web targets, always generate the HTML wireframe. Skip this section entirely for `tui`. Use multiple `<section>` blocks for multi-state wireframes (default, loading, error, success, empty).

```bash
cat > "docs/design/wireframes/${STORY_ID}.wireframe.html" <<'HTMLEOF'
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Wireframe: {story title}</title>
  <style>
    * { box-sizing: border-box; font-family: monospace; }
    body { background: #e9ecef; padding: 2rem; }
    h1 { font-size: 1rem; color: #495057; margin-bottom: 1.5rem; }
    .states { display: flex; flex-wrap: wrap; gap: 1.5rem; }
    .state { background: #f8f9fa; border: 1px solid #adb5bd; border-radius: 4px; padding: 0; min-width: 360px; }
    .state-label { background: #343a40; color: #fff; font-size: .75rem; padding: .25rem .75rem; border-radius: 4px 4px 0 0; }
    .frame { padding: 1.5rem; }
    .screen-title { font-weight: bold; font-size: 1.1rem; margin-bottom: 1rem; border-bottom: 1px solid #dee2e6; padding-bottom: .5rem; }
    label { display: block; font-size: .8rem; color: #495057; margin-bottom: .2rem; margin-top: .75rem; }
    .input { border: 1px solid #868e96; padding: .4rem .6rem; width: 100%; background: #fff; }
    .btn { border: none; padding: .5rem 1rem; cursor: default; margin-top: .75rem; width: 100%; font-weight: bold; }
    .btn-primary { background: #343a40; color: #fff; }
    .btn-secondary { background: transparent; border: 1px solid #343a40; color: #343a40; }
    .error { color: #c0392b; font-size: .8rem; margin-top: .25rem; }
    .success { color: #2d6a4f; font-size: .8rem; margin-top: .25rem; }
    .link { color: #1971c2; font-size: .85rem; margin-top: .75rem; display: block; }
    .nav { display: flex; gap: 1rem; background: #343a40; color: #fff; padding: .6rem 1rem; font-size: .85rem; margin-bottom: .75rem; }
    .badge { background: #868e96; color: #fff; font-size: .7rem; padding: .1rem .4rem; border-radius: 3px; }
    .divider { border: none; border-top: 1px solid #dee2e6; margin: .75rem 0; }
    .tab-order { font-size: .7rem; color: #868e96; margin-top: 1.5rem; border-top: 1px dashed #dee2e6; padding-top: .5rem; }
  </style>
</head>
<body>
  <h1>Wireframe: {story title} — {story_id}</h1>
  <div class="states">

    <div class="state">
      <div class="state-label">Default state</div>
      <div class="frame">
        <div class="screen-title">{Screen Title}</div>
        <!-- Add form fields, buttons, content blocks here -->
        <label>Field label</label>
        <input class="input" type="text" placeholder="placeholder" disabled />
        <div class="btn btn-primary">Primary Action</div>
        <a class="link" href="#">Secondary link</a>
        <div class="tab-order">Tab order: Field → Primary Action → Secondary link</div>
      </div>
    </div>

    <div class="state">
      <div class="state-label">Error state</div>
      <div class="frame">
        <div class="screen-title">{Screen Title}</div>
        <label>Field label</label>
        <input class="input" type="text" placeholder="placeholder" disabled style="border-color:#c0392b" />
        <div class="error">⚠ Error message describing the problem</div>
        <div class="btn btn-primary">Primary Action</div>
        <div class="tab-order">Tab order: Field → Primary Action</div>
      </div>
    </div>

    <div class="state">
      <div class="state-label">Success state</div>
      <div class="frame">
        <div class="screen-title">{Screen Title}</div>
        <div class="success">✓ Success confirmation message</div>
        <div class="btn btn-secondary">Back / Next step</div>
      </div>
    </div>

  </div>
</body>
</html>
HTMLEOF
```

Extend with real field names, content, and states from the story Gherkin. One `<div class="state">` block per Gherkin scenario.

## Excalidraw Wireframe (always required)

Always generate the Excalidraw file — every story, every run. Excalidraw is the canonical editable wireframe format reviewers annotate.

Write the file as valid JSON to `docs/design/wireframes/${STORY_ID}.wireframe.excalidraw`. Each UI element is one entry in the `elements` array. Use the element templates below, copy and adapt:

```bash
python3 - <<'PYEOF'
import json, pathlib, os

story_id = os.environ.get("STORY_ID", "unknown")
out = pathlib.Path(f"docs/design/wireframes/{story_id}.wireframe.excalidraw")
out.parent.mkdir(parents=True, exist_ok=True)

# ── Element helpers ──────────────────────────────────────────────────────────
def rect(id, x, y, w, h, label="", bg="transparent", stroke="#343a40", bold=False):
    els = [{
        "id": id, "type": "rectangle", "x": x, "y": y, "width": w, "height": h,
        "angle": 0, "strokeColor": stroke, "backgroundColor": bg,
        "fillStyle": "solid", "strokeWidth": 2, "strokeStyle": "solid",
        "roughness": 1, "opacity": 100, "groupIds": [], "roundness": {"type": 3},
        "version": 1, "versionNonce": 1, "isDeleted": False,
        "boundElements": None, "updated": 1, "link": None, "locked": False,
    }]
    if label:
        els.append(text(id + "_lbl", x + w/2, y + h/2, label, bold=bold, anchor="center"))
    return els

def text(id, x, y, content, bold=False, anchor="left", color="#343a40"):
    return {
        "id": id, "type": "text", "x": x, "y": y,
        "width": len(content) * 8, "height": 20,
        "angle": 0, "strokeColor": color, "backgroundColor": "transparent",
        "fillStyle": "solid", "strokeWidth": 1, "strokeStyle": "solid",
        "roughness": 1, "opacity": 100, "groupIds": [], "roundness": None,
        "version": 1, "versionNonce": 1, "isDeleted": False,
        "boundElements": None, "updated": 1, "link": None, "locked": False,
        "text": content, "fontSize": 16,
        "fontFamily": 3,  # monospace
        "textAlign": anchor, "verticalAlign": "middle",
        "baseline": 14, "containerId": None, "originalText": content,
        "lineHeight": 1.25,
        "fontWeight": "bold" if bold else "normal",
    }

def input_field(id, x, y, w, label_text):
    return (
        [text(id + "_lbl", x, y - 18, label_text)] +
        rect(id, x, y, w, 32, stroke="#868e96")
    )

def button_primary(id, x, y, w, label_text):
    return rect(id, x, y, w, 36, label=label_text, bg="#343a40", stroke="#343a40", bold=True)

def button_secondary(id, x, y, w, label_text):
    return rect(id, x, y, w, 36, label=label_text, bg="transparent", stroke="#343a40")

def section_label(id, x, y, content):
    return [text(id, x, y, f"[ {content} ]", color="#868e96")]

# ── Build elements ────────────────────────────────────────────────────────────
elements = []

# Outer frame
elements += rect("frame", 50, 30, 700, 600, stroke="#343a40")

# Screen title
elements.append(text("title", 70, 50, "{Screen Title}", bold=True))

# --- Default state ---
elements += section_label("s_default", 70, 90, "Default state")
elements += input_field("field1", 70, 130, 560, "Field label")
elements += button_primary("btn_primary", 70, 190, 560, "Primary Action")
elements.append(text("link1", 70, 238, "← Secondary link / back", color="#1971c2"))

# --- Error state ---
elements += section_label("s_error", 70, 280, "Error state")
elements += input_field("field1_err", 70, 320, 560, "Field label")
elements += rect("err_border", 70, 320, 560, 32, stroke="#c0392b")
elements.append(text("err_msg", 70, 360, "⚠ Error message text", color="#c0392b"))
elements += button_primary("btn_primary_err", 70, 390, 560, "Primary Action")

# --- Success state ---
elements += section_label("s_success", 70, 450, "Success state")
elements.append(text("success_msg", 70, 490, "✓ Success confirmation", color="#2d6a4f"))
elements += button_secondary("btn_back", 70, 520, 260, "Back / Next step")

doc = {
    "type": "excalidraw",
    "version": 2,
    "source": "MAPLE wireframe-architect",
    "elements": elements,
    "appState": {"viewBackgroundColor": "#f8f9fa", "gridSize": None},
    "files": {},
}
out.write_text(json.dumps(doc, indent=2) + "\n")
print(f"[wireframe] wrote {out}")
PYEOF
```

Adapt element positions and labels to match the actual story screens. Add more `rect`/`text`/`input_field`/`button_primary` calls per Gherkin scenario. Do not leave placeholder text (`{Screen Title}`) in the final file.

## Approval Gate

All three files must exist and the `.md` must be `status: approved` before `mockup` or `ui-mockup-builder` proceeds:

```bash
python3 - <<'EOF'
import re, sys, pathlib
sid = open(".claude/state/maple.json").read()  # or pass STORY_ID
# Check required artifacts exist (html is web-only)
cfg = open("project.config.yaml").read() if pathlib.Path("project.config.yaml").exists() else ""
tm = re.search(r'^\s*target:\s*(\w+)', cfg, re.MULTILINE)
target = tm.group(1) if tm else "web"
exts = ["md", "excalidraw"] if target == "tui" else ["md", "html", "excalidraw"]
for ext in exts:
    p = pathlib.Path(f"docs/design/wireframes/{sid}.wireframe.{ext}")
    if not p.exists():
        print(f"BLOCKED: missing {p}")
        sys.exit(1)
# Check approval status in .md
md = pathlib.Path(f"docs/design/wireframes/{sid}.wireframe.md").read_text()
m = re.search(r'^status:\s*(\w+)', md, re.MULTILINE)
status = m.group(1) if m else 'draft'
if status != 'approved':
    print(f"BLOCKED: wireframe {sid} not approved (status={status})")
    sys.exit(1)
print("approved — all three artifacts present")
EOF
```

## Failure Modes

| Condition | Action |
|---|---|
| Story has no Gherkin | Generate skeleton wireframe with placeholder states. Log `NO_GHERKIN — skeleton only`. |
| `docs/design/wireframes/` missing | Create it. |
| Wireframe `.md` exists and is `approved` | Do not overwrite. Log `SKIP — approved wireframe exists`. |
| Wireframe `.md` exists and is `draft` | Overwrite only if story Gherkin has changed. |
| `.html` or `.excalidraw` missing despite `.md` existing | Generate the missing file(s) immediately. |

## Logging

```
[wireframe] CREATE   docs/design/wireframes/auth-reset-0001.wireframe.md
[wireframe] CREATE   docs/design/wireframes/auth-reset-0001.wireframe.html
[wireframe] CREATE   docs/design/wireframes/auth-reset-0001.wireframe.excalidraw
[wireframe] SKIP     docs/design/wireframes/auth-reset-0001.wireframe.md  (approved — locked)
[wireframe] BLOCKED  auth-reset-0001  status=draft — needs approval before mockup
[wireframe] BLOCKED  auth-reset-0001  missing .html — generating now
```

