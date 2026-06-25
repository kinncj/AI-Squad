# Multi-target Design Phase Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make MAPLE's design/UX phase produce artifacts for the project's UI medium (web or TUI) by reading a single `design.target` config value, then prove it by running a new maple overlay through the full pipeline on the MAPLE repo itself.

**Architecture:** One set of medium-aware design agents/skills (no parallel `tui-*` agents). Each reads `design.target` from `project.config.yaml` and follows a target profile (`docs/design/design-targets.md`) to decide what to emit. Artifact paths and the a11y JSON shape stay identical across targets, so the SDLC gate scripts need no changes. Design review/approval works in two surfaces: a new in-TUI overlay in the maple dashboard (`D` key) and the existing browser portal degraded to monospace for terminal artifacts.

**Tech Stack:** Go 1.x + Bubble Tea/lipgloss (the `maple` binary in `tui/`), Bash (sdlc gate scripts + CLI test suite `tests/cli/test_ai_squad.sh`), Python 3 (design-review-portal), Markdown agent/skill prompt files, JSON Schema (`project.config.schema.json`).

## Global Constraints

- **Harness parity (structural):** Every agent/skill/taffy file exists byte-identically in three mirrors: `template/.claude/`, `template/.opencode/`, `template/.cursor/`. Any edit to one MUST be propagated to the other two. Pattern: edit the `.claude` copy, then `cp` it over the `.opencode` and `.cursor` copies, then `diff` to confirm identical.
- **Schema location:** `project.config.schema.json` lives ONLY in `template/.claude/schemas/` (not in the `.opencode`/`.cursor` mirrors). `project.config.yaml` lives only at `template/` root (and repo root).
- **Build dance:** Any change under `tui/` must be built with `make build-tui` (it replaces the `tui/template` symlink with a real copy for `go:embed`, builds, then restores the symlink). A bare `go build`/`go test` in `tui/` fails on the embed symlink.
- **Go unit tests:** run dance-wrapped: `rm -f tui/template && cp -rL template tui/template && (cd tui && go test ./... -run <Name>); rm -rf tui/template && ln -s ../template tui/template`. Full CLI suite: `make test`.
- **Default target is `web`:** existing users must see no behavior change. `design.target` defaults to `web` everywhere.
- **Artifact paths are fixed** (gates depend on them): wireframe `docs/design/wireframes/{id}.wireframe.md`, mockup `docs/design/mockups/{id}.mockup.md`, a11y `docs/design/mockups/{id}.a11y.json`. The a11y JSON must always contain `violations[]` where each item has an `impact` field in `critical|serious|moderate|minor`.
- **Commit messages:** imperative, lowercase, under 72 chars, humanized (no "enhance/leverage/ensure/implement/utilize/facilitate", no "improve maintainability"). NEVER add `Co-Authored-By: Claude`. Stage specific files; never `git add -A`.
- **Versioning:** this is a new feature → minor bump (`v4.x.y → v4.(x+1).0`). Update `CHANGELOG.md`.
- **Style:** no comments unless the *why* is non-obvious; no backwards-compat shims; keep overlay handlers grouped (check `if m.showX` before the global key switch, same order as `View()`).

---

## File Structure

**Config / schema:**
- Modify `template/.claude/schemas/project.config.schema.json` — add `design.target` enum.
- Modify `template/project.config.yaml` and repo-root `project.config.yaml` — add `design.target`.

**New profile doc:**
- Create `template/docs/design/design-targets.md` (+ copy to repo-root `docs/design/`).

**Agents (×3 mirrors each):** `wireframe-architect.md`, `ui-mockup-builder.md`, `a11y-auditor.md`, `design-system-author.md`, `visual-identity-designer.md`, `product-owner.md`, `orchestrator.md`.

**Skills (×3 mirrors each):** `wireframe/SKILL.md`, `mockup/SKILL.md`, `a11y-audit/SKILL.md`, `design-tokens/SKILL.md`, `visual-identity/SKILL.md`.

**Taffy (×3 mirrors):** `new-ui-feature.yaml`.

**Go (maple binary):**
- Create `tui/design_review.go` — `loadDesignReview`, `parseArtifactStatus`, `a11ySummary`, `approveDesignArtifact`.
- Create `tui/design_review_test.go` — unit tests.
- Modify `tui/dashboard.go` — model fields, `D` key, overlay handler, View dispatch, reload hook.
- Modify `tui/dashboard_views.go` — `designReviewView()` render + help/footer entries.

**Python:**
- Modify `template/scripts/design-review-portal.py` (+ repo-root copy) — monospace rendering for `.md`/tui artifacts.

**Gate verification:**
- Create `tests/cli/test_tui_design_gates.sh` — fixture proving gates pass on TUI artifacts.

**Docs / release:**
- Modify `template/docs/design/README.md`, `template/CLAUDE.md`, `template/OPENCODE.md`, `template/CURSOR.md`, `template/COPILOT.md`, `template/AGENTS.md`, repo-root `CLAUDE.md`, `CHANGELOG.md`.

---

## Phase 0 — Foundation: config, schema, target profile

### Task 0.1: Add `design.target` to schema and config files

**Files:**
- Modify: `template/.claude/schemas/project.config.schema.json` (the `design` object, after the `wireframe_format` property)
- Modify: `template/project.config.yaml` (the `design:` block)
- Modify: `project.config.yaml` (repo root — set `tui`)

**Interfaces:**
- Produces: config key `design.target` with values `web` (default) | `tui`, read by every design agent/skill and by the orchestrator.

- [ ] **Step 1: Add the `target` property to the schema.** In `template/.claude/schemas/project.config.schema.json`, inside `properties.design.properties`, after the `wireframe_format` block, add:

```json
,
        "target": {
          "type": "string",
          "description": "UI medium the design phase targets. web = HTML/React/CSS artifacts and browser a11y. tui = terminal ASCII wireframes, lipgloss mockups, and terminal a11y. Selects which artifacts the design agents emit; artifact paths are identical across targets.",
          "enum": ["web", "tui"],
          "default": "web"
        }
```

- [ ] **Step 2: Validate the schema is still valid JSON.**

Run: `python3 -c "import json; json.load(open('template/.claude/schemas/project.config.schema.json')); print('valid')"`
Expected: `valid`

- [ ] **Step 3: Add `target` to the template config.** In `template/project.config.yaml`, change the `design:` block to:

```yaml
design:
  target: web             # web | tui — UI medium the design phase targets
  ui_library: none        # mantine | tailwind | shadcn | none  (web only)
  tokens_file: docs/design/identity/tokens.json
  wireframe_format: ascii # ascii | svg | html  (web only)
```

- [ ] **Step 4: Set the repo-root config to `tui`.** In `/Users/kinncj/Development/kinncj/MAPLE/project.config.yaml`, under `design:`, set `target: tui` (add the line if the block lacks it). Leave the rest of that file unchanged.

- [ ] **Step 5: Verify both configs.**

Run: `grep -A1 'design:' template/project.config.yaml | grep target; grep -A1 'design:' project.config.yaml | grep target`
Expected: two lines — `  target: web` and `  target: tui`

- [ ] **Step 6: Commit.**

```bash
git add template/.claude/schemas/project.config.schema.json template/project.config.yaml project.config.yaml
git commit -m "add design.target config (web|tui), default web"
```

### Task 0.2: Write the target-profile reference doc

**Files:**
- Create: `template/docs/design/design-targets.md`
- Create: `docs/design/design-targets.md` (repo-root copy)

**Interfaces:**
- Produces: the canonical per-target artifact table that all design agents/skills cite. Consumed by every Phase 1/2/3 prompt edit (they reference this doc by path).

- [ ] **Step 1: Write the profile doc.** Create `template/docs/design/design-targets.md`:

```markdown
# Design Targets

The design phase produces artifacts for one UI medium, chosen by `design.target` in
`project.config.yaml` (default `web`). Design agents and skills read this value and emit
the matching column below. Artifact **paths** and the a11y **JSON shape** are identical
across targets — only the content differs — so the SDLC gates work unchanged for any target.

| Concern | `web` | `tui` |
|---|---|---|
| Wireframe | `{id}.wireframe.md` (ASCII) + `.html` (preview) + `.excalidraw` | `{id}.wireframe.md` (ASCII/box-drawing) + `.excalidraw` (editable diagram). No `.html`. |
| Mockup | `{id}.mockup.tsx` (React/Tailwind/Mantine) + `{id}.mockup.md` | `{id}.mockup.md` only: a fenced monospace terminal render + a Styles section annotating lipgloss (Foreground/Background/Border/Padding) per region. No `.tsx`. |
| Identity tokens | `tokens.css`, `tailwind.tokens.js`, `mantine.theme.ts` | `terminal-theme.json`: color roles mapped to ANSI-256/truecolor hex + lipgloss style names. |
| Accessibility | axe-core/pa11y against a preview URL | terminal a11y checklist (see below), written to the same `{id}.a11y.json` shape. |
| Review render | browser portal (HTML/SVG/TSX preview) | ASCII inline in the maple `D` overlay; monospace `<pre>` in the portal; `.excalidraw` opens in the portal. |

All targets write to: `docs/design/wireframes/{id}.wireframe.md`, `docs/design/mockups/{id}.mockup.md`,
`docs/design/mockups/{id}.a11y.json`, `docs/design/identity/`.

## Terminal a11y checklist (`tui` target)

Each check that fails becomes a `violations[]` entry in `{id}.a11y.json` with an `impact`:

| Check | id | impact when failing |
|---|---|---|
| Every action reachable by keyboard (no mouse-only path) | `keyboard-reachable` | critical |
| Selected/focused element is visually distinct (not color-only) | `focus-visible` | serious |
| Foreground/background pairs meet WCAG 2.2 AA contrast (4.5:1 text, 3:1 large/UI) | `color-contrast` | serious |
| State never conveyed by color alone (has symbol/text too) | `color-only-signaling` | serious |
| Degrades under `NO_COLOR` / monochrome terminals | `no-color-support` | moderate |
| No content loss/overflow at the minimum supported width | `min-width-resize` | moderate |

## a11y JSON shape (all targets)

```json
{
  "target": "tui",
  "url": "<story-id> terminal UI",
  "timestamp": "<ISO-8601>",
  "testEngine": { "name": "maple-tui-a11y", "version": "1" },
  "violations": [
    {
      "id": "color-contrast",
      "impact": "serious",
      "description": "status text on muted background is 3.1:1",
      "help": "raise contrast to >= 4.5:1",
      "nodes": [ { "target": ["footer status"], "failureSummary": "fg #888 on bg #1a1a1a" } ]
    }
  ],
  "passes": []
}
```

The gate reads only `violations[].impact` and fails on any `critical` or `serious`.
```

- [ ] **Step 2: Copy to the repo-root dogfood docs.**

Run: `cp template/docs/design/design-targets.md docs/design/design-targets.md`

- [ ] **Step 3: Verify both exist and mention both targets.**

Run: `grep -l 'Terminal a11y checklist' template/docs/design/design-targets.md docs/design/design-targets.md`
Expected: both paths printed.

- [ ] **Step 4: Commit.**

```bash
git add template/docs/design/design-targets.md docs/design/design-targets.md
git commit -m "add design-targets profile doc (web vs tui artifacts)"
```

### Task 0.3: Generate `design.target` in fresh-project config (Go)

**Files:**
- Modify: `tui/init.go` — `projectConfigYAML` (~line 466)

**Why:** `maple init` writes a NEW project's `project.config.yaml` from the Go generator
`projectConfigYAML`, NOT from `template/project.config.yaml`. Without this, fresh projects created
the usual way (e.g. in `/tmp/maple_testing_test`) would lack `design.target`.

- [ ] **Step 1:** In `projectConfigYAML`, add `target: web             # web | tui — UI medium the design phase targets` as the first line of the `design:` block.
- [ ] **Step 2: Build with the dance.** Run: `make build-tui` — Expected: `Built: ./maple`.
- [ ] **Step 3: Commit.** `git add tui/init.go && git commit -m "init: generate design.target: web in new project config"`

---

## Phase 1 — Target-aware design agents (×3 mirrors)

> Each task in this phase: insert a `## Target awareness` section into the `.claude` copy at the insertion point, propagate to `.opencode`/`.cursor` with `cp`, then `diff` to confirm parity. A shared helper for the propagation+verify step is used in every task:
>
> ```bash
> f="agents/<name>.md"   # or skills/<name>/SKILL.md
> cp "template/.claude/$f" "template/.opencode/$f" && cp "template/.claude/$f" "template/.cursor/$f"
> diff "template/.claude/$f" "template/.opencode/$f" && diff "template/.claude/$f" "template/.cursor/$f" && echo "PARITY OK"
> ```

### Task 1.1: wireframe-architect — target awareness

**Files:**
- Modify: `template/.claude/agents/wireframe-architect.md` (after the output-format section, ~line 28)
- Propagate: `template/.opencode/agents/wireframe-architect.md`, `template/.cursor/agents/wireframe-architect.md`

- [ ] **Step 1: Replace the "always produce all three" instruction.** In `template/.claude/agents/wireframe-architect.md`, find the bullet that says all three output files (`.md`, `.html`, `.excalidraw`) are always required, and replace that bullet plus add a target section so it reads:

```markdown
## Target awareness

Read `design.target` from `project.config.yaml` (default `web`). Emit only the artifacts for that
medium, per `docs/design/design-targets.md`. The output path is always
`docs/design/wireframes/<story-id>.wireframe.{md,...}` and the `.md` carries `status:` frontmatter.

- **web** — produce `.md` (ASCII), `.html` (browser preview), and `.excalidraw` (editable diagram).
- **tui** — produce `.md` (ASCII/box-drawing layout of panes, overlays, and status bar, with a
  keybinding legend and focus order) and `.excalidraw` (the same layout as an editable diagram:
  panes/overlays as rectangles, labels, state-transition arrows). Do NOT produce `.html`.

Producing fewer than the required artifacts for the active target is an incomplete run.
```

- [ ] **Step 2: Propagate to the two mirrors and verify parity.**

Run:
```bash
f="agents/wireframe-architect.md"; cp "template/.claude/$f" "template/.opencode/$f" && cp "template/.claude/$f" "template/.cursor/$f" && diff "template/.claude/$f" "template/.opencode/$f" && diff "template/.claude/$f" "template/.cursor/$f" && echo "PARITY OK"
```
Expected: `PARITY OK` (no diff output).

- [ ] **Step 3: Verify the section is present in all three.**

Run: `grep -l '## Target awareness' template/.claude/agents/wireframe-architect.md template/.opencode/agents/wireframe-architect.md template/.cursor/agents/wireframe-architect.md`
Expected: all three paths.

- [ ] **Step 4: Commit.**

```bash
git add template/.claude/agents/wireframe-architect.md template/.opencode/agents/wireframe-architect.md template/.cursor/agents/wireframe-architect.md
git commit -m "wireframe-architect: emit per design.target (tui = md + excalidraw)"
```

### Task 1.2: ui-mockup-builder — target awareness

**Files:**
- Modify: `template/.claude/agents/ui-mockup-builder.md` (after the output spec, before "## Skill Usage")
- Propagate: `.opencode`, `.cursor` copies

- [ ] **Step 1: Insert the target section.** Add to `template/.claude/agents/ui-mockup-builder.md`:

```markdown
## Target awareness

Read `design.target` from `project.config.yaml` (default `web`), per `docs/design/design-targets.md`.
Always write `docs/design/mockups/<story-id>.mockup.md` with `status:` frontmatter.

- **web** — also write `<story-id>.mockup.tsx` (or `.html`) using the project `ui_library` stack
  (react-mantine / react-tailwind / react-shadcn / html).
- **tui** — write ONLY `<story-id>.mockup.md`. It contains: (1) a fenced code block with the
  high-fidelity terminal render at the target width, showing every state (default, selected,
  empty, error, loading); and (2) a "Styles" section annotating each region with its lipgloss
  styling (Foreground, Background, Border, Padding) drawn from `terminal-theme.json`. Do NOT
  write `.tsx`/`.html` and do NOT run a stack detector.
```

- [ ] **Step 2: Propagate + verify parity** (use the helper from the Phase 1 intro with `f="agents/ui-mockup-builder.md"`).
Expected: `PARITY OK`

- [ ] **Step 3: Commit.**

```bash
git add template/.claude/agents/ui-mockup-builder.md template/.opencode/agents/ui-mockup-builder.md template/.cursor/agents/ui-mockup-builder.md
git commit -m "ui-mockup-builder: tui target emits lipgloss-annotated mockup.md"
```

### Task 1.3: a11y-auditor — target awareness (terminal checklist, same JSON)

**Files:**
- Modify: `template/.claude/agents/a11y-auditor.md` (after the report-path line, before "## Skill Usage")
- Propagate: `.opencode`, `.cursor` copies

- [ ] **Step 1: Insert the target section.** Add to `template/.claude/agents/a11y-auditor.md`:

```markdown
## Target awareness

Read `design.target` from `project.config.yaml` (default `web`), per `docs/design/design-targets.md`.
Always write the report to `docs/design/mockups/<story-id>.a11y.json` in the shape below, where each
`violations[]` entry has an `impact` of `critical|serious|moderate|minor`. The gate fails on any
`critical` or `serious`.

- **web** — run axe/pa11y against the preview URL and write its native JSON to that path.
- **tui** — there is no browser. Audit the approved mockup against the terminal a11y checklist in
  `docs/design/design-targets.md` (keyboard-reachable=critical, focus-visible=serious,
  color-contrast=serious, color-only-signaling=serious, no-color-support=moderate,
  min-width-resize=moderate) and write the findings as `violations[]` in the same JSON shape:

```json
{
  "target": "tui",
  "url": "<story-id> terminal UI",
  "timestamp": "<ISO-8601>",
  "testEngine": { "name": "maple-tui-a11y", "version": "1" },
  "violations": [],
  "passes": []
}
```

Compute color-contrast from the `terminal-theme.json` foreground/background pairs (WCAG 2.2 AA:
4.5:1 normal text, 3:1 large text and UI components).
```

- [ ] **Step 2: Propagate + verify parity** (`f="agents/a11y-auditor.md"`).
Expected: `PARITY OK`

- [ ] **Step 3: Commit.**

```bash
git add template/.claude/agents/a11y-auditor.md template/.opencode/agents/a11y-auditor.md template/.cursor/agents/a11y-auditor.md
git commit -m "a11y-auditor: tui target audits terminal checklist into same a11y.json"
```

### Task 1.4: design-system-author — target awareness (terminal theme emit)

**Files:**
- Modify: `template/.claude/agents/design-system-author.md` (after the emit-targets list)
- Propagate: `.opencode`, `.cursor` copies

- [ ] **Step 1: Insert the target section.** Add to `template/.claude/agents/design-system-author.md`:

```markdown
## Target awareness

Read `design.target` from `project.config.yaml` (default `web`), per `docs/design/design-targets.md`.
`tokens.json` (W3C DTCG) is always the canonical source.

- **web** — emit `docs/design/identity/tokens.css`, `tailwind.tokens.js`, `mantine.theme.ts`.
- **tui** — emit `docs/design/identity/terminal-theme.json`: map each color role from `tokens.json`
  to an ANSI-256 or truecolor hex value and a lipgloss style name (e.g. `primary`, `muted`,
  `accent`, `error`, `success`, `border`). Include per role the foreground/background pairing used,
  so the a11y auditor can compute contrast. Do NOT emit CSS/Tailwind/Mantine.
```

- [ ] **Step 2: Propagate + verify parity** (`f="agents/design-system-author.md"`).
Expected: `PARITY OK`

- [ ] **Step 3: Commit.**

```bash
git add template/.claude/agents/design-system-author.md template/.opencode/agents/design-system-author.md template/.cursor/agents/design-system-author.md
git commit -m "design-system-author: tui target emits terminal-theme.json"
```

### Task 1.5: visual-identity-designer — target awareness (terminal palette)

**Files:**
- Modify: `template/.claude/agents/visual-identity-designer.md` (after the output spec)
- Propagate: `.opencode`, `.cursor` copies

- [ ] **Step 1: Insert the target section.** Add to `template/.claude/agents/visual-identity-designer.md`:

```markdown
## Target awareness

Read `design.target` from `project.config.yaml` (default `web`), per `docs/design/design-targets.md`.
`palette.json`, `typography.json`, and `tokens.json` are written for all targets.

- **web** — full color palette and web font stacks.
- **tui** — constrain the palette to terminal-safe colors (assume an ANSI-256 floor; truecolor is a
  bonus, not a requirement) and verify every foreground/background role pair meets WCAG 2.2 AA
  contrast. Typography is limited to what a terminal offers: bold, dim/faint, italic (if supported),
  underline, reverse — record these as the `typography.json` "weights"/"styles" instead of font
  families and px sizes.
```

- [ ] **Step 2: Propagate + verify parity** (`f="agents/visual-identity-designer.md"`).
Expected: `PARITY OK`

- [ ] **Step 3: Commit.**

```bash
git add template/.claude/agents/visual-identity-designer.md template/.opencode/agents/visual-identity-designer.md template/.cursor/agents/visual-identity-designer.md
git commit -m "visual-identity-designer: tui target constrains palette to terminal-safe + AA"
```

---

## Phase 2 — Target-aware design skills (×3 mirrors)

> Same propagate+verify helper as Phase 1, with `f="skills/<name>/SKILL.md"`.

### Task 2.1: wireframe skill — target branch

**Files:**
- Modify: `template/.claude/skills/wireframe/SKILL.md` (after the output-formats table)
- Propagate: `.opencode`, `.cursor`

- [ ] **Step 1: Insert after the output-formats table:**

```markdown
## Target awareness

`design.target` (`project.config.yaml`, default `web`) decides which files are required:

- **web** — `<id>.wireframe.md` (ASCII) + `.html` (preview) + `.excalidraw`.
- **tui** — `<id>.wireframe.md` (ASCII/box-drawing) + `.excalidraw`. No `.html`.

Use box-drawing primitives (`┌ ─ ┐ │ └ ┘ ├ ┤`) for tui layouts, label panes/overlays/status bar,
and include the keybinding legend and focus order inline in the `.md`.
```

- [ ] **Step 2: Propagate + verify parity** (`f="skills/wireframe/SKILL.md"`). Expected: `PARITY OK`
- [ ] **Step 3: Commit.**

```bash
git add template/.claude/skills/wireframe/SKILL.md template/.opencode/skills/wireframe/SKILL.md template/.cursor/skills/wireframe/SKILL.md
git commit -m "wireframe skill: tui target requires md + excalidraw, no html"
```

### Task 2.2: mockup skill — target branch

**Files:**
- Modify: `template/.claude/skills/mockup/SKILL.md` (after the output-formats table)
- Propagate: `.opencode`, `.cursor`

- [ ] **Step 1: Insert:**

```markdown
## Target awareness

`design.target` (`project.config.yaml`, default `web`):

- **web** — generate the code mockup (`.tsx`/`.html`) for the configured `ui_library`, plus `.mockup.md`.
- **tui** — generate ONLY `<id>.mockup.md`: a fenced monospace terminal render at the target width
  covering default/selected/empty/error/loading states, followed by a "Styles" section mapping each
  region to lipgloss `Foreground`/`Background`/`Border`/`Padding` taken from
  `docs/design/identity/terminal-theme.json`. Do not run the stack detector.
```

- [ ] **Step 2: Propagate + verify parity** (`f="skills/mockup/SKILL.md"`). Expected: `PARITY OK`
- [ ] **Step 3: Commit.**

```bash
git add template/.claude/skills/mockup/SKILL.md template/.opencode/skills/mockup/SKILL.md template/.cursor/skills/mockup/SKILL.md
git commit -m "mockup skill: tui target emits lipgloss-annotated mockup.md only"
```

### Task 2.3: a11y-audit skill — terminal checklist branch

**Files:**
- Modify: `template/.claude/skills/a11y-audit/SKILL.md` (after the parse/classify section)
- Propagate: `.opencode`, `.cursor`

- [ ] **Step 1: Insert:**

```markdown
## Target awareness

When `design.target` is `tui`, skip browser tooling (axe/pa11y). Instead evaluate the terminal a11y
checklist from `docs/design/design-targets.md` and write the findings to
`docs/design/mockups/<id>.a11y.json` using the SAME object shape the web path produces — an object
with `violations[]`, where each violation has `id`, `impact` (`critical|serious|moderate|minor`),
`description`, `help`, and `nodes[]` (`{target:[...], failureSummary}`). The downstream gate
(`scripts/sdlc/a11y-gate.sh`) reads only `violations[].impact`, so matching this shape keeps the
gate unchanged. Map checklist failures to impacts: keyboard-reachable→critical, focus-visible→serious,
color-contrast→serious, color-only-signaling→serious, no-color-support→moderate, min-width-resize→moderate.
```

- [ ] **Step 2: Propagate + verify parity** (`f="skills/a11y-audit/SKILL.md"`). Expected: `PARITY OK`
- [ ] **Step 3: Commit.**

```bash
git add template/.claude/skills/a11y-audit/SKILL.md template/.opencode/skills/a11y-audit/SKILL.md template/.cursor/skills/a11y-audit/SKILL.md
git commit -m "a11y-audit skill: tui target writes terminal findings in same json shape"
```

### Task 2.4: design-tokens skill — terminal emit branch

**Files:**
- Modify: `template/.claude/skills/design-tokens/SKILL.md` (after the Mantine emit block, before "## Run All Emitters")
- Propagate: `.opencode`, `.cursor`

- [ ] **Step 1: Insert a terminal emitter section:**

```markdown
## Emit: terminal theme (tui target)

When `design.target` is `tui`, emit `docs/design/identity/terminal-theme.json` from `tokens.json`
instead of the CSS/Tailwind/Mantine outputs. Shape:

```json
{
  "roles": {
    "primary":  { "fg": "#7aa2f7", "ansi256": 111, "lipgloss": "primary" },
    "muted":    { "fg": "#565f89", "ansi256": 60,  "lipgloss": "muted" },
    "accent":   { "fg": "#bb9af7", "ansi256": 141, "lipgloss": "accent" },
    "error":    { "fg": "#f7768e", "ansi256": 204, "lipgloss": "error" },
    "success":  { "fg": "#9ece6a", "ansi256": 149, "lipgloss": "success" },
    "background": { "bg": "#1a1b26", "ansi256": 234 },
    "foreground": { "fg": "#c0caf5", "ansi256": 189 }
  },
  "pairs": [
    { "role": "foreground", "on": "background", "contrast": 12.6 },
    { "role": "muted", "on": "background", "contrast": 3.4 }
  ]
}
```

Compute each `contrast` ratio so the a11y auditor can flag pairs under WCAG 2.2 AA.
```

- [ ] **Step 2: Propagate + verify parity** (`f="skills/design-tokens/SKILL.md"`). Expected: `PARITY OK`
- [ ] **Step 3: Commit.**

```bash
git add template/.claude/skills/design-tokens/SKILL.md template/.opencode/skills/design-tokens/SKILL.md template/.cursor/skills/design-tokens/SKILL.md
git commit -m "design-tokens skill: add terminal-theme.json emitter for tui target"
```

### Task 2.5: visual-identity skill — terminal palette guidance

**Files:**
- Modify: `template/.claude/skills/visual-identity/SKILL.md` (after the output-files table)
- Propagate: `.opencode`, `.cursor`

- [ ] **Step 1: Insert:**

```markdown
## Target awareness

When `design.target` is `tui`, keep the same JSON outputs (`palette.json`, `typography.json`,
`tokens.json`) but constrain choices to the terminal: colors must be expressible in ANSI-256 (record
the nearest 256 index alongside the hex), every fg/bg role pair must clear WCAG 2.2 AA contrast, and
`typography.json` describes terminal text styles (bold, dim, italic, underline, reverse) rather than
font families and px sizes.
```

- [ ] **Step 2: Propagate + verify parity** (`f="skills/visual-identity/SKILL.md"`). Expected: `PARITY OK`
- [ ] **Step 3: Commit.**

```bash
git add template/.claude/skills/visual-identity/SKILL.md template/.opencode/skills/visual-identity/SKILL.md template/.cursor/skills/visual-identity/SKILL.md
git commit -m "visual-identity skill: tui target constrains to ansi-256 + AA"
```

---

## Phase 3 — Pipeline wiring (×3 mirrors)

### Task 3.1: orchestrator — make the Design Gate medium-neutral

**Files:**
- Modify: `template/.claude/agents/orchestrator.md` ("## Design Gate (ui: true stories)" + "## Design Agent Routing")
- Propagate: `.opencode`, `.cursor`

- [ ] **Step 1: Rewrite the Design Gate + routing to reference `design.target`.** Replace those two sections with:

```markdown
## Design Gate (ui: true stories)

When a story frontmatter contains `ui: true`, insert a design sub-pipeline before Phase 5 IMPLEMENT.
The same agents run for every project; they read `design.target` from `project.config.yaml` (default
`web`) and emit artifacts for that medium, per `docs/design/design-targets.md`.

1. **UX Research** — `@ux-researcher`: personas + journey map.
2. **Wireframe** — `@wireframe-architect`: wireframe for every screen/overlay state. **PAUSE — await human wireframe approval.**
3. **Visual Identity** — if `docs/design/identity/tokens.json` is missing, `@visual-identity-designer`. **PAUSE — await human palette approval.**
4. **Design Tokens** — `@design-system-author`: emit the identity tokens for the active target.
5. **Mockup** — `@ui-mockup-builder`: high-fidelity mockup for the active target. **PAUSE — await human mockup approval.**
6. **Component Scaffold** — run `component-scaffold` skill (web target only; skip for tui).
7. After Phase 5 IMPLEMENT: `@a11y-auditor`. Gate: no critical/serious accessibility violations.

Human approvals can be granted in the maple `D` (Design Review) overlay or the browser portal.
If any approval is not received, do not advance. Log `AWAITING APPROVAL` and surface to human.

## Design Agent Routing

| Task | Agent |
|---|---|
| User personas, journey maps | `@ux-researcher` |
| Wireframes (per design.target) | `@wireframe-architect` |
| Palette, typography, spacing | `@visual-identity-designer` |
| Token authoring + target emit | `@design-system-author` |
| High-fidelity mockups (per design.target) | `@ui-mockup-builder` |
| Accessibility audit | `@a11y-auditor` |
```

- [ ] **Step 2: Propagate + verify parity** (`f="agents/orchestrator.md"`). Expected: `PARITY OK`
- [ ] **Step 3: Confirm no remaining hard web-only routing wording.**

Run: `grep -niE 'CSS vars, Tailwind|Storybook story URL|ASCII/SVG/HTML' template/.claude/agents/orchestrator.md || echo "clean"`
Expected: `clean`

- [ ] **Step 4: Commit.**

```bash
git add template/.claude/agents/orchestrator.md template/.opencode/agents/orchestrator.md template/.cursor/agents/orchestrator.md
git commit -m "orchestrator: design gate reads design.target, drop web-only wording"
```

### Task 3.2: product-owner — ui:true means web or terminal

**Files:**
- Modify: `template/.claude/agents/product-owner.md` (the `ui: true` definition)
- Propagate: `.opencode`, `.cursor`

- [ ] **Step 1: Replace the `ui: true` definition** so it reads:

```markdown
**`ui: true`** for any story where the user sees or interacts with a rendered surface — whether a web
page or a terminal UI (pages, cards, modals, forms, navigation, overlays, panes, status bars). The
medium comes from `design.target` in `project.config.yaml`, not from the story. When in doubt, set `ui: true`.
**`ui: false`** only for purely backend, data-pipeline, or infrastructure stories with zero rendered output.
```

- [ ] **Step 2: Propagate + verify parity** (`f="agents/product-owner.md"`). Expected: `PARITY OK`
- [ ] **Step 3: Commit.**

```bash
git add template/.claude/agents/product-owner.md template/.opencode/agents/product-owner.md template/.cursor/agents/product-owner.md
git commit -m "product-owner: ui:true covers terminal UIs, medium from design.target"
```

### Task 3.3: new-ui-feature taffy — medium-neutral stage descriptions

**Files:**
- Modify: `template/.claude/taffy/new-ui-feature.yaml` (the `wireframe`, `design-tokens`, `ui-mockup-builder`, `a11y-audit` stage `description:` lines)
- Propagate: `.opencode`, `.cursor`

- [ ] **Step 1: Neutralize the four web-flavored stage descriptions.** Edit only the `description:` strings:
  - `wireframe`: `"Wireframes for every screen/overlay state (format per design.target)"`
  - `design-tokens`: `"Emit identity tokens for the target medium from approved tokens.json"`
  - `ui-mockup-builder`: `"High-fidelity mockup for the target medium from approved wireframe and tokens"`
  - `a11y-audit`: `"Accessibility audit for the target medium — blocks merge on critical/serious"`

  Leave all `agent:`, `gate:`, `when:`, `depends_on:` fields unchanged.

- [ ] **Step 2: Validate YAML still parses.**

Run: `python3 -c "import yaml; yaml.safe_load(open('template/.claude/taffy/new-ui-feature.yaml')); print('valid')"`
Expected: `valid`

- [ ] **Step 3: Propagate + verify parity** (`f="taffy/new-ui-feature.yaml"`). Expected: `PARITY OK`
- [ ] **Step 4: Commit.**

```bash
git add template/.claude/taffy/new-ui-feature.yaml template/.opencode/taffy/new-ui-feature.yaml template/.cursor/taffy/new-ui-feature.yaml
git commit -m "new-ui-feature: medium-neutral stage descriptions"
```

---

## Phase 4 — Prove the gates pass on TUI artifacts (no gate change)

### Task 4.1: Fixture test that TUI artifacts satisfy both gates

**Files:**
- Create: `tests/cli/test_tui_design_gates.sh`

**Interfaces:**
- Consumes: `scripts/sdlc/design-approved-gate.sh`, `scripts/sdlc/a11y-gate.sh` (unchanged).
- Produces: a runnable proof that a `tui` story with `.md` wireframe/mockup + a11y JSON passes.

- [ ] **Step 1: Write the test.** Create `tests/cli/test_tui_design_gates.sh`:

```bash
#!/usr/bin/env bash
# Proves the SDLC design gates pass on terminal (tui) artifacts unchanged.
set -uo pipefail
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
cd "$TMP"
ROOT="${REPO_ROOT:-$OLDPWD}"

mkdir -p docs/stories docs/design/wireframes docs/design/mockups
cat > docs/stories/STORY-TUI.md <<'EOF'
---
id: "STORY-TUI"
title: "Gate status overlay"
ui: true
phase: validate
---
Scenario: shows gate status
EOF
printf 'status: approved\n# wireframe\n' > docs/design/wireframes/STORY-TUI.wireframe.md
printf 'status: approved\n# mockup\n'    > docs/design/mockups/STORY-TUI.mockup.md
cat > docs/design/mockups/STORY-TUI.a11y.json <<'EOF'
{ "target": "tui", "violations": [ { "id": "no-color-support", "impact": "moderate" } ], "passes": [] }
EOF

bash "$ROOT/scripts/sdlc/design-approved-gate.sh" docs/stories/STORY-TUI.md || { echo "FAIL design gate"; exit 1; }
bash "$ROOT/scripts/sdlc/a11y-gate.sh"            docs/stories/STORY-TUI.md || { echo "FAIL a11y gate"; exit 1; }
echo "PASS tui design gates"
```

- [ ] **Step 2: Make it executable and run it.**

Run: `chmod +x tests/cli/test_tui_design_gates.sh && REPO_ROOT="$(pwd)" bash tests/cli/test_tui_design_gates.sh`
Expected: ends with `PASS tui design gates`

- [ ] **Step 3: Add a negative check** (a critical violation must FAIL the a11y gate). Append before the final echo:

```bash
cat > docs/design/mockups/STORY-TUI.a11y.json <<'EOF'
{ "target": "tui", "violations": [ { "id": "keyboard-reachable", "impact": "critical" } ] }
EOF
if bash "$ROOT/scripts/sdlc/a11y-gate.sh" docs/stories/STORY-TUI.md; then echo "FAIL: critical should block"; exit 1; fi
```

- [ ] **Step 4: Re-run; confirm still `PASS tui design gates`.**

Run: `REPO_ROOT="$(pwd)" bash tests/cli/test_tui_design_gates.sh`
Expected: `PASS tui design gates`

- [ ] **Step 5: Commit.**

```bash
git add tests/cli/test_tui_design_gates.sh
git commit -m "test: tui artifacts pass design+a11y gates, critical still blocks"
```

---

## Phase 5 — In-TUI Design Review overlay (Go, TDD)

### Task 5.1: design_review.go core readers (TDD)

**Files:**
- Create: `tui/design_review.go`
- Create: `tui/design_review_test.go`

**Interfaces:**
- Produces (consumed by Tasks 5.2–5.4):
  - `type designArtifact struct { Kind, Path, Status, Summary string; Exists bool }`
  - `type designReview struct { StoryID string; Artifacts []designArtifact }`
  - `func parseArtifactStatus(content string) string` — returns the `status:` value or `"draft"`.
  - `func a11ySummary(jsonBytes []byte) (crit int, total int)` — counts critical/serious and total violations.
  - `func loadDesignReview(storyID string) designReview` — reads the three artifact paths (relative to cwd).
  - `func approveDesignArtifact(path string) error` — rewrites the first `status:` line to `approved`.

- [ ] **Step 1: Write failing tests.** Create `tui/design_review_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseArtifactStatus(t *testing.T) {
	if got := parseArtifactStatus("status: approved\n# x\n"); got != "approved" {
		t.Errorf("got %q", got)
	}
	if got := parseArtifactStatus("# no frontmatter"); got != "draft" {
		t.Errorf("default got %q", got)
	}
}

func TestA11ySummary(t *testing.T) {
	j := []byte(`{"violations":[{"impact":"critical"},{"impact":"serious"},{"impact":"minor"}]}`)
	crit, total := a11ySummary(j)
	if crit != 2 || total != 3 {
		t.Errorf("crit=%d total=%d", crit, total)
	}
}

func TestLoadDesignReviewAndApprove(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)
	os.MkdirAll("docs/design/wireframes", 0o755)
	os.MkdirAll("docs/design/mockups", 0o755)
	os.WriteFile("docs/design/wireframes/S1.wireframe.md", []byte("status: draft\n"), 0o644)
	os.WriteFile("docs/design/mockups/S1.mockup.md", []byte("status: approved\n"), 0o644)
	os.WriteFile("docs/design/mockups/S1.a11y.json", []byte(`{"violations":[{"impact":"serious"}]}`), 0o644)

	r := loadDesignReview("S1")
	if len(r.Artifacts) != 3 {
		t.Fatalf("want 3 artifacts, got %d", len(r.Artifacts))
	}
	if r.Artifacts[0].Kind != "wireframe" || r.Artifacts[0].Status != "draft" {
		t.Errorf("wireframe: %+v", r.Artifacts[0])
	}
	if r.Artifacts[2].Kind != "a11y" || r.Artifacts[2].Summary != "1 critical/serious" {
		t.Errorf("a11y: %+v", r.Artifacts[2])
	}

	if err := approveDesignArtifact(filepath.Join("docs/design/wireframes/S1.wireframe.md")); err != nil {
		t.Fatal(err)
	}
	if parseArtifactStatus(readFileT(t, "docs/design/wireframes/S1.wireframe.md")) != "approved" {
		t.Errorf("approve did not set status")
	}
}

func readFileT(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
```

- [ ] **Step 2: Run the tests; verify they FAIL to compile (functions undefined).**

Run: `rm -f tui/template && cp -rL template tui/template; (cd tui && go test ./... -run 'TestParseArtifactStatus|TestA11ySummary|TestLoadDesignReviewAndApprove'); rm -rf tui/template && ln -s ../template tui/template`
Expected: FAIL — `undefined: parseArtifactStatus` etc.

- [ ] **Step 3: Implement.** Create `tui/design_review.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type designArtifact struct {
	Kind    string // "wireframe" | "mockup" | "a11y"
	Path    string
	Status  string // "approved" | "draft" | "" (a11y has no status)
	Summary string
	Exists  bool
}

type designReview struct {
	StoryID   string
	Artifacts []designArtifact
}

var statusLineRe = regexp.MustCompile(`(?m)^status:\s*(\w+)`)

func parseArtifactStatus(content string) string {
	if m := statusLineRe.FindStringSubmatch(content); m != nil {
		return m[1]
	}
	return "draft"
}

func a11ySummary(jsonBytes []byte) (crit int, total int) {
	var data struct {
		Violations []struct {
			Impact string `json:"impact"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return 0, 0
	}
	total = len(data.Violations)
	for _, v := range data.Violations {
		if v.Impact == "critical" || v.Impact == "serious" {
			crit++
		}
	}
	return crit, total
}

func loadDesignReview(storyID string) designReview {
	r := designReview{StoryID: storyID}
	wf := designArtifact{Kind: "wireframe", Path: "docs/design/wireframes/" + storyID + ".wireframe.md"}
	if b, err := os.ReadFile(wf.Path); err == nil {
		wf.Exists = true
		wf.Status = parseArtifactStatus(string(b))
	}
	mk := designArtifact{Kind: "mockup", Path: "docs/design/mockups/" + storyID + ".mockup.md"}
	if b, err := os.ReadFile(mk.Path); err == nil {
		mk.Exists = true
		mk.Status = parseArtifactStatus(string(b))
	}
	a := designArtifact{Kind: "a11y", Path: "docs/design/mockups/" + storyID + ".a11y.json"}
	if b, err := os.ReadFile(a.Path); err == nil {
		a.Exists = true
		crit, total := a11ySummary(b)
		a.Summary = fmt.Sprintf("%d critical/serious", crit)
		if total == 0 {
			a.Summary = "no audit findings"
		}
	}
	r.Artifacts = []designArtifact{wf, mk, a}
	return r
}

func approveDesignArtifact(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	if statusLineRe.MatchString(content) {
		content = statusLineRe.ReplaceAllString(content, "status: approved")
	} else {
		content = "status: approved\n" + content
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func (a designArtifact) label() string {
	switch a.Kind {
	case "a11y":
		if !a.Exists {
			return "a11y.json        [missing]"
		}
		return "a11y.json        " + a.Summary
	default:
		st := a.Status
		if !a.Exists {
			st = "missing"
		}
		return fmt.Sprintf("%-16s [%s]", a.Kind+".md", strings.ToUpper(st))
	}
}
```

- [ ] **Step 4: Run the tests; verify PASS.**

Run: `rm -f tui/template && cp -rL template tui/template; (cd tui && go test ./... -run 'TestParseArtifactStatus|TestA11ySummary|TestLoadDesignReviewAndApprove' -v); rm -rf tui/template && ln -s ../template tui/template`
Expected: `ok` / all three tests PASS.

- [ ] **Step 5: Commit.**

```bash
git add tui/design_review.go tui/design_review_test.go
git commit -m "add design review readers: load artifacts, a11y summary, approve"
```

### Task 5.2: Wire model state + reload

**Files:**
- Modify: `tui/dashboard.go` (model struct ~line 289; `reload()` ~line 318)

**Interfaces:**
- Consumes: `loadDesignReview` (Task 5.1).
- Produces: model fields `showDesignReview bool`, `designReview designReview`, `designReviewCur int`, `designReviewScroll int`.

- [ ] **Step 1: Add fields to `dashboardModel`.** After the "Manual launch modal" block (the `manualLaunchCopied bool` field, ~line 288), insert:

```go
	// Design Review overlay ([D] key) — approve wireframe/mockup for the selected story
	showDesignReview   bool
	designReview       designReview
	designReviewCur    int
	designReviewScroll int
```

- [ ] **Step 2: Refresh the open overlay on tick.** In `reload()`, after the `m.approvalPending = approvalPending()` line, add:

```go
	if m.showDesignReview && m.designReview.StoryID != "" {
		m.designReview = loadDesignReview(m.designReview.StoryID)
	}
```

- [ ] **Step 3: Compile-check.**

Run: `make build-tui`
Expected: `Built: ./maple`.

- [ ] **Step 4: Commit.**

```bash
git add tui/dashboard.go
git commit -m "dashboard: add design review overlay state + reload refresh"
```

### Task 5.3: Open key + overlay key handler

**Files:**
- Modify: `tui/dashboard.go` (`handleKey`: add overlay block before the global switch ~line 1170; add `D` to the global switch ~line 1183)

**Interfaces:**
- Consumes: `loadDesignReview`, `approveDesignArtifact`, `notifyAllPanesContinue` (existing), `m.stories`, `m.storiesCur`.

- [ ] **Step 1: Add the overlay handler.** Immediately BEFORE the `// Global keys` comment (~line 1170), insert:

```go
	// Design Review overlay
	if m.showDesignReview {
		arts := m.designReview.Artifacts
		switch k {
		case "j", "down":
			if m.designReviewCur < len(arts)-1 {
				m.designReviewCur++
				m.designReviewScroll = 0
			}
		case "k", "up":
			if m.designReviewCur > 0 {
				m.designReviewCur--
				m.designReviewScroll = 0
			}
		case "a":
			if m.designReviewCur < len(arts) {
				art := arts[m.designReviewCur]
				if art.Exists && (art.Kind == "wireframe" || art.Kind == "mockup") {
					if err := approveDesignArtifact(art.Path); err != nil {
						return m, m.setStatus("✗ approve: "+err.Error(), true)
					}
					m.designReview = loadDesignReview(m.designReview.StoryID)
					if m.designReviewAllApproved() && m.approvalPending != "" {
						_ = os.Remove(".claude/state/approval-pending.txt")
						m.approvalPending = ""
						n := notifyAllPanesContinue()
						if n > 0 {
							return m, m.setStatus(fmt.Sprintf("✓ %s approved — sent 'continue' to %d pane(s)", art.Kind, n), false)
						}
					}
					return m, m.setStatus("✓ "+art.Kind+" approved", false)
				}
			}
		case "q", "esc", "b", "ctrl+c":
			m.showDesignReview = false
		}
		return m, nil
	}

```

- [ ] **Step 2: Add the `D` open key.** In the global `switch k`, after the `case "?":` block (~line 1183), add:

```go
	case "D":
		if m.focus == paneStories && m.storiesCur < len(m.stories) {
			m.designReview = loadDesignReview(m.stories[m.storiesCur].id)
			m.designReviewCur = 0
			m.designReviewScroll = 0
			m.showDesignReview = true
		} else {
			return m, m.setStatus("press [s] to focus Stories, then [D] to review its design", false)
		}
```

- [ ] **Step 3: Add the `designReviewAllApproved` helper.** In `tui/design_review.go`, add:

```go
func (m *dashboardModel) designReviewAllApproved() bool {
	for _, a := range m.designReview.Artifacts {
		if (a.Kind == "wireframe" || a.Kind == "mockup") && a.Status != "approved" {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Compile-check.**

Run: `make build-tui`
Expected: `Built: ./maple`.

- [ ] **Step 5: Commit.**

```bash
git add tui/dashboard.go tui/design_review.go
git commit -m "dashboard: [D] opens design review, [a] approves selected artifact"
```

### Task 5.4: Render the overlay + View dispatch + footer/help

**Files:**
- Modify: `tui/dashboard_views.go` (add `designReviewView()`; add a help row)
- Modify: `tui/dashboard.go` (`View()` dispatch ~line 1613; `footer()` keys ~line 1742)

**Interfaces:**
- Consumes: `m.designReview`, `m.designReviewCur`, `m.designReviewScroll`, theme styles, `m.header()/m.footer()`.

- [ ] **Step 1: Add the render function.** Append to `tui/dashboard_views.go`:

```go
func (m *dashboardModel) designReviewView() string {
	t := m.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", 62))

	var rows []string
	rows = append(rows, titleStyle.Render("  Design Review — "+m.designReview.StoryID), sep)

	cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
	for i, a := range m.designReview.Artifacts {
		col := t.Foreground
		if a.Kind != "a11y" && a.Status == "approved" {
			col = t.Success
		}
		if a.Kind != "a11y" && a.Exists && a.Status != "approved" {
			col = t.Warning
		}
		line := lipgloss.NewStyle().Foreground(col).Render(a.label())
		if i == m.designReviewCur {
			rows = append(rows, "  "+cursor+" "+line)
		} else {
			rows = append(rows, "    "+line)
		}
	}
	rows = append(rows, "")

	// preview of the selected artifact
	if m.designReviewCur < len(m.designReview.Artifacts) {
		a := m.designReview.Artifacts[m.designReviewCur]
		rows = append(rows, titleStyle.Render("  "+a.Path), sep)
		if !a.Exists {
			rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render("  (not generated yet)"))
		} else if b, err := os.ReadFile(a.Path); err == nil {
			lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
			budget := m.height - 14
			start := m.designReviewScroll
			if start > len(lines) {
				start = len(lines)
			}
			for j := start; j < len(lines) && j < start+budget; j++ {
				rows = append(rows, lipgloss.NewStyle().Foreground(t.Foreground).Render("  "+lines[j]))
			}
		}
	}
	return strings.Join(rows, "\n")
}
```

- [ ] **Step 2: Add the View dispatch.** In `tui/dashboard.go` `View()`, after the `if m.showHelp { ... }` block (~line 1615), add:

```go
	if m.showDesignReview {
		return m.header() + m.designReviewView() + m.footer()
	}
```

- [ ] **Step 3: Add the footer hint.** In `footer()`, add a case to the `switch` (near `case m.showStory:`):

```go
	case m.showDesignReview:
		keys = "  [j/k] select · [a] approve wireframe/mockup · [Esc] close"
```

- [ ] **Step 4: Add a help-overlay row.** In `helpView()`'s `keyBindings`, after the `{"d", "toggle Design pane (full-screen)"}` row, add:

```go
		{"D", "Design Review — approve wireframe/mockup (Stories pane)"},
```

- [ ] **Step 5: Build via the dance + run the full Go test set.**

Run: `make build-tui && echo BUILT`
Expected: `Built: ./maple` then `BUILT`.

Run: `rm -f tui/template && cp -rL template tui/template; (cd tui && go test ./...); rm -rf tui/template && ln -s ../template tui/template`
Expected: `ok` for the `main` package (all tests pass).

- [ ] **Step 6: Run the CLI suite to confirm nothing regressed.**

Run: `make test`
Expected: the suite reports its pass summary with no failures.

- [ ] **Step 7: Commit.**

```bash
git add tui/dashboard_views.go tui/dashboard.go
git commit -m "dashboard: render design review overlay, wire view/footer/help"
```

---

## Phase 6 — Portal degrade to monospace for terminal artifacts

### Task 6.1: Portal renders `.md` / tui artifacts as monospace

**Files:**
- Modify: `template/scripts/design-review-portal.py` (the artifact-rendering branch that maps extensions to preview)
- Copy: `scripts/design-review-portal.py` (repo-root)

**Interfaces:**
- Consumes: existing portal artifact list + the extension→content-type logic (the `.html/.htm/.svg` iframe branch vs the `.json/.md/...` text branch).

- [ ] **Step 1: Make `.md` (and any artifact when target=tui) render as escaped monospace.** In the artifact preview branch that currently distinguishes `.html/.htm/.svg` (iframe) from text, ensure `.md`, `.txt`, and `.json` are wrapped in `<pre>` with HTML-escaped content rather than iframed. Add a small helper and use it in the text branch:

```python
def render_monospace(text: str) -> str:
    return '<pre class="mono">' + html.escape(text) + '</pre>'
```

Wire it so the text branch (the one already handling `.json`, `.md`, `.tsx`, `.css`, etc.) returns `render_monospace(read_text(path))` for `.md` and `.txt`, and pretty-printed JSON for `.json`. Do NOT change the `.html/.htm/.svg` iframe branch (web target keeps its preview).

- [ ] **Step 2: Add a `.mono` style.** In the portal's inline CSS (the `<style>` block), add:

```css
.mono { white-space: pre; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; line-height: 1.35; overflow-x: auto; }
```

- [ ] **Step 3: Syntax-check the Python.**

Run: `python3 -m py_compile template/scripts/design-review-portal.py && echo "py ok"`
Expected: `py ok`

- [ ] **Step 4: Smoke-test rendering of a `.md` artifact.**

Run: `python3 -c "import importlib.util,sys; spec=importlib.util.spec_from_file_location('p','template/scripts/design-review-portal.py'); m=importlib.util.module_from_spec(spec); spec.loader.exec_module(m); print(m.render_monospace('┌─ box ─┐\n<b>x</b>'))"`
Expected: a `<pre class="mono">` string with `&lt;b&gt;` escaped and the box-drawing chars intact.

- [ ] **Step 5: Copy to the repo-root dogfood script and confirm identical.**

Run: `cp template/scripts/design-review-portal.py scripts/design-review-portal.py && diff template/scripts/design-review-portal.py scripts/design-review-portal.py && echo "PARITY OK"`
Expected: `PARITY OK`

- [ ] **Step 6: Commit.**

```bash
git add template/scripts/design-review-portal.py scripts/design-review-portal.py
git commit -m "design portal: render md/tui artifacts as escaped monospace"
```

---

## Phase 7 — Docs + release

### Task 7.1: Document the multi-target design phase

**Files:**
- Modify: `template/docs/design/README.md`
- Modify: `template/CLAUDE.md`, `template/OPENCODE.md`, `template/CURSOR.md`, `template/COPILOT.md`, `template/AGENTS.md` (harness-doc parity — same edit in all five)
- Modify: repo-root `CLAUDE.md`

- [ ] **Step 1: Update `template/docs/design/README.md`.** Add a short section near the top:

```markdown
## Design target (web or TUI)

`design.target` in `project.config.yaml` (default `web`) selects the UI medium. The same design
agents run for any target and emit medium-appropriate artifacts at the same paths — see
[design-targets.md](./design-targets.md). For `tui` projects (terminal UIs), wireframes are ASCII
`.md` + `.excalidraw`, mockups are lipgloss-annotated `.md`, identity tokens are `terminal-theme.json`,
and accessibility is a terminal checklist written to the same `{id}.a11y.json` shape. Review and
approve in the maple `D` overlay or the browser portal.
```

- [ ] **Step 2: Add a one-line note to each harness root doc.** In each of `template/CLAUDE.md`, `template/OPENCODE.md`, `template/CURSOR.md`, `template/COPILOT.md`, `template/AGENTS.md`, near where the design phase is described, add:

```markdown
The design phase targets `design.target` (`web` | `tui`) from `project.config.yaml`; see `docs/design/design-targets.md`.
```

- [ ] **Step 3: Add the `D` overlay invariant to the repo-root `CLAUDE.md`.** Under "Architecture: Key Invariants", add a short subsection noting: the `D` Design Review overlay reads `docs/design/{wireframes,mockups}/{id}.*` for the focused story, `[a]` writes `status: approved` and (when no design approval remains) clears `approval-pending.txt` and notifies panes; it must honor the merge-not-overwrite rule for `maple.json` and never call `tea.Quit`.

- [ ] **Step 4: Verify the parity note is in all five harness docs.**

Run: `grep -l 'design.target' template/CLAUDE.md template/OPENCODE.md template/CURSOR.md template/COPILOT.md template/AGENTS.md`
Expected: all five paths.

- [ ] **Step 5: Commit.**

```bash
git add template/docs/design/README.md template/CLAUDE.md template/OPENCODE.md template/CURSOR.md template/COPILOT.md template/AGENTS.md CLAUDE.md
git commit -m "docs: document design.target and the D review overlay"
```

### Task 7.2: CHANGELOG + version bump

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add a changelog entry** at the top, under a new minor version heading (next minor after the current `v4.16.0` — i.e. `v4.17.0`):

```markdown
## v4.17.0

- design phase is now target-aware: `design.target: web | tui` in project.config.yaml
- tui target produces ASCII + excalidraw wireframes, lipgloss-annotated mockups,
  terminal-theme.json tokens, and a terminal a11y checklist (same a11y.json shape, gates unchanged)
- new in-TUI Design Review overlay ([D] on the Stories pane) approves wireframes/mockups
- design review portal renders terminal/markdown artifacts as monospace
```

- [ ] **Step 2: Verify the heading.**

Run: `head -8 CHANGELOG.md | grep 'v4.17.0'`
Expected: the `## v4.17.0` line.

- [ ] **Step 3: Commit.**

```bash
git add CHANGELOG.md
git commit -m "changelog: target-aware design phase + D review overlay (4.17.0)"
```

---

## Phase 8 — Dogfood: run a new overlay through the pipeline

> This phase runs MAPLE's own pipeline on the MAPLE repo. It is executed (not hand-coded): the
> orchestrator/agents produce the artifacts; the steps below are the operator checklist and the
> objective pass criteria. The proof feature is a small new overlay (suggested: an **SDLC gate-status
> overlay** showing pass/skip/fail of the sdlc gates for the current branch's stories). It must use a
> currently-unbound key — `G` is taken (go-to-bottom) and `d` is taken (Design pane), so pick e.g.
> a `:gates` command + a free capital, decided in the story's own design phase.

### Task 8.1: Refresh the dogfood instance from the rebuilt binary

**Files:**
- Regenerate: repo-root `.claude/`, `.opencode/`, `.cursor/`, `scripts/`, `docs/` from `template/`.

- [ ] **Step 1: Build the binary with the new template baked in.**

Run: `make build-tui`
Expected: `Built: ./maple`

- [ ] **Step 2: Re-sync the dogfood copies from the freshly built binary** (this is MAPLE initializing MAPLE). Use force so files that init skips when present get refreshed:

Run: `./maple init --force`
Expected: log lines showing `.claude/`, `.opencode/`, `.cursor/`, `scripts/`, `docs/` updated.

- [ ] **Step 3: Confirm the dogfood config still targets tui** (init must not clobber it):

Run: `grep -A1 'design:' project.config.yaml | grep 'target: tui'`
Expected: `  target: tui`. If missing, re-set it (Task 0.1 Step 4) and re-commit.

- [ ] **Step 4: Confirm the new agents/skills propagated into the dogfood `.claude`.**

Run: `grep -l '## Target awareness' .claude/agents/wireframe-architect.md .claude/skills/design-tokens/SKILL.md`
Expected: both paths.

- [ ] **Step 5: Commit the refreshed dogfood scaffold.**

```bash
git add .claude .opencode .cursor scripts docs project.config.yaml
git commit -m "dogfood: re-sync MAPLE's own .claude from updated template (target: tui)"
```

### Task 8.2: Drive the gate-status overlay story through all 8 phases

- [ ] **Step 1: Author the story.** Run the spec-kit path to write a Gherkin story to `docs/stories/` for the new overlay, `ui: true`. Concretely launch the pipeline (from the maple TUI `[x]` quick prompt or directly): `/pipeline-runner new-ui-feature <describe the gate-status overlay>`.

- [ ] **Step 2: Design phase produces terminal artifacts.** Confirm, after the design stages, that these exist for the story id `<ID>`:
  - `docs/design/wireframes/<ID>.wireframe.md` (ASCII) and `docs/design/wireframes/<ID>.wireframe.excalidraw`
  - `docs/design/mockups/<ID>.mockup.md` (lipgloss-annotated; NO `.tsx`)
  - `docs/design/identity/terminal-theme.json`

Run: `ls docs/design/wireframes/<ID>.wireframe.* docs/design/mockups/<ID>.mockup.md docs/design/identity/terminal-theme.json`
Expected: all listed, and no `<ID>.wireframe.html` / `<ID>.mockup.tsx`.

- [ ] **Step 3: Approve in the `D` overlay.** Open maple, focus the Stories pane (`s`), select the story, press `D`, review wireframe + mockup, press `a` on each. Confirm both show `[APPROVED]` and the design gate passes:

Run: `bash scripts/sdlc/design-approved-gate.sh docs/stories/<ID>*/Story.md 2>/dev/null || bash scripts/sdlc/design-approved-gate.sh docs/stories/<ID>.md`
Expected: `[design-gate] OK`.

- [ ] **Step 4: Implement + validate.** Let the pipeline implement the overlay in `tui/` (TDD), then confirm a11y:

Run: `cat docs/design/mockups/<ID>.a11y.json | python3 -c "import json,sys; d=json.load(sys.stdin); print(sum(1 for v in d['violations'] if v['impact'] in ('critical','serious')), 'critical/serious')"`
Expected: `0 critical/serious`.

- [ ] **Step 5: Build + full suite green.**

Run: `make build-tui && make test`
Expected: `Built: ./maple` and the CLI suite passes.

- [ ] **Step 6: Confirm the overlay works in the binary.** Launch `./maple`, exercise the new overlay key, confirm it renders the gate status.

- [ ] **Step 7: Commit the dogfood feature.**

```bash
git add tui docs/stories docs/design CHANGELOG.md
git commit -m "add sdlc gate-status overlay (built via MAPLE pipeline on MAPLE)"
```

- [ ] **Step 8: Record the dogfood outcome.** Note in the PR/commit body that the design phase ran in `tui` mode end-to-end, produced terminal artifacts, was approved in the `D` overlay, and all gates passed — i.e. MAPLE ran its own pipeline on itself.

---

## Phase 9 — Fresh-project init regression (a new project still works)

> Proves the template changes didn't break the normal `maple init` flow for a brand-new project,
> tested the usual way in `/tmp/maple_testing_test`. A fresh project must default to `design.target: web`
> and behave exactly as before — the tui path is opt-in.

### Task 9.1: Build, init a fresh project, verify web defaults

- [ ] **Step 1: Build the binary with the new template.** Run: `make build-tui` — Expected: `Built: ./maple`.

- [ ] **Step 2: Create a fresh project and init it.**

```bash
rm -rf /tmp/maple_testing_test && mkdir -p /tmp/maple_testing_test
( cd /tmp/maple_testing_test && CI=1 /Users/kinncj/Development/kinncj/MAPLE/maple init )
```
Expected: init logs copying `.claude/`, `.opencode/`, `.cursor/`, `scripts/`, `docs/` and writing `project.config.yaml`.

- [ ] **Step 3: Verify the fresh project defaults to web and has the new pieces.**

```bash
cd /tmp/maple_testing_test
grep -A1 '^design:' project.config.yaml | grep 'target: web'
grep -l '## Target awareness' .claude/agents/wireframe-architect.md
test -f docs/design/design-targets.md && echo "profile present"
ls scripts/sdlc/design-approved-gate.sh scripts/sdlc/a11y-gate.sh >/dev/null && echo "gates present"
```
Expected: `  target: web`, the agent path, `profile present`, `gates present`.

- [ ] **Step 4: Verify the web design path is intact** (wireframe-architect still lists `.html`/`.excalidraw` for web).

Run: `grep -A6 '## Target awareness' /tmp/maple_testing_test/.claude/agents/wireframe-architect.md | grep -iE 'web.*html|html.*excalidraw'`
Expected: a line describing the web outputs (`.md` + `.html` + `.excalidraw`).

- [ ] **Step 5: Run the gate fixture against the fresh project's gate scripts.**

Run: `REPO_ROOT=/tmp/maple_testing_test bash /Users/kinncj/Development/kinncj/MAPLE/tests/cli/test_tui_design_gates.sh`
Expected: `PASS tui design gates`.

(No commit — throwaway verification project outside the repo.)

---

## Self-Review

**Spec coverage** (against `docs/superpowers/specs/2026-06-25-tui-design-target-design.md`):
- Config + schema → Task 0.1 ✓
- Target profile → Task 0.2 ✓
- Target-aware agents (wireframe/mockup/a11y/design-system/visual-identity) → Tasks 1.1–1.5 ✓
- Target-aware skills (wireframe/mockup/a11y-audit/design-tokens/visual-identity) → Tasks 2.1–2.5 ✓
- Orchestrator + product-owner + taffy neutralized → Tasks 3.1–3.3 ✓
- Gates unchanged, proven → Task 4.1 ✓
- In-TUI `D` review overlay → Tasks 5.1–5.4 ✓
- Portal degrade → Task 6.1 ✓
- Docs (incl. harness parity for 5 root docs) + version bump → Tasks 7.1–7.2 ✓
- Dogfood a new overlay end-to-end → Tasks 8.1–8.2 ✓
- Fresh project init still works, defaults to web, tested in /tmp/maple_testing_test → Phase 9 ✓
- Fresh-project config gets design.target from the Go generator → Task 0.3 ✓
- `tui` wireframe = `.md` + `.excalidraw`, no `.html` → encoded in Tasks 0.2, 1.1, 2.1, 8.2 ✓
- Both review surfaces (overlay + portal) → Phase 5 + Phase 6 ✓

**Type/name consistency:** `designArtifact{Kind,Path,Status,Summary,Exists}`, `designReview{StoryID,Artifacts}`, `loadDesignReview`, `parseArtifactStatus`, `a11ySummary`, `approveDesignArtifact`, `designReviewAllApproved`, and model fields `showDesignReview/designReview/designReviewCur/designReviewScroll` are defined in Tasks 5.1–5.2 and used consistently in 5.3–5.4. Footer/View/help reference the same `showDesignReview`.

**Known follow-ups (out of this plan's scope):** the dogfood overlay's exact key binding is decided in its own design phase (Phase 8 note); `component-scaffold` skill is skipped for tui (orchestrator step 6) rather than given a tui variant — acceptable since tui components are scaffolded directly in `tui/` by TDD.
