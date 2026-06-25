# Design — Multi-target design phase for MAPLE's SDLC pipeline (and dogfood on MAPLE)

Date: 2026-06-25
Status: draft (awaiting user review)
Topic: make MAPLE's design/UX phase aware of the UI medium (web, TUI, …), then run a real MAPLE feature through the pipeline to prove MAPLE can run on MAPLE.

---

## Why

MAPLE ships an 8-phase SDLC pipeline as template files and installs it into user projects with `maple init`. One phase is design/UX. That phase is hard-wired for **web**: the design agents force `.html`/`.excalidraw` wireframes, build React/Tailwind/Mantine mockups, run axe-core/pa11y against a browser URL, and emit CSS/Tailwind/Mantine tokens. The gates and the review portal assume the same.

MAPLE itself is a Bubble Tea **TUI**. So when MAPLE runs its own pipeline on its own repo, the design phase has no medium to target — it can only produce web artifacts for a thing that has no web surface. The user's framing: the design review is not building web pages or HTML mockups anymore — it's a TUI — but we still want wireframes and the rest of the design artifacts, just terminal-native.

The fix is not a parallel TUI-only design pipeline. It is to make the design agents **medium-aware**: one competent set of designers that reads the project's `design.target` and produces the right artifacts for that surface — web today, TUI today, other surfaces later. MAPLE then dogfoods it: a new small overlay goes through all 8 phases, design phase included.

## Goal

1. Add `design.target` (`web` | `tui`, default `web`) to the product config.
2. Make the existing design agents and skills target-aware via a small, extensible **target profile**, so each one emits artifacts appropriate to the medium.
3. Keep the SDLC gates unchanged — TUI artifacts land at the same paths and use the same JSON shape the gates already check.
4. Make design review work on both surfaces: an in-TUI approval overlay in the maple dashboard, and the existing browser portal degraded to render terminal artifacts as monospace.
5. Dogfood: build a new small maple overlay by running it through the pipeline end to end, with the design phase producing terminal wireframe/mockup/a11y artifacts approved through the new overlay.

## Non-goals (YAGNI)

- Auto-detecting the target from repo contents. The target is explicit config.
- Compiling design tokens into the `maple` binary. The token emit is a reference theme, not wired into the Go build.
- Removing or rewriting the web design path. Web stays a first-class target.
- TUI analogs of Storybook / visual-regression tooling.
- New design targets beyond `web` and `tui` in this change. The profile is built so adding one later is cheap, but only the two are implemented now.

---

## Approach: target-aware agents, not parallel agents

The earlier draft created dedicated `tui-*` agents and a `new-tui-feature.yaml` workflow alongside the web ones. The chosen approach is the opposite: one set of design agents that each understand multiple media. Reasons:

- A designer who can wireframe a terminal screen and a web page is one role, not two. Splitting them duplicates the pipeline and would force a third copy for the next medium ("etc.").
- It collapses the orchestrator routing fork — there is one design sub-pipeline; the agents adapt.
- It localizes the medium knowledge in a single profile the agents, skills, and review surfaces all read.

### The target profile

A lightweight profile enumerates, per target, what the design phase produces and how it is reviewed. It is data, not code — adding a target is adding a row.

| Field | `web` | `tui` |
|-------|-------|-------|
| wireframe format | ASCII `.md` + `.html` preview + `.excalidraw` | ASCII/box-drawing `.md` + `.excalidraw` (editable diagram); no `.html` |
| mockup format | React/Tailwind/Mantine `.tsx` | terminal mock `.md` with lipgloss style annotations |
| token emit | CSS vars, Tailwind config, Mantine theme | lipgloss/ANSI terminal theme |
| a11y method | axe-core/pa11y against a preview URL | terminal a11y checklist (keyboard reachability, focus visibility, fg/bg WCAG contrast, no color-only signaling, `NO_COLOR`, min-width/resize/truncation) |
| artifact paths | `docs/design/wireframes/{id}.wireframe.md`, `docs/design/mockups/{id}.mockup.md`, `docs/design/mockups/{id}.a11y.json` | identical paths |
| review render | browser portal (HTML/SVG/TSX preview) | ASCII inline in the maple TUI overlay + monospace `<pre>` in portal; `.excalidraw` opens in the portal |

The artifact **paths** and the a11y **JSON schema** are identical across targets on purpose — that is what lets the gates stay untouched (see Gates).

Profile location: a single reference both the agents and the review surfaces cite, e.g. `template/docs/design/design-targets.md` (human-readable table) backed by the `design.target` value in `project.config.yaml`. No heavy abstraction — the agents read the target string and follow the matching column.

---

## Components and concrete changes

Everything under `template/` is the product source of truth and must be applied identically across the three harness mirrors: `template/.claude/`, `template/.opencode/`, `template/.cursor/`. The repo-root copies (`.claude/`, `.opencode/`, `.cursor/`, `scripts/`, `docs/`, `project.config.yaml`) are the dogfood instance, regenerated from the rebuilt binary via `maple init`.

### 1. Config + schema
- `template/project.config.yaml`: add `design.target: web` (default; preserves current behavior for existing users). Keep `ui_library` / `token_format` — they apply only when `target: web`.
- `template/.claude/schemas/project.config.schema.json`: add `design.target` with enum `["web", "tui"]`, default `"web"`.
- Repo-root `project.config.yaml`: set `design.target: tui`.

### 2. Target-aware design agents (edit, ×3 mirrors)
Each gains a short "Target awareness" section: read `design.target`, follow the matching profile column, and emit those artifacts only.
- `wireframe-architect` — for `tui` emit ASCII `.md` + `.excalidraw` (editable diagram of the terminal layout: panes/overlays as rectangles, labels, state-transition arrows) and drop the `.html` browser preview; web keeps all three.
- `ui-mockup-builder` — for `tui` produce a terminal mock `.md` with lipgloss style annotations instead of `.tsx`.
- `a11y-auditor` — for `tui` run the terminal a11y checklist and write the same `a11y.json` shape; no browser/axe-core.
- `design-system-author` — for `tui` emit a lipgloss/ANSI theme instead of CSS/Tailwind/Mantine.
- `visual-identity-designer` — for `tui` constrain palette to terminal-safe colors with contrast targets.
- `ux-researcher` — unchanged (medium-agnostic).

### 3. Target-aware design skills (edit, ×3 mirrors)
- `wireframe`, `mockup`, `a11y-audit` — branch output on the profile.
- `design-tokens` — add a `terminal` emit target alongside the existing CSS/Tailwind/Mantine emitters; `tokens.json` (DTCG) stays canonical with terminal color values.
- `visual-identity` — terminal-palette guidance.

### 4. Taffy workflow (edit, ×3 mirrors)
- `new-ui-feature.yaml`: keep one workflow; rewrite stage descriptions to be medium-neutral (e.g. "wireframes for every screen state" rather than "ASCII/SVG wireframes"; "design tokens for the target medium" rather than "CSS vars, Tailwind config, Mantine theme"). Stages still gate on `ui:true`; the agents adapt to `design.target`.

### 5. Orchestration (edit, ×3 mirrors)
- `orchestrator.md` §"Design Gate (ui: true stories)": make medium-neutral and reference `design.target`; no web/tui fork.
- `product-owner.md`: clarify `ui:true` means the user sees or interacts with a rendered surface — web or terminal — and the medium comes from `design.target`, not the story.

### 6. Gates — no change (verified)
- `scripts/sdlc/design-approved-gate.sh` already checks `docs/design/wireframes/{id}.wireframe.md` and `docs/design/mockups/{id}.mockup.md` for `status: approved`. No extension assumptions.
- `scripts/sdlc/a11y-gate.sh` already reads `docs/design/mockups/{id}.a11y.json` and counts `violations[].impact in (critical, serious)`. Producer-agnostic.
- TUI artifacts satisfy both as long as they keep these paths and the JSON shape — which the profile guarantees. The only task here is to confirm nothing else in the gate chain hard-requires a web artifact.

### 7. Design review — both surfaces
- **In-TUI overlay** (new Go in `tui/`): a Design Review overlay (proposed key `D`) that, for the story awaiting design approval, renders its wireframe `.md` + mockup `.md` + a11y `.json` summary inline and approves with a keystroke — writing `status: approved` into the artifact frontmatter and clearing `.claude/state/approval-pending.txt`. Follows the existing `showHelp` overlay pattern. Touches `dashboard.go` (state field, key handler, reload of design artifacts), `dashboard_views.go` (render function), `loaders.go` (read the artifacts), with Go tests. Respects the "harness launching never calls tea.Quit" and "maple.json is merge-not-overwrite" invariants.
- **Portal degrade** (`scripts/design-review-portal.py`): when the artifact is `.md` (or `design.target: tui`), render it as monospace `<pre>` instead of trying to iframe HTML; keep the approve/upload flow. Web target rendering is unchanged.

### 8. Docs + release
- `template/docs/design/README.md`: document the multi-target design path and the profile table.
- `template/docs/design/design-targets.md`: the profile reference.
- Harness root docs that describe the design phase — `template/CLAUDE.md`, `template/OPENCODE.md`, `template/CURSOR.md`, `template/COPILOT.md`, `template/AGENTS.md` — updated together (harness-doc parity).
- Repo `CLAUDE.md` (maple-repo instructions): record the new `D` overlay and the "harness launching never calls tea.Quit" / merge-not-overwrite invariants it must honor.
- `CHANGELOG.md` + a **minor** version bump (new feature / new overlay).

### 9. Dogfood proof
A new small overlay built by running it through all 8 phases. Proposed: an **SDLC gate-status overlay** (proposed key `G`) that shows pass / skip / fail of the sdlc gates for the current branch's stories — useful for running MAPLE on MAPLE, and a clean greenfield terminal surface to wireframe. Swappable for another small overlay.

Path through the pipeline:
- spec-kit → a Gherkin story in `docs/stories/` (`ui: true`, repo target = `tui`).
- design sub-pipeline → `wireframe-architect` (ASCII `.md` + `.excalidraw`), `visual-identity` / `design-tokens` (terminal), `ui-mockup-builder` (terminal mock `.md`), approved through the new `D` overlay.
- architecture → ADR if the change warrants one.
- implement → TDD in `tui/` (`dashboard.go`, `dashboard_views.go`, `loaders.go`).
- validate → `a11y-auditor` writes `a11y.json` (terminal checklist), QA suite green.
- karpathy audit → humanizer → ship, all gates green, `maple.json` state visible in the TUI throughout.

---

## Build & test discipline

- Any change under `tui/`: `make build-tui` (handles the embed template dance), plus `cd tui && go build -o /tmp/maple_test .` as a quick compile check.
- Go tests need the same template dance as the build (the `go:embed` symlink breaks a bare `go test` in `tui/`).
- Run the full CLI test suite (218 tests) before committing.
- Validate the gates by running `scripts/sdlc/*.sh` against the dogfood story file directly.
- CI `maple init` smoke test must still pass (init copies the template intact).

## Risks / open questions

- **Prompt length.** Folding web + tui into each agent lengthens its prompt. Mitigated by pushing the medium specifics into the profile table the agent references, rather than inlining two full output specs.
- **Key binding collisions.** `D` and `G` are proposals; confirm against the current keymap in `dashboard_views.go` (the help overlay already lists bindings) before wiring.
- **Mirror drift.** Three harness copies of every agent/skill/taffy must stay identical. The plan should treat "apply to all three" as one atomic step per file, not three independent edits.
- **Dogfood refresh.** After editing `template/`, the repo-root dogfood copies must be regenerated (`maple init`, possibly `--force` for files init skips when present) so the dogfood run exercises the new template, not the stale copy.

## Definition of done

- `design.target` exists in config + schema; default `web` keeps existing users unchanged.
- The five design agents and five skills each produce correct artifacts for `web` and `tui`, selected by the profile, across all three harness mirrors.
- `new-ui-feature.yaml`, `orchestrator.md`, `product-owner.md` are medium-neutral.
- Gates pass on TUI artifacts with no gate-script changes.
- The maple TUI has a working `D` design-review overlay; the portal renders terminal artifacts as monospace.
- A new overlay has gone through all 8 phases on the MAPLE repo with the design phase producing approved terminal artifacts, all gates green, binary builds, full test suite passes.
