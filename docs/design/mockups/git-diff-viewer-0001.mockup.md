---
story_id: "git-diff-viewer-0001"
wireframe: "docs/design/wireframes/git-diff-viewer-0001.wireframe.md"
target: tui
status: approved
approved_by: "human (pipeline)"
approved_at: "2026-06-25T22:34:45Z"
---

## Mockup: Git changes diff viewer overlay (TUI)

High-fidelity terminal render. Colors reference `docs/design/identity/terminal-theme.json`
(tokyo-night). Layout: `header() + gitChangesView() + footer()`; body is
`lipgloss.JoinHorizontal(Top, filePane, diffPane)`.

### Render — default (changes present), 78 cols

```
  project: MAPLE · theme: tokyo-night                                        ← header (muted)
  📋 Gherkin: 7 | ▶️ Taffy: 1 (0 running)                                     ← badges (accent)
╭─ Git Changes — 4 files ──────────────╮╭──────────────────────────────────╮
│ ▸ M  tui/dashboard.go                ││ @@ -1182,6 +1197,15 @@ handleKey  │
│   M  tui/dashboard_views.go          ││   case "?":                       │
│   A  tui/git_changes.go              ││ + case "C":                       │
│   ?? docs/scratch.md                 ││ +   m.showGitChanges = true       │
│                                      ││   return m, nil                   │
│ 4 changed · 2 staged                 ││ @@ -1610,0 +1620,3 @@ View        │
╰──────────────────────────────────────╯╰──────────────────────────────────╯
  [j/k] file · [J/K] scroll diff · [g/G] diff top/bottom · [q/esc] close      ← footer (muted)
```
- `▸` selected marker = **accent** (#bb9af7); selected row text **bold**.
- Status glyphs: `M` **warning** (#e0af68) · `A` **success** (#9ece6a) · `D` **error** (#f7768e) · `??` **muted** (#565f89).
- Diff: `+` lines **success**, `-` lines **error**, `@@` hunk headers **accent**, context **foreground** (#c0caf5).

### Render — empty (clean tree)

```
╭─ Git Changes ────────────────────────────────────────────────────────────╮
│                                                                           │
│              ✓ working tree clean — no changes to show                    │   ← success
│                                                                           │
╰───────────────────────────────────────────────────────────────────────────╯
  [q/esc] close
```

### Render — error (git unavailable)

```
╭─ Git Changes ────────────────────────────────────────────────────────────╮
│   ✗ git not available (not a git repo, or git not on PATH)                │   ← error
╰───────────────────────────────────────────────────────────────────────────╯
  [q/esc] close
```

### Styles (lipgloss → terminal-theme.json roles)

| Region | Foreground | Background | Border | Notes |
|---|---|---|---|---|
| Overlay title | `primary` bold | — | — | "Git Changes — N files" |
| File pane | `foreground` | — | RoundedBorder `muted` (active: `primary`) | width = (w-4)/2 |
| File row (default) | `foreground` | — | — | `  {glyph}  {path}` |
| File row (selected) | `accent` bold + `▸` | — | — | marker is a glyph, not color-only |
| Status `M` | `warning` | — | — | letter + color |
| Status `A` | `success` | — | — | |
| Status `D` | `error` | — | — | |
| Status `??` | `muted` | — | — | incidental |
| File count summary | `muted` | — | — | incidental text |
| Diff pane | `foreground` | — | RoundedBorder `muted` | width = (w-4)/2, scrollable |
| Diff `+` line | `success` | — | — | added |
| Diff `-` line | `error` | — | — | removed |
| Diff `@@` hunk | `accent` | — | — | hunk header |
| Footer keys | `muted` | — | — | incidental |

### States covered
default (changes), selected/navigated, scrolled diff, empty (clean), error (no git). All actions keyboard-reachable; selection has a non-color `▸` marker; status uses letter+color.
