# ADR-003: One focus + scroll primitive for the maple TUI

## Status
Accepted

## Context
In the current `tui/`, scroll and selection are re-implemented per surface: each of
the four dashboard panes, the logs/design fullscreen panes, and all 18 overlays
carry their own offset math and edge handling. Selection indices are not wired to
scroll offsets, so a highlighted row can leave the viewport. There is no shared
focus model and mouse handling is inconsistent. This is the single biggest source of
duplication in `dashboard.go`/`dashboard_views.go`.

Heimdall solved the same problem with a framework-free `pane` package
([Heimdall ADR-0023](https://github.com/kinncj/Heimdall/blob/main/docs/architecture/0023-tui-focus-scroll-pane-primitive.md)):
one tested home for scroll/focus, ISP capability interfaces, and a `Group` that owns
focus + two-level scroll + pointer-targeted mouse. The maple rebuild adopts the same
primitive.

## Goals
- One tested scroll/focus primitive; selection always follows the cursor.
- Identical keys and affordances (`▲/▼ more · y/total`, `/` filter, sort cycle)
  across every pane and overlay.
- Tab / Shift-Tab focus between regions with a visible focus ring.
- Two-level (page + pane) scroll that never hides the focused active row.
- Pointer-targeted mouse (wheel scrolls the pane under the cursor; click focuses it).

## Non-goals
- No new persisted settings, no mouse drag-select, no text reflow.
- The pane renders rows; it does not know what the rows mean (the source does).

## Proposal
A framework-free package `app/internal/tui/pane` depending only on `theme` and
`render`. Nothing it depends on imports it back.

```go
// Capability interfaces — a pane gets only what its source opts into (ISP).
type Source     interface { Rows() []string }
type Selectable interface { Source; RowCount() int }
type Filterable interface { SetFilter(q string) }
type Sortable   interface { SortKeys() []string; SetSort(string) }
```

- **`Pane`** — a content `Source`, scroll offset, optional selection index, filter
  and sort state, and its on-screen `Rect` (captured at render for hit-testing). It
  owns the scroll window + clamp, the `▲/▼ more · y/total` affordance,
  selection-follows-scroll, the `/` filter input, the sort-key cycle, and the focus
  ring. Static panels implement only `Source`.
- **`Group`** — ordered panes + focus index + page-scroll offset. Tab/Shift-Tab
  cycle focus and draw the ring; scroll/select/`/`/sort route to the focused pane;
  the page auto-scrolls so the focused pane's active row stays visible; a `MouseMsg`
  is hit-tested against each pane's `Rect`.

`dashboard` (and every overlay) depends on `pane`; `pane` depends on neither. All
existing per-surface scroll code is deleted at cutover.

## Alternatives Considered
- **Keep per-surface scroll.** Rejected — it is exactly the duplication the rework
  exists to remove.
- **A `bubbles/viewport` per pane.** Heavier, still needs the focus/selection/filter
  glue on top, and couples panes to a Bubble Tea widget; a small framework-free
  primitive is more testable and matches Heimdall.

## Consequences
- One place to test and fix scroll/focus bugs; identical UX everywhere.
- Overlays become thin: a `Source` plus a title, not a scroll implementation.
