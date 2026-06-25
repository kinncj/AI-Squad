---
story_id: "git-diff-viewer-0001"
story_file: "docs/stories/git-diff-viewer-0001.md"
target: tui
status: approved
approved_by: "human (pipeline)"
approved_at: "2026-06-25T22:34:45Z"
---

## Wireframe: Git changes diff viewer overlay (TUI)

Full-screen overlay over the dashboard grid: `header() + gitChangesView() + footer()` — same shape
as the `[P]` pipeline and the story/PR detail overlays. Body is a two-pane `lipgloss.JoinHorizontal`:
left = changed-file list, right = diff of the selected file. Never calls `tea.Quit`.

### Default state — changes present

```
  project: MAPLE · theme: tokyo-night
  📋 Gherkin: 7 | ▶️ Taffy: 1 (0 running)
┌ Git Changes ── 4 files ──────────────┬───────────────────────────────────┐
│ ▸ M  tui/dashboard.go                │ @@ -1182,6 +1197,15 @@ handleKey   │
│   M  tui/dashboard_views.go          │   	case "?":                       │
│   A  tui/git_changes.go              │ + 	case "C":                       │
│   ?? docs/scratch.md                 │ + 		m.gitChanges = loadGitChange… │
│                                      │ + 		m.showGitChanges = true        │
│                                      │   		return m, nil                  │
│  4 changed · 2 staged                │ ~ diff scrolls · +added −removed   │
└──────────────────────────────────────┴───────────────────────────────────┘
  [j/k] file · [J/K] scroll diff · [g/G] diff top/bottom · [q/esc] close
```

### Empty state — clean working tree

```
┌ Git Changes ──────────────────────────────────────────────────────────────┐
│                                                                            │
│              ✓ working tree clean — no changes to show                     │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
  [q/esc] close
```

### Error state — git unavailable / not a repo

```
┌ Git Changes ──────────────────────────────────────────────────────────────┐
│   ✗ git not available (not a git repo, or git not on PATH)                 │
└────────────────────────────────────────────────────────────────────────────┘
  [q/esc] close
```

### Focus order & keys

- `C` (global) — open the overlay (loads `git status` + first file's diff).
- `j` / `k` (or ↓/↑) — move file selection; preview updates to that file's diff; diff scroll resets.
- `J` / `K` — scroll the diff preview down / up (capital = diff, lowercase = file list).
- `g` / `G` — diff scroll to top / bottom.
- `q` / `esc` / `ctrl+c` — close.

### Regions (lipgloss styling — detailed in the mockup)

- Title bar: `Primary` bold; file count `Muted`.
- File-row status glyph: `M`=Warning, `A`=Success, `D`=Error, `??`=Muted. Selected row: `Accent ▸` prefix.
- Diff preview: added `+` lines `Success`, removed `−` lines `Error`, `@@` hunk headers `Accent`, context `Foreground`.
- Footer: `Muted`.

### Accessibility notes (terminal)

- File status is conveyed by the **letter glyph AND color** (never color alone) → passes `color-only-signaling`.
- Every action is keyboard-reachable; selected row has a visible `▸` marker (not color-only) → `focus-visible`.
- Read-only: `git status --porcelain -z`, `git diff`, `git diff --staged`. Never mutates the working tree.
