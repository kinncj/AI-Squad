# Design — MAPLE TUI rework (better-ui-ux)

Date: 2026-07-03
Branch: `feature/better-ui-ux`
Reference: Heimdall (`kinncj/Heimdall`) — same maintainer, same Charm stack,
Heimdall-grade architecture is the bar.

## Why

MAPLE's `CLAUDE.md` preaches the BusinessRepo model (`app/`, `common/`, `tests/`),
Clean Architecture, SOLID, and per-component tests. The `maple` binary itself does
none of that: it is a flat `package main` under `tui/` with a **2511-line
`dashboard.go`**, a 1493-line `dashboard_views.go`, a 1238-line `req.go`, and no
tests on any of them. Heimdall — the same maintainer's other Go/TUI project on the
**same** `bubbletea v1.3.10` + `lipgloss v1.1.0` stack — is layered
(`app/cmd/*` + `app/internal/tui/{pane,render,theme,brand,splash,dashboard}`), every
file has a `_test.go`, and it has 23+ ADRs. This rework makes MAPLE eat its own dog
food and gives the TUI a real UX pass.

## Decisions (locked with the user)

| Decision | Choice |
|---|---|
| Strategy | **Clean parallel rebuild.** Build a fresh `app/internal/tui` to Heimdall's standard, port features to parity, cut over in one switch. Old `tui/` keeps building/shipping until cutover. |
| Primary goal | **Engineering quality AND UX, equally.** Heimdall-grade code (SOLID, small tested units, ADRs) and a visibly better terminal UX. |
| Framework | Stay on `bubbletea v1.3.10` + `lipgloss v1.1.0` (Heimdall matches). Bump Go `1.24.2 → 1.26` (Heimdall targets 1.26; toolchain present). |
| Scope | The **`maple` binary TUI only**. `template/` (what `maple init` copies into user repos) is out of scope and untouched. |
| Release | **No release/merge from this branch** until tested. Everything lands on `feature/better-ui-ux`; the shipping binary stays the old `tui/` build. |

## Non-goals

- No change to `template/` content or the `maple init` behaviour.
- No Bubble Tea v2 migration (Heimdall, the model, stays on v1.3.10).
- No new user-facing features beyond parity + UX polish (parity first).
- No rewrite of the install scripts or release workflow.

## Target architecture (mirrors Heimdall)

```
app/
  cmd/
    maple/            # new binary entry (thin: flags → run dashboard)
  internal/
    tui/
      theme/          # embedded maple-theme.json → lipgloss styles (dark/light)
      render/         # framework-free render helpers (bars, badges, truncation)
      pane/           # THE focus/scroll primitive + ISP capability interfaces
      brand/          # maple ASCII/wordmark assets + styling
      splash/         # splash screen
      dashboard/      # the model/update/view, composed of small tested files
    state/            # ports + adapters for file-based state (stories, sessions,
                      # pipeline, qa, design, logs) — replaces tui/loaders.go
```

Rules (Clean Architecture / SOLID):
- `pane` depends only on `theme` + `render`; nothing depends *up* into `dashboard`.
- `dashboard` depends on `pane`, `theme`, `render`, `state`; `state` has no TUI import.
- One responsibility per file; every file ships with a `_test.go`.
- Capability interfaces (ISP): a source implements only what it supports
  (`Source`/`Selectable`/`Filterable`/`Sortable`).

## The pane primitive (see ADR-003)

A framework-free `pane` package: `Pane` (one focusable, scrollable region owning
scroll window, selection-follows-scroll, `/` filter, sort cycle, focus ring, and
its on-screen `Rect` for mouse hit-testing) and `Group` (ordered panes + focus
index + page scroll; Tab/Shift-Tab focus, routed scroll/select/filter/sort keys,
pointer-targeted mouse). This is the single tested home for scroll/focus behaviour —
today that logic is inlined and duplicated across `dashboard.go`'s panes and every
overlay. Directly modeled on Heimdall ADR-0023.

## Parity target (what the rebuilt TUI must eventually match)

Top-level panes: `paneStories`, `paneAgents` (sessions), `panePRs`, `paneQA`
(2×2 grid), plus fullscreen `paneDesign` and `paneLogs`.

Overlays (18): help, story detail, session detail, QA file detail, test output,
PR detail, design, design review, git changes, logs, skills browser, pipeline
status, launcher, quick launch, quick prompt, RTK harness, ship-safe, manual launch.
Plus the boot check screen and the animated logo/splash.

Cross-cutting: 5s reload tick, shared-state protocol (`.claude/state/*`),
`spawnInNewTerminal` (8-env terminal detection), rate-limit handling, RTK wiring.

Parity is tracked as a checklist; the rebuild is not "done" until every item is
reachable in the new binary and the old `tui/` can be deleted.

## UX direction (the "better" half)

Grounded in what makes Heimdall's TUI good, applied to MAPLE's maple/arborist theme:
- **Theme system** — JSON-driven dark/light with structure/state/severity roles,
  replacing ad-hoc `lipgloss` color literals scattered through `dashboard_views.go`.
- **Consistent panes** — every pane/overlay uses the same focus ring, scroll
  affordances (`▲/▼ more · y/total`), and `/` filter, instead of each overlay
  re-implementing its own scroll.
- **Splash + brand** — a real maple splash and wordmark (today: `logo_anim.go`).
- **Focus model** — Tab/Shift-Tab across panes with a visible ring; selection never
  leaves the viewport; pointer-targeted mouse.
- Concrete layout/interaction mockups decided per-pane during implementation.

## Decomposition (sequential sub-projects, each spec → build → test on this branch)

1. **Foundation** — Go 1.26; `app/cmd/maple` + `app/internal/tui` skeleton; Makefile
   target for the new binary; old `tui/` untouched and still building.
2. **Primitives** — `theme`, `render`, `pane` (with `Group`), `brand`, `splash` —
   each fully tested. The architectural spine.
3. **State ports/adapters** — `app/internal/state`: port interfaces + adapters that
   read `.claude/state/*` and `docs/**`, replacing `tui/loaders.go`, fully tested.
4. **Dashboard + panes** — rebuild the 2×2 grid + fullscreen panes on `pane.Group`,
   small tested files.
5. **Overlays** — port the 18 overlays onto the pane/scroll primitive, tested.
6. **Cutover** — wire `maple init`/embed, terminal spawn, rate-limit, RTK; switch the
   default build to `app/cmd/maple`; delete `tui/`.
7. **ADRs + docs** — one ADR per real decision (layout, pane, theme, state ports,
   cutover); update `docs/` Heimdall-style.

## Coexistence + safety

- The new binary builds to `./bin/maple`; the shipping binary stays `./maple`
  (old `tui/`). `make build-tui` keeps working the entire time.
- Every commit keeps **both** builds green and all tests passing.
- No branch merge / no release until the user signs off after testing.

## Testing

- Every new package ships `_test.go` with meaningful coverage (Heimdall discipline).
- `pane`/`theme`/`render`/`state` are framework-free and unit-tested in isolation.
- `go test ./app/...` green on every commit; `make build-tui` (old) green too.
- Golden/snapshot tests for render output where stable.
