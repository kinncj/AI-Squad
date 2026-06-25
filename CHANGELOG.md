# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Conventional Commits](https://www.conventionalcommits.org/).

<!-- Agents append entries here using: gh issue comment + docs agent -->

## [4.17.0] — 2026-06-25

### Added
- **Target-aware design phase**: `design.target: web | tui` in `project.config.yaml` (default `web`). The design agents and skills (wireframe-architect, ui-mockup-builder, a11y-auditor, design-system-author, visual-identity-designer + their skills) now read the target and emit medium-appropriate artifacts at the same paths, so the SDLC gates are unchanged. For `tui`: ASCII + Excalidraw wireframes, lipgloss-annotated `.mockup.md`, `terminal-theme.json` tokens, and a terminal a11y checklist written to the same `a11y.json` shape. New profile doc `docs/design/design-targets.md`.
- **`D` Design Review overlay**: review and approve a story's wireframe/mockup inline in the maple dashboard (Stories pane → `D`), without a browser. `[a]` writes `status: approved` and resumes the pipeline.
- **Portal terminal rendering**: the design review portal renders box-drawing/terminal `.md` artifacts as raw monospace instead of mangling them through the markdown renderer.

### Changed
- `maple init` writes `design.target: web` into new projects' `project.config.yaml`; web behavior is unchanged.

## [4.16.0] — 2026-06-25

### Added
- **`RATE_LIMITED` pipeline state**: rate-limited TAFFY runs are now a real, recoverable state instead of a documented-only one. When a usage limit / 429 / "resets at" message stops a run, the agent writes `status=RATE_LIMITED` with `resume_at` and `rate_limit_reason` to `maple.json` plus a `.claude/state/rate-limit.txt` breadcrumb. The `[P]` pipeline overlay shows a yellow `⚑` flag with a live reset countdown.
- **Rate-limit resume (hybrid)**: `[r]` resumes the paused pipeline on its recorded harness's pinned session with a continue-from-stage prompt; `[A]` arms opt-in auto-resume that relaunches automatically when the window clears. A thrash guard disarms auto-resume if a resumed run re-hits the limit within 5 minutes. State persists on disk, so a run rate-limited today resumes the next day.
- **Stale-run reclassify safety net**: the TUI promotes a stale `RUNNING` pipeline to `RATE_LIMITED` from the breadcrumb file (consumed like `approval-pending.txt`), and `[m]` manually marks a stale run as rate-limited — covering hard cutoffs the agent cannot self-report.
- **Session-start protocol parity**: `OPENCODE.md` and `CURSOR.md` gain the mandatory session-start status check (previously only in `CLAUDE.md`/`COPILOT.md`); all four now treat `RATE_LIMITED` as an active pipeline to resume rather than bypass.

### Changed
- Header taffy badge and pipeline overlay count `RATE_LIMITED` as an in-flight pipeline.
- `pipeline-runner` skill documents rate-limit handling, the new `maple.json` fields (`resume_at`, `rate_limit_reason`, `harness`, `auto_resume`), and the `rate-limit.txt` signal file.
- README FAFE rows rewritten to match shipped behavior (was aspirational).

## [4.13.0] — 2026-05-20

### Added
- **`implement-stories` design gates**: `implement-stories.yaml` now includes `wireframe → visual-identity → design-tokens → ui-mockup-builder` stages (all `when: ui:true`, human-approval gates) before `standard-8-phase`, matching `new-ui-feature.yaml`. Synced to `.opencode` and `.cursor` mirrors. Fixes silent skip of design review for `ui: true` stories run via MAPLE req handoff.
- **Pipeline write guard (Claude Code)**: new `PreToolUse[Write/Edit]` hook (`pre-write.sh`) hard-blocks AI writes to `app/` or `tests/` when no MAPLE pipeline is active. Surfaces the correct `/pipeline-runner` command to start.
- **Session Start Protocol**: all harness root markdowns (`CLAUDE.md`, `COPILOT.md`, `copilot-instructions.md`) now open with an explicit mandatory pipeline-check step, making the orchestrator-first contract visible at the top of every session.
- **Cursor pipeline rules**: two new rules in `.cursor/rules/` — `pipeline-gate.mdc` (`alwaysApply: true`, injected into every conversation) and `impl-files.mdc` (glob-scoped to `app/**` + `tests/**`, fires when those files enter context).
- **OpenCode orchestrator agent**: `orchestrator.md` gains `mode: primary` and `permission: edit: deny`, making it a selectable primary agent in the Tab-cycle and enforcing at the runtime level that it never writes files.
- **Design portal dynamic ports**: portal no longer binds to fixed port 4173. Uses `MAPLE_DESIGN_PORT` if set; otherwise asks the OS for a free port. Chosen port written to `.claude/state/design-review-portal.port`.
- **Design portal stop on maple quit**: `runDashboard` calls `design-review-portal.sh stop` when the TUI exits, so the portal process does not outlive the maple session.
- **Portal URL in status bar**: the URL (with actual port) is now shown in the status bar when the portal starts or opens.

### Changed
- `schema.yaml`: skipped stages (due to `when:` guard) are now explicitly documented as dependency-satisfied for `depends_on` resolution.

## [4.12.15] — 2026-05-19

### Fixed
- **Portal approval resume signal**: design review portal approval actions now send `continue` to recorded tmux/zellij panes from `.claude/state/panes.json`, so approving in the web portal resumes paused TAFFY runs waiting on terminal input.
- **Approval state fallback handling**: portal approval actions continue to resolve pending stage via `awaiting_approval` when `approval-pending.txt` is missing, and clear paused state fields consistently.

## [4.12.14] — 2026-05-19

### Fixed
- **Design Review Portal artifact scope**: review artifact discovery now targets review-centric paths (`docs/design`, `docs/stories`, `docs/specs`, `artifacts`, `screenshots`, `previews`) and reviewable file types only, avoiding broad codebase listings.
- **Markdown + Mermaid rendering**: markdown previews now support Mermaid fenced code blocks (```` ```mermaid ````) and render diagrams inline in both preview pane and modal.
- **Portal page stability**: corrected script-template escaping issues that could break frontend execution and cause empty artifact views.

## [4.12.13] — 2026-05-19

### Fixed
- **Design Review Portal artifact completeness**: declared design artifacts now resolve absolute paths to repo-relative entries and are prioritized in merge output so required approval files (for example nested `docs/design/**/wireframes.html`) reliably appear in the portal.
- **Design review feedback UX**: Request Changes / Reject / Approve now clear feedback and upload inputs after submit, and feedback auto-hydration is limited to active pending-approval states to avoid stale text repopulation.
- **Modal rendering parity**: artifact modal rendering now consistently handles Excalidraw scene previews, markdown, HTML/SVG previews, and uploaded image formats (`.jpg/.jpeg/.png/.gif/.webp`).

### Changed
- **UI-gate policy behavior**: `new-ui-feature` design stages are `when: ui:true` by default across Claude/OpenCode/Cursor, with MAPLE `[x]` quick-launch override via `force_ui=true` and `launch_source=maple-x`.
- **Terminology clarification**: `spec-kit` now explicitly means MAPLE internal **Specification Knowledge & Integration Toolkit** in workflows, pipeline-runner docs, TUI prompt context, and user-facing docs.

## [4.12.12] — 2026-05-19

### Fixed
- **`new-ui-feature` design phases no longer skip**: wireframe, visual-identity, design-tokens, ui-mockup-builder, and a11y-audit TAFFY stages now run with `when: always` across Claude/OpenCode/Cursor templates so review gates trigger consistently.
- **Design review artifact visibility and contract hardening**: pipeline-runner contracts now require reviewable design artifacts and explicit artifact-manifest updates during UI approval flows, and the portal surfaces those artifacts for approve/reject/request-changes review loops.

## [4.12.11] — 2026-05-19

### Added
- **Copilot harness governance contract**: added `COPILOT.md` as a first-class runtime instruction file for MAPLE launches, including pipeline-runner behavior, heartbeat requirements, BusinessRepo boundaries, and approval-loop handling.

### Changed
- **Governance bootstrap harness mapping**: Copilot launches now include `COPILOT.md` in mandatory instruction bootstrap alongside shared policy files.
- **Pipeline-runner harness docs parity**: pipeline-runner skill templates (Claude/OpenCode/Cursor) now explicitly list Copilot root markdown support in runtime policy enforcement.
- **Req handoff instruction wording**: implementation handoff prompts now reference harness-specific root markdown (`CLAUDE.md` / `OPENCODE.md` / `CURSOR.md` / `COPILOT.md`) instead of a Claude-only root file.

## [4.12.10] — 2026-05-19

### Fixed
- **Quick prompt paste support**: `[x]` quick-prompt and related text-input overlays now correctly accept pasted clipboard content by handling multi-rune key events instead of only single-character keystrokes.

## [4.12.9] — 2026-05-19

### Added
- **Design review uploads + decisions**: portal now supports review-file uploads (including `.excalidraw`, images, HTML/SVG, and text), explicit **Approve / Reject / Request changes** actions, and attachment propagation to `.claude/state/design-feedback.json`.
- **Approval feedback handoff**: pipeline-runner templates (Claude/OpenCode/Cursor) now require processing `design-feedback.json` attachment inputs during paused human-approval stages before advancing.

### Fixed
- **Portal token resilience**: design review portal now recovers from stale/missing token state by minting a token when absent and retrying client requests after token refresh.
- **In-portal preview UX**: artifact links now open in a modal (not a new tab), with richer platform-aware previews including markdown rendering, video previews, and Excalidraw-aware handling.

## [4.12.8] — 2026-05-19

### Fixed
- **Implement-stories execution contract**: added a mandatory `qa-cucumber` stage to `implement-stories` TAFFY workflows (Claude/OpenCode/Cursor) so story implementation runs must materialize and validate Gherkin/Cucumber outputs before final UI audit.
- **Pipeline completion gating**: strengthened `pipeline-runner` requirements to fail when implementation runs do not produce required `/tests` and `/tests/features` artifacts, instead of reporting success with incomplete outputs.
- **Language-agnostic boundary enforcement**: clarified governance rules that runtime/test code may copy/adapt artifact content from docs, but must never import from `docs/`, `.github/`, or `.claude/` paths; violating runs must fail with offending paths.

## [4.12.7] — 2026-05-19

### Fixed
- **TAFFY harness permission mode**: TAFFY launches now run in permissive mode per harness where supported so orchestrated phases do not stall on interactive permission prompts. Claude uses `--dangerously-skip-permissions`, Copilot uses `--allow-all`, and Cursor uses `--yolo --sandbox disabled --approve-mcps`. Non-TAFFY launches are unchanged.
- **Implement-stories TAFFY handoff mode**: `maple req` implementation handoff now uses the same TAFFY permissive launch path for consistent runtime behavior across quick-launch and req flows.

## [4.10.2] — 2026-04-22

### Fixed
- **Cross-platform `rtk` install fallback**: the v4.10.1 fallback only covered linux/macos. Both `scripts/install.ps1` and `maple init` now also try `cargo install --git https://github.com/rtk-ai/rtk` when cargo is on PATH — rtk-ai's officially documented Windows install method, and a universal escape hatch when binary downloads are blocked (proxies, corp firewalls). `findInstalledRTK` is now Windows-aware, searching `%USERPROFILE%\.cargo\bin` on Windows vs. `~/.cargo/bin`, `~/.local/bin`, `/usr/local/bin` on Unix.

## [4.10.1] — 2026-04-22

### Added
- **Copilot sessions in the Agents pane**: `maple` now discovers GitHub Copilot CLI sessions from `~/.copilot/session-state/*`, filtering by matching `cwd` in each session's `workspace.yaml`. Shown with a `[gh]` badge alongside Claude (`[cc]`) and OpenCode (`[oc]`) rows. The session detail popup parses `events.jsonl` for user/assistant messages, tool calls, and model changes. `o` (open) and the `L` launcher resume Copilot sessions via `copilot --resume=<id>`. `maple resume-session copilot` works from the CLI.

### Fixed
- **`rtk` install fallback**: on linux/macos, if the GitHub release tarball approach fails (rate limit, unsupported arch, extraction error), both `scripts/install.sh` and `maple init` now fall back to piping the upstream install script — `curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh | sh`. After the upstream script exits, the Go installer searches `$PATH`, `~/.cargo/bin`, `~/.local/bin`, and `/usr/local/bin` for the resulting binary.

## [4.9.0] — 2026-04-21

### Changed
- **maple never exits for harness launches**: pressing `o` (open session), Enter in the superpower launch overlay, or Enter in the `L` launcher no longer calls `tea.Quit`. Spawning is now a `tea.Cmd` — if `spawnInNewTerminal` succeeds, a status message confirms it; if it fails, a manual-launch modal appears inside the TUI showing the command to paste, with `[c]` to copy to clipboard. maple stays alive in all cases.

### Added
- **Manual-launch modal**: when no new-terminal mechanism is detected (no tmux/zellij/WezTerm/Kitty, no GUI emulator), a centered overlay shows the exact command to paste in a new terminal window. Clipboard copy via `[c]` (pbcopy / xclip / xsel / clip depending on OS).

## [4.8.1] — 2026-04-21

### Fixed
- **Pipeline overlay live-updates**: `pipelineState` and `approvalPending` are now refreshed on every 5-second tick, so the `[P]` overlay shows current stage/status/approval without the user having to close and reopen it. The `[a]` approve button appears automatically when `approval-pending.txt` is written by the skill.

## [4.8.0] — 2026-04-21

### Added
- **New terminal spawning**: launching a harness (superpower overlay, `L` launcher, `o` open-session) now tries to open it in a new terminal tab/window so maple stays visible. Detection order: tmux → screen → WezTerm → Kitty → macOS iTerm2/Terminal.app → Linux (`x-terminal-emulator`, gnome-terminal, konsole, xterm) → Windows Terminal / cmd. Falls back to suspend-and-resume in the same terminal if none is found (SSH, headless, etc.).
- **`maple resume-session [harness]`**: resumes the project's pinned session from `.claude/state/sessions.json` directly from the CLI — no need to open the TUI. With no argument, prefers claude then opencode. Pass `claude` or `opencode` to target a specific harness.

## [4.7.2] — 2026-04-21

### Fixed
- **`o` key (open+pin session)**: now works for OpenCode sessions — `sessionUUID` was returning `""` for all non-Claude sources, so OpenCode sessions were never pinned and `opencode` was launched without a `--session` flag. Both are fixed.
- **`o` key feedback**: pressing `o` outside the Agents pane now shows a helpful status message instead of silently doing nothing.
- **`p` key pin confirmation**: status bar now shows "saved to .claude/state/sessions.json" so the user knows where to find it.
- **`opencode --session <id>` resume**: `agentOpenCmd` now passes `--session <id>` when opening an OpenCode session.

## [4.7.1] — 2026-04-21

### Fixed
- Launching a harness from the superpower overlay (or `L` launcher) no longer kills maple. The harness runs in the foreground of the same terminal; when it exits maple restarts the dashboard automatically. Works on all platforms and terminals — no new-tab/window APIs needed.

## [4.7.0] — 2026-04-21

### Changed
- **Superpower selection now launches directly**: pressing Enter on a superpower no longer exits and prints instructions. Instead it opens a launch overlay:
  - If a pinned session exists: goes straight to prompt input, showing the harness + session that will be resumed
  - If no pinned session: shows a harness picker first (Claude Code / OpenCode / Copilot), then prompt input
  - Optional context/prompt can be typed; Enter builds `<harness> [--resume <id>] "/superpower-runner <name> <context>"` and execs it

## [4.6.0] — 2026-04-21

### Added
- **RTK harness selector**: `R` key opens a multi-select overlay listing all supported harnesses (Claude Code/Copilot, Gemini CLI, Codex, Cursor, Windsurf, Cline/Roo Code, Kilo Code, Google Antigravity). Space toggles, Enter runs `rtk init` with the right flags for each. Already-installed harnesses shown as `✓`. State persisted to `.claude/state/rtk-harnesses.json`.

### Fixed
- **Stale pipeline**: P overlay now detects when a RUNNING pipeline hasn't been updated in >10 min (agent likely died). Shows grey "RUNNING (stale)" with a hint to press `[c]` to clear.
- **`[c]` key in P overlay**: deletes `maple.json` and `approval-pending.txt` so a dead pipeline doesn't block future runs.

## [4.5.0] — 2026-04-21

### Added
- **RTK token optimizer**: `install.sh` and `install.ps1` now auto-install `rtk` alongside `maple` for all platforms (mac/linux/windows). `maple init` runs `rtk init` to wire the `PreToolUse` hook. Boot check shows `rtk` status. Skip with `--skip-rtk`.
- **Session pinning**: TUI `p` key pins the selected agent session to `.claude/state/sessions.json`; `o` key auto-pins on open. `superpower-runner` reads pinned session IDs to resume work in the right context.
- **Launch dialog**: TUI `L` key opens a tool picker overlay — select Claude Code / OpenCode / Copilot, type a command, and launch it directly. Pinned sessions shown with ★.
- **File-based approval handoff**: at human-approval gates, `superpower-runner` writes `.claude/state/approval-pending.txt` and polls for its deletion. TUI `P` → `a` approves from the dashboard without switching to the agent terminal.
- **Shared state protocol**: `superpower-runner` SKILL.md (both `.claude/` and `.opencode/`) now documents all three shared state files (`maple.json`, `approval-pending.txt`, `sessions.json`) with owner columns so agents and TUI stay in sync.

### Fixed
- `install.ps1` had stale repo `kinncj/AI-Squad` — corrected to `kinncj/MAPLE`

## [4.4.2] — 2026-04-21

### Fixed
- `writeRecoveryMarker` now merges with existing `maple.json` instead of overwriting — TUI start/exit no longer wipes superpower pipeline state written by the `superpower-runner` skill, so the `P` pane shows the correct status

## [4.4.1] — 2026-04-21

### Fixed
- `install.sh` now finds the highest semver instead of using `/releases/latest` (which sorts by publish date, not version)
- Release fetch capped at `?per_page=100` — GitHub API max, single request
- Added `--version`/`-v` flag and `--install-dir` flag to `install.sh`

## [4.4.0] — 2026-04-21

### Changed — Gherkin-first cleanup sweep
- Rewrote `spec-kit` skill: story file IS the spec; dropped PROBLEM→SPEC→PLAN→TASKS state machine
- All doc examples (`ui-feature`, `api-endpoint`, `spike`) updated to Gherkin-first format
- Removed stale PROBLEM/SPEC/TASKS references from orchestrator, architect, and spec-kit agents
- Story frontmatter schema unified across template, validator, and spec-kit skill
- `copilot-instructions.md` and `stories.instructions` synced to canonical frontmatter
- Design and a11y gates made phase-aware (skip before design is approved)
- Pipeline status overlay (`P` key) added to TUI — reads `.claude/state/maple.json`
- `qa-cucumber` agent added; ADR enforcement gate wired into orchestrator
- `product-owner` agent deduplicated; `api-endpoint` example added to agents docs
- Added 8 missing agents to `AGENTS.md` quick-reference table

## [4.3.0] — 2026-04-16

### Added — Gherkin-first gates, orchestrator guardrails, superpowers
- `ship-safe` skill added to both `.claude/skills/` and `.opencode/skills/`
- `architect` and `orchestrator` agents updated to reference `ship-safe`
- OpenCode agents corrected to read `.opencode/skills/` (was wrongly reading `.claude/skills/`)
- `copilot-instructions.md` fleshed out with full commands, agents, and skills tables
- Ship-safe CI gate made opt-in via `ENABLE_SHIP_SAFE=true` repo variable
- TUI binary moved to `tui/`; `template/` stays at repo root via symlink
- Build dance (`rm symlink → cp -rL → go build → restore symlink`) wired into Makefile and CI
- Test suite updated: Go file checks target `tui/`, superpowers phase guarded by directory check
- `maple` binary and `.maple.json` added to `.gitignore`

## [3.7.0] — 2026-04-16

### Added — Phase VIII: Reference Implementations
- `docs/examples/ui-feature/` — full `new-ui-feature` superpower walkthrough (`ui: true` story, design gates, a11y)
- `docs/examples/api-endpoint/` — `api-endpoint` superpower (`ui: false`, no design gate)
- `docs/examples/spike/` — `spike/*` branch, Spec-Kit skip, no-production-merge pattern
- `README.md` updated: `maple` binary, `go build` quickstart, TUI keybindings, phase summary

## [3.6.0] — 2026-04-16

### Added — Phase VII: Enforcement Gates
- `template/lefthook.yml`: pre-push gates (spec-kit, frontmatter, design-approved, a11y), pre-commit no-secrets, post-merge log rotation
- `template/scripts/sdlc/validate-frontmatter.sh` — required fields + priority enum check
- `template/scripts/sdlc/a11y-gate.sh` — blocks on unresolved WCAG 2.2 AA critical/serious violations
- `template/scripts/sdlc/design-approved-gate.sh` — blocks if wireframe/mockup not approved for `ui: true` stories
- `template/scripts/sdlc/spec-kit-gate.sh` — checks spec progression integrity
- `template/scripts/sdlc/rotate-logs.sh` — rotates `.claude/logs/` at 10 MB, keeps last 5 compressed
- `template/.github/workflows/sdlc-gates.yml` — CI mirrors all lefthook gates
- `template/scripts/bootstrap-branch-protection.sh` — sets required checks + PR review on main

## [3.5.0] — 2026-04-16

### Added — Phase VI: TUI `maple`
- `tui/` — Go + Bubble Tea TUI binary; cross-compiles to macOS/Linux/Windows with zero runtime deps
- 8-pane dashboard: Stories, Agents, PRs, QA, Design, Logs, Help, Superpowers
- 5 built-in themes: `tokyo-night` (default), `catppuccin-mocha`, `gruvbox`, `nord`, `everforest`
- Animated logo: 3D depth shading, idle shimmer, theme-reactive palette per §5.10
- `--no-animate` flag; narrow terminal (<80 cols) degrades to single-pane scrolling mode
- Command mode (`:`), search mode (`/`), help overlay (`?`), superpower picker (`F`)
- `tui/README.md`: build, install, cross-compile, keybindings, theme switching

## [3.4.0] — 2026-04-16

### Added — Phase V: Superpowers
- `template/.claude/superpowers/schema.yaml` — YAML contract for all superpower declarations
- Built-in superpowers: `new-ui-feature`, `api-endpoint`, `bugfix`, `design-refresh`
- `superpower-runner` skill: YAML loader, stage executor, gate handler, resume logic (Claude Code + OpenCode)

## [3.3.0] — 2026-04-16

### Added — Phase IV: Spec-Kit
- `spec-kit` skill: PROBLEM→SPEC→PLAN→TASKS state machine, story emitter, skip logic for spike/chore branches
- `spec-kit.md` agent: approval-halted intake, hands off to orchestrator after TASKS.md approved
- `docs/specs/` directory standard; pre-DISCOVER integration into orchestrator (both platforms)

## [3.2.0] — 2026-04-16

### Added — Phase III: Design & UX Suite
- 6 new agents: `ux-researcher`, `wireframe-architect`, `visual-identity-designer`, `design-system-author`, `ui-mockup-builder`, `a11y-auditor` (Claude Code + OpenCode)
- 6 new skills: `wireframe`, `visual-identity`, `design-tokens`, `mockup`, `component-scaffold`, `a11y-audit`
- `docs/design/` directory standard: `research/`, `wireframes/`, `mockups/`, `identity/`, `system/components/`
- Orchestrator updated with design intake gate for `ui: true` stories

## [3.1.0] — 2026-04-16

### Added — Phase II: GitHub Integration Skills
- `gh-issues`, `gh-projects`, `gh-labels-milestones`, `gherkin-authoring`, `story-issue-sync`, `cucumber-automation` skills (Claude Code + OpenCode)
- `cmd_project` in `scripts/maple`: creates GitHub Project v2, stores `project_number` + `project_node_id` in `project.config.yaml`

## [3.0.0] — 2026-04-16

### Added — Phase I: Foundation
- `template/.claude/schemas/project.config.schema.json` — JSON Schema for `project.config.yaml`
- `template/project.config.yaml` — canonical per-project config with all v3 fields
- `template/docs/stories/_template.md` — story file template with frontmatter + Gherkin + DoD
- `template/docs/dod/definition-of-done.md` — default DoD including design and a11y gates
- Expanded label set: 11 label groups in `cmd_labels`

### Changed — Simplification track
- Removed swarm mode (`scripts/swarm-*.sh`) and hardcoded model names from all 54 agent files
- Removed empty scaffold stubs (`template/app/`, `template/common/`, `template/infra/`)
- Baked BusinessRepo/SOLID standards into `template/CLAUDE.md`, orchestrators, and architect agent
- Rewrote `scripts/maple`: 1,254 → ~700 lines
