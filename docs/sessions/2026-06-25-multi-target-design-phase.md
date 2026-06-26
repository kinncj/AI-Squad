# Session — 2026-06-25

Theme: "make MAPLE run on MAPLE." Built a target-aware (web | tui) design phase into the product,
proved it by running MAPLE's own pipeline on the MAPLE repo to build a real TUI feature, and shipped
**v4.17.0**.

## What was done

- **Multi-target design phase (product feature).** Added `design.target: web | tui` to
  `project.config.yaml` + schema + the Go `init` generator (fresh projects default to `web`).
  Made the existing design agents and skills medium-aware via a single `docs/design/design-targets.md`
  profile — no parallel `tui-*` agents. For `tui`: ASCII + Excalidraw wireframes, lipgloss-annotated
  `mockup.md`, `terminal-theme.json` tokens, and a terminal a11y checklist written to the **same**
  `a11y.json` shape, so the SDLC gates are unchanged. Edits propagated per-harness across
  `.claude`/`.opencode`/`.cursor`.
- **`D` Design Review overlay** (Go, TDD) — review/approve a story's wireframe/mockup inline in the
  maple dashboard; `[a]` writes `status: approved` and resumes the pipeline.
- **Portal monospace degrade** — the design review portal renders box-drawing `.md` artifacts as raw
  monospace instead of mangling them through the markdown renderer.
- **Dogfood: `C` Git Changes overlay.** Ran `/pipeline-runner new-ui-feature` on the MAPLE repo
  (`design.target: tui`) to build a working-tree diff popup end-to-end: Gherkin story →
  ASCII+Excalidraw wireframe → terminal identity/tokens → lipgloss mockup → TDD implementation
  (`tui/git_changes.go` + dashboard wiring) → terminal a11y. Every SDLC gate green.
- **Fresh-project regression** verified in `/tmp/maple_testing_test` — new `maple init` still works and
  defaults to `web`, web design path intact.
- **Shipped v4.17.0** — merged to `main`, CI green, `gh release create` cross-compiled all 5 platforms.
  https://github.com/kinncj/MAPLE/releases/tag/v4.17.0

## Decisions made

- **One medium-aware agent set, not parallel `tui-*` agents** (user steer). A `design.target` profile
  is the single source of truth; adding a future target is a profile row, not a new pipeline.
- **Gates stay unchanged.** TUI artifacts use the same paths + a11y JSON shape, so
  `design-approved-gate.sh` / `a11y-gate.sh` needed zero edits — confirmed with a fixture test.
- **`tui` wireframes keep `.excalidraw`** (user steer) — it's an editable diagram, not a web page;
  only `.html` is dropped for tui.
- **Removed `seed-test` from the template entirely** (user call) — MAPLE has no test DB. Rippled
  through Makefile, qa/tdd/validate docs (×3 mirrors), pipeline.md, CI structure test, and ARTICLE.md.
- **`make build-tui` is the build command** (user preference) — not the quick `go build`.

## Fixes applied

- **Harness-mirror clobber** (caught by user): Phase 1–3 `cp .claude → .opencode/.cursor` was unsafe
  for files with harness-specific content. Audit showed only `orchestrator.md` was affected (its
  `.opencode` frontmatter `mode:`/`permission:` got overwritten). Restored from `main` and re-applied
  the edit per-harness. All other propagated files were genuinely byte-identical.
- **shellcheck SC2164** in `tests/cli/test_tui_design_gates.sh` (`cd "$TMP"` → `cd "$TMP" || exit 1`)
  — caught by CI before the release tag.
- **template/CLAUDE.md (+ harness mirrors) still hard-coded the web-only wireframe rule** ("all three
  files required") — made target-aware.

## Unfinished / follow-up

1. **Approval auto-continue bug** (diagnosed, not fixed). The file signal works (TUI `[a]` and portal
   both delete `approval-pending.txt`), but a chat-harness agent only resumes when it gets a turn — the
   "continue" keystroke (`notifyAllPanesContinue` / `notify_continue`) only reaches a maple-**spawned**
   pane recorded in `panes.json`; on a direct `claude` launch (case 4 in `spawnWithPane`) the paneRef is
   empty and the keystroke silently no-ops, while the TUI still says "approved — pipeline resuming."
   Fix: (a) `[a]`/portal should report honestly when 0 panes were notified, and (b) the pipeline agent
   should reliably poll the file so deletion alone resumes it. Good `/bugfix` dogfood candidate.
2. **`pipeline-runner` skill's wireframe gate** still hard-requires `.html` (Step 4) — predates the tui
   work; should be made `tui`-aware. The authoritative `design-approved-gate.sh` only checks `.md`, so
   it's cosmetic for now.

## Commits (this session, 27 on `main`, a4d7f5d → 6b6ad33)

- `add design.target config (web|tui), default web`
- `init: generate design.target: web in new project config`
- `add design-targets profile doc (web vs tui artifacts)`
- `wireframe-architect` / `ui-mockup-builder` / `a11y-auditor` / `design-system-author` /
  `visual-identity-designer` / `product-owner` — tui target awareness
- `wireframe` / `mockup` / `a11y-audit` / `design-tokens` / `visual-identity` skills — tui target
- `orchestrator: design gate reads design.target` · `new-ui-feature: medium-neutral stage descriptions`
- `test: tui artifacts pass design+a11y gates, critical still blocks`
- `add design review readers …` · `dashboard: add [D] design review overlay …`
- `design portal: render box-drawing md artifacts as raw monospace`
- `docs: document design.target and the D review overlay` · `changelog: … (4.17.0)`
- `add MAPLE-on-MAPLE dogfood harness (tui-aware) + spec and plan`
- `remove seed-test from template, fix orchestrator mirror paths` ·
  `scrub seed-test from ARTICLE.md, fix target count 13->12`
- `add git-changes overlay (C), built via MAPLE's own tui pipeline`
- `fix shellcheck SC2164 in tui design gates test (cd || exit)`

Release: **v4.17.0** (5-platform binaries built).
