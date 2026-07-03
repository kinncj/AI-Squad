# ADR-002: Rebuild the maple TUI as a BusinessRepo app/ layout

## Status
Accepted

## Context
The `maple` binary lives in a flat `tui/` `package main`: 30 files, a 2511-line
`dashboard.go`, a 1493-line `dashboard_views.go`, a 1238-line `req.go`, and no tests
on the large ones. This directly contradicts the BusinessRepo model, Clean
Architecture, and SOLID/testing rules that MAPLE's own `CLAUDE.md` mandates for the
projects it scaffolds.

Heimdall — the same maintainer, the same `bubbletea v1.3.10` + `lipgloss v1.1.0`
stack — is organised as `app/cmd/*` + `app/internal/tui/{pane,render,theme,brand,
splash,dashboard}` with a `_test.go` beside every file and 23+ ADRs. It is the proof
that this stack supports a clean, testable architecture. [ADR-001](ADR-001-tui-go-over-rust.md)
established Go as the TUI language; it did not address structure.

## Goals
- Make the maple binary follow the BusinessRepo layout MAPLE preaches.
- Small, single-responsibility, independently testable units with clear boundaries.
- Keep the product shipping throughout — no long broken window.

## Non-goals
- No language or framework change (Go + Bubble Tea stay; ADR-001 holds).
- No change to `template/` or `maple init` behaviour.
- No Bubble Tea v2 migration.

## Proposal
Rebuild the TUI clean, in parallel, under a BusinessRepo `app/` tree:

```
app/cmd/maple/            # thin entry: flags → run dashboard
app/internal/tui/         # theme, render, pane, brand, splash, dashboard
app/internal/state/       # ports + adapters for file-based state (no TUI imports)
```

Dependency rule: `state` and `pane`/`theme`/`render` never import `dashboard`;
`dashboard` composes them. The old `tui/` stays the shipping build (`./maple`,
`make build-tui`) until the new binary (`./bin/maple`) reaches feature parity; then
the default build switches and `tui/` is deleted. Every commit keeps both builds
green and all tests passing. Bump Go `1.24.2 → 1.26` to match Heimdall.

## Alternatives Considered
- **Incremental refactor of `tui/` in place.** Lower churn, but the monoliths and
  the flat `package main` fight every boundary; the maintainer chose a clean rebuild
  for a Heimdall-grade result.
- **Rewrite on Bubble Tea v2.** Rejected — Heimdall (the exemplar) stays on v1.3.10;
  no reason to add a framework migration to an architecture migration.

## Consequences
- Two binaries coexist on the branch until cutover; a Makefile target builds each.
- A parity checklist gates cutover (see the rework design spec).
- No merge/release from `feature/better-ui-ux` until the maintainer signs off.
