# Design Targets

The design phase produces artifacts for one UI medium, chosen by `design.target` in
`project.config.yaml` (default `web`). Design agents and skills read this value and emit
the matching column below. Artifact **paths** and the a11y **JSON shape** are identical
across targets — only the content differs — so the SDLC gates work unchanged for any target.

| Concern | `web` | `tui` |
|---|---|---|
| Wireframe | `{id}.wireframe.md` (ASCII) + `.html` (preview) + `.excalidraw` | `{id}.wireframe.md` (ASCII/box-drawing) + `.excalidraw` (editable diagram). No `.html`. |
| Mockup | `{id}.mockup.tsx` (React/Tailwind/Mantine) + `{id}.mockup.md` | `{id}.mockup.md` only: a fenced monospace terminal render + a Styles section annotating lipgloss (Foreground/Background/Border/Padding) per region. No `.tsx`. |
| Identity tokens | `tokens.css`, `tailwind.tokens.js`, `mantine.theme.ts` | `terminal-theme.json`: color roles mapped to ANSI-256/truecolor hex + lipgloss style names. |
| Accessibility | axe-core/pa11y against a preview URL | terminal a11y checklist (see below), written to the same `{id}.a11y.json` shape. |
| Review render | browser portal (HTML/SVG/TSX preview) | ASCII inline in the maple `D` overlay; monospace `<pre>` in the portal; `.excalidraw` opens in the portal. |

All targets write to: `docs/design/wireframes/{id}.wireframe.md`, `docs/design/mockups/{id}.mockup.md`,
`docs/design/mockups/{id}.a11y.json`, `docs/design/identity/`.

## Terminal a11y checklist (`tui` target)

Each check that fails becomes a `violations[]` entry in `{id}.a11y.json` with an `impact`:

| Check | id | impact when failing |
|---|---|---|
| Every action reachable by keyboard (no mouse-only path) | `keyboard-reachable` | critical |
| Selected/focused element is visually distinct (not color-only) | `focus-visible` | serious |
| Foreground/background pairs meet WCAG 2.2 AA contrast (4.5:1 text, 3:1 large/UI) | `color-contrast` | serious |
| State never conveyed by color alone (has symbol/text too) | `color-only-signaling` | serious |
| Degrades under `NO_COLOR` / monochrome terminals | `no-color-support` | moderate |
| No content loss/overflow at the minimum supported width | `min-width-resize` | moderate |

## a11y JSON shape (all targets)

```json
{
  "target": "tui",
  "url": "<story-id> terminal UI",
  "timestamp": "<ISO-8601>",
  "testEngine": { "name": "maple-tui-a11y", "version": "1" },
  "violations": [
    {
      "id": "color-contrast",
      "impact": "serious",
      "description": "status text on muted background is 3.1:1",
      "help": "raise contrast to >= 4.5:1",
      "nodes": [ { "target": ["footer status"], "failureSummary": "fg #888 on bg #1a1a1a" } ]
    }
  ],
  "passes": []
}
```

The gate reads only `violations[].impact` and fails on any `critical` or `serious`.
