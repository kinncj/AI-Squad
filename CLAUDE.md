# CLAUDE.md — MAPLE Repository

This file is for agents and contributors working on the MAPLE codebase itself (the `maple` binary, install scripts, template files). It is not the CLAUDE.md that gets copied into user projects — that lives at `template/CLAUDE.md`.

---

## Repository Layout

The Go source lives under `app/` (BusinessRepo / Clean Architecture rebuild; the old flat
`tui/` package is deleted). `app/cmd/maple` is the only `main` (holds the go:embed); every
other concern is a small, tested `app/internal/*` package.

```
.
├── app/
│   ├── cmd/maple/          # main: os.Args dispatch, runDashboardLoop, runUpdate,
│   │   ├── main.go         #   startDesignPortal, wrapInTmux, subcommand routing
│   │   ├── embed.go        #   //go:embed all:template
│   │   └── template/       #   symlink → ../../../template (real copy at build; see Build)
│   └── internal/
│       ├── state/          # FS reader: stories, sessions, PRs, tests, pipeline, gates
│       ├── scaffold/       # maple init copy + `maple update` Plan/Apply/patch (fs.FS)
│       ├── gh/             # maple labels / maple project (gh CLI)
│       ├── resume/         # maple resume-session
│       ├── selfupdate/     # maple self-update
│       ├── harness/        # launch a harness in a herdr/tmux/wezterm/kitty split; nudge panes
│       ├── portalsock/     # control-socket client (Hold connectivity + Emit events)
│       ├── req/            # pure gherkin-conversion core (parse/save/convert)
│       └── tui/
│           ├── dashboard/  # the Bubble Tea Model, panes, overlays, key/mouse handlers
│           ├── req/        # `maple req` UI + implement-stories launch glue
│           ├── menu/       # setup menu (uninitialised dir)
│           ├── pane/       # focus+scroll pane primitive (ADR-003)
│           ├── theme/      # embedded JSON themes, roles/states
│           ├── render/     # display-width truncate/pad (no bubbletea dep)
│           ├── splash/     # PNG/ASCII splash
│           └── brand/      # leaf glyph + tagline
├── template/               # Everything copied on `maple init` (.claude/.opencode/.cursor,
│                           #   .github, docs, scripts/design-review-portal.{sh,py}, Makefile)
├── scripts/                # install.sh / install.ps1 / legacy `maple` bash CLI
├── docs/                   # this repo's own specs + ADRs (ADR-002/003 = the rebuild)
├── tests/                  # shell test suite for the repo (tests/cli, tests/template)
└── CHANGELOG.md
```

---

## Build

`app/cmd/maple/template` is a **symlink** to `../../../template` for development. `go:embed`
can't follow symlinks, so the build/test targets swap it for a real copy, then restore it.
The Makefile handles the dance:

```bash
make build-tui       # → ./maple   (canonical binary, built from ./app/cmd/maple)
make build-app       # → ./bin/maple   (same, for local dev)
make build-tui-all   # cross-compile 5 platforms → dist/
```

**IMPORTANT — raw `go test ./app/...` BREAKS** on the `app/cmd/maple` symlink embed
(`pattern all:template: cannot embed irregular file template`). Always use:

```bash
make test-app        # does the dance, runs `go test ./app/...` + `go vet ./app/...`
```

Packages that don't import the embed (e.g. `./app/internal/harness`, `./app/internal/state`)
can be run directly with `go test ./app/internal/<pkg>/...`. Pass `TMUX=` to keep tmux-aware
tests hermetic.

---

## Test After Every Change

Before committing any change under `app/`:

```bash
make test-app && echo OK
```

No exceptions — if it doesn't build + pass, it doesn't commit. For the portal Python:
`python3 -c "import ast; ast.parse(open('template/scripts/design-review-portal.py').read())"`.
For install scripts: `bash -n scripts/install.sh`.

---

## Release Process

1. Commit and push to `main`
2. Wait for CI to go green (`gh run list --limit 3`)
3. Create the release tag — this triggers `release.yml` which cross-compiles all 5 platforms:

```bash
gh release create vX.Y.Z --title "vX.Y.Z — short description" --notes "..."
```

The release workflow builds:
- `maple-darwin-amd64.tar.gz`
- `maple-darwin-arm64.tar.gz`
- `maple-linux-amd64.tar.gz`
- `maple-linux-arm64.tar.gz`
- `maple-windows-amd64.zip`

**Never push a tag directly** — always use `gh release create`. The release action reads `on: release: types: [published]`.

---

## Versioning

Semver (`vMAJOR.MINOR.PATCH`). Current stream: `v4.x.x`.

| Change type | Bump |
|-------------|------|
| Bug fix, single broken behaviour | patch (`v4.9.1 → v4.9.2`) |
| New feature or TUI overlay | minor (`v4.9.x → v4.10.0`) |
| Breaking change to template schema or protocol | major (rare) |

Multiple related bug fixes in one release are still patch. A release that adds one new keybinding/overlay counts as minor.

---

## Commit Messages

Imperative, lowercase, under 72 characters. No AI phrasing.

**Banned:** enhance, leverage, ensure, implement, utilize, facilitate, improve maintainability, align with best practices, `Co-Authored-By: Claude`

**Good examples:**
```
fix rtk install: find binary in archive tree, isolate from set -e
fix gherkin runner: match 'test-features:' target not just substring
maple never exits when launching a harness
add zellij detection to terminal spawning, document multiplexer setup
fix test discovery: ** globs, Go dedup, Python unittest cmd, id[:8] panic
```

Stage specific files. Never `git add -A`.

---

## GitHub Issues and Project Board

Every bug fix and feature gets a GitHub issue before or immediately after the commit. Use the v4.10.0 milestone for current work.

```bash
gh issue create --title "..." --body "..." --label "bug"       # or "enhancement"
gh api repos/kinncj/MAPLE/issues/N --method PATCH -f milestone=1
```

Roadmap: https://github.com/users/kinncj/projects/67/views/1

After shipping a fix, close the issue:

```bash
gh issue close N --comment "Fixed in vX.Y.Z"
```

---

## Architecture: Key Invariants

### The dashboard is a Store-driven Bubble Tea Model

`dashboard.Model` renders from a `Store` interface (satisfied by `state.FS`). It never reads
the filesystem directly — all project state goes through `Store` methods, so the model is
unit-tested against fake stores. Side-effecting deps (`execFn`, `openFn`, `lookPath`,
`notifyContinue`) are injectable package vars/fields so tests never touch a real
terminal/browser/multiplexer.

### Real-time refresh: 2s local tick + 60s async net tick

`Init` starts `tickMsg` (every 2s → `reload()` local file state, preserving pane selection via
`pane.SetSource`) and `netTickMsg` (every 60s → async `gh` PR load → `prsLoadedMsg`). The
header/footer read pipeline status live per render, so status/approvals update without a
keypress. Adding a file-based state field means refreshing it in `reload()`.

### Harness launch: split pane in a multiplexer, else in-terminal — never a lost maple

`o`/`L`/`x`/`S`/`i` route through `Model.launch(args) → harness.LaunchInPane`. Backend
detection is **herdr-first** (`harness.InMultiplexer`, env order `HERDR_PANE_ID`, `TMUX`,
`WEZTERM_PANE`, `KITTY_WINDOW_ID`, `ZELLIJ`). Inside **herdr** it splits via the socket-API
CLI (`herdr pane split → run → rename`, id parsed from `result.pane.pane_id`); inside
tmux/wezterm/kitty it opens a **right-side split** (captures the pane id, titles the pane) so
maple stays visible and the pane is addressable; zellij gets a side pane (no pane-id API);
otherwise it falls back to `tea.ExecProcess` (suspend/resume in the current terminal). herdr
is the preferred backend but always optional — never bundled, never required (see
`docs/architecture/0001-herdr-primary-multiplexer.md`). The `maple req` "Implement via TAFFY"
launch shares this path (side pane when in a multiplexer, else in-terminal). To make splits
work in plain terminals, `runTUI` **auto-wraps maple into a multiplexer** when not already
in one: `wrapInHerdr` first (isolated persistent `maple` herdr session — stop-stale → headless
`server` → `workspace create` → sentinel+`wait output` → `exec maple` → attach; opt out
`MAPLE_NO_HERDR=1`), else `wrapInTmux` (styled tmux session; opt out `MAPLE_NO_TMUX=1`). Harness
launches must NOT `tea.Quit`.

`tea.Quit` + `ExitAction` is only for follow-up workflows that need the whole terminal:
`n`→req, `u`→update, `:labels`/`:project`. The outer `runDashboardLoop` runs them in-process
and re-enters.

### Approval gate is dual-signal and portal-synced

A gate is pending when EITHER `.claude/state/approval-pending.txt` exists OR
`maple.json.awaiting_approval` is set — matching the portal. `ApproveGate`/`RejectGate`
(state pkg) delete the file AND clear `maple.json` (`awaiting_approval`→null, `PAUSED`→`RUNNING`,
merge-preserving skill keys). On any tick where a gate just cleared — TUI **or** portal —
`reload()` calls `notifyContinue`, which types `continue` into recorded harness panes
(herdr `pane send-text`+`send-keys enter`, tmux/wezterm/kitty send-text) plus, as a fallback,
broadcasts to sibling tmux panes running a harness.

### `maple.json` is merge-not-overwrite

Any write to `maple.json` reads the existing file, unmarshals to a map, updates only its own
keys, and re-marshals. The skill owns the pipeline fields; the TUI/state only reconciles
`awaiting_approval`/`status`/`updated_at`.

### Design portal: dynamic port, control socket, SPA routing

`startDesignPortal` picks a free port (`findFreePort`, 7800–7900, `MAPLE_DESIGN_PORT` override),
runs `scripts/design-review-portal.sh start <port>`, and passes the authoritative URL to the
dashboard header (a clickable OSC-8 hyperlink; a plain click also opens it since mouse capture
eats OSC-8 clicks). maple holds a Unix-socket connection (`portalsock.Hold`) so the portal shows
live connectivity, plus a file heartbeat fallback; `maple emit` pushes events. The portal serves
its SPA for any non-`/api/`, non-`/artifact/` path (so agent-guessed URLs don't 404).

### Story parsing handles multi-line YAML

`state.frontmatter` parses scalar keys AND multi-line block lists (`labels:` → `- item` lines),
so `phase`/`priority`/`title` come from the real `Story.md` format. The story detail is
syntax-highlighted (`dashboard/storyview.go`).

### Test discovery uses `filepath.WalkDir`, not `filepath.Glob`

`filepath.Glob` does not support `**` in Go's stdlib — it silently matches nothing. All test
detectors that need to recurse use `filepath.WalkDir`.

---

## RTK Token Optimizer

RTK is installed alongside `maple` by both `install.sh` and `maple init`. It wires a `PreToolUse` hook that compresses Bash tool output before it reaches the LLM.

The `R` key in the dashboard opens a per-harness RTK wiring overlay. State is saved to `.claude/state/rtk-harnesses.json`.

---

## Shared State Protocol (TUI ↔ Portal ↔ Agents)

Communication goes through files in `.claude/state/` plus a control socket:

| File / channel | Writer | Reader | Purpose |
|------|--------|--------|---------|
| `maple.json` | Skill (pipeline fields) + TUI/portal (`awaiting_approval`/`status`) | All | Pipeline progress + gate |
| `approval-pending.txt` | Skill | TUI + portal | Human gate — delete to approve |
| `sessions.json` | TUI (`p`/`o`) | Skill (resume) | Pinned session IDs per harness |
| `panes.json` | harness pkg (on launch) | TUI + portal | Harness pane refs (kind/target) for the "continue" nudge |
| `design-portal.url` | portal script | — | Portal URL (header prefers the port maple assigned) |
| `maple-alive` | TUI (2s heartbeat) | portal | Connectivity fallback |
| `maple-sock.addr` + `maple.sock` | portal | TUI (`portalsock`) | Control socket: live connectivity + `maple emit` events |
| `rtk-harnesses.json` | TUI (`R` overlay) | — | Which harnesses have rtk wired |

---

## Current State

The TUI/CLI is the `app/`-based rebuild (ADR-002/003), developed on `feature/better-ui-ux`.
It reached full parity with the old `tui/` and added: real-time refresh, tmux side-pane
launch + auto-wrap, portal control socket, reviewable `maple update`, story rendering/
highlighting. Track further work as GitHub issues on the current milestone. Do not merge or
release without explicit direction.

---

## Style

- No comments unless the *why* is non-obvious. Never comment what the code does.
- No multi-line docstrings.
- No backwards-compatibility shims for removed behaviour.
- Unused exported symbols get deleted, not commented out.
- Keep overlay handlers grouped: check `if m.showX` before the global key switch, same order as the `View()` function.
