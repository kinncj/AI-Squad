---
id: "git-diff-viewer-0001"
title: "Git changes diff viewer overlay"
epic: "tui-navigation"
priority: "medium"
ui: true
adr_required: false
milestone: null
phase: validate
labels:
  - "type:feature"
  - "priority:medium"
  - "phase:validate"
status: draft
issue_number: null
---

## Story

**As a** developer using the maple TUI,
**I want** a keybinding that opens a popup listing my git changes and lets me navigate through each file's diff,
**so that** I can review what changed in the working tree without leaving maple (the same way `[P]` pops up the pipeline).

## Notes

- Target medium: **TUI** (`design.target: tui`). No web/HTML.
- Modeled on the existing `[P]` pipeline overlay and the scrollable detail overlays (story/PR/QA): full-screen overlay over the grid, `header() + view + footer()`, never calls `tea.Quit`.
- Proposed keybinding: **`C`** (git **C**hanges) — capital, like `[P]`. Confirm during design; must not collide (`d`/`D`/`G`/`g` are taken).
- Data source: `git status --porcelain` for the file list, `git diff` / `git diff --staged` for per-file diffs. Read-only; no mutation of the working tree.

## Acceptance Criteria

```gherkin
@story:git-diff-viewer-0001 @epic:tui-navigation @priority:medium
Feature: Git changes diff viewer overlay

  Scenario: Open the overlay shows changed files
    Given the working tree has uncommitted changes
    When I press the git-changes keybinding from the dashboard
    Then an overlay opens listing each changed file with its status (modified/added/deleted/untracked)
    And the first file is selected
    And its diff is shown in a preview pane

  Scenario: Navigate between changed files
    Given the git-changes overlay is open with more than one changed file
    When I press "j" or "k"
    Then the selection moves to the next or previous file
    And the preview pane updates to that file's diff

  Scenario: Scroll a long diff
    Given a changed file whose diff is taller than the preview pane
    When I press the scroll keys in the preview
    Then the diff content scrolls without losing the file list

  Scenario: Clean working tree
    Given the working tree has no changes
    When I open the git-changes overlay
    Then it shows a "no changes" message instead of an empty list

  Scenario: Close the overlay
    Given the git-changes overlay is open
    When I press "q" or "esc"
    Then the overlay closes and the dashboard grid is shown again

  Scenario: git is unavailable
    Given git is not on PATH or the directory is not a git repo
    When I open the git-changes overlay
    Then it shows an error message and does not crash
```

## Definition of Done

- Unit tests green (Go, dance-wrapped `go test ./...`)
- Overlay renders the file list + selected diff; navigation and scroll work
- `q`/`esc` close; clean tree and no-git cases handled gracefully
- Keybinding documented in the `?` help overlay
- TUI design artifacts approved (wireframe + mockup + terminal a11y) with zero critical/serious
- `make build-tui` green; CHANGELOG entry added
