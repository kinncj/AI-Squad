---
name: visual-identity
description: "Produce a coherent visual identity (palette, typography, spacing, radius) from a brief and serialise to JSON for the design-tokens skill. Use when establishing brand identity."
---

# SKILL: visual-identity

## Purpose

Produce a coherent visual identity from a brief or set of brand keywords. Outputs are palette, typography pairing, spacing scale, and radius scale — all serialised to JSON files that feed the `design-tokens` skill. Human approval required before tokens propagate to UI.

## Inputs

| Field | Source | Example |
|---|---|---|
| `brief` | free-text brand description | `"modern fintech, trustworthy, minimal"` |
| `keywords` | 3–6 brand keywords | `["trust", "clarity", "speed"]` |
| `primary_color` | optional hex override | `"#2563eb"` |
| `existing_tokens` | path to existing `tokens.json` | `docs/design/identity/tokens.json` |

## Outputs

| File | Location | Description |
|---|---|---|
| `palette.json` | `docs/design/identity/` | All color roles with hex + WCAG contrast |
| `typography.json` | `docs/design/identity/` | Font families, scale, weights, line-heights |
| `tokens.json` | `docs/design/identity/` | W3C DTCG format — canonical token file |

## Target awareness

When `design.target` (`project.config.yaml`, default `web`) is `tui`, keep the same JSON outputs but
constrain choices to the terminal: colors must be expressible in ANSI-256 (record the nearest 256
index alongside each hex), every fg/bg role pair must clear WCAG 2.2 AA contrast, and `typography.json`
describes terminal text styles (bold, dim/faint, italic, underline, reverse) rather than font families
and px sizes. The palette/typography/scale templates below still apply structurally; adapt their values
for the terminal.

## Palette Structure

`docs/design/identity/palette.json`:

```json
{
  "brand": {
    "primary":   { "value": "#2563eb", "on": "#ffffff", "contrast_aa": true },
    "secondary": { "value": "#7c3aed", "on": "#ffffff", "contrast_aa": true },
    "accent":    { "value": "#0ea5e9", "on": "#ffffff", "contrast_aa": true }
  },
  "semantic": {
    "success":  { "value": "#16a34a", "on": "#ffffff", "contrast_aa": true },
    "warning":  { "value": "#d97706", "on": "#000000", "contrast_aa": true },
    "error":    { "value": "#dc2626", "on": "#ffffff", "contrast_aa": true },
    "info":     { "value": "#0284c7", "on": "#ffffff", "contrast_aa": true }
  },
  "neutral": {
    "50":  "#f8fafc",
    "100": "#f1f5f9",
    "200": "#e2e8f0",
    "300": "#cbd5e1",
    "400": "#94a3b8",
    "500": "#64748b",
    "600": "#475569",
    "700": "#334155",
    "800": "#1e293b",
    "900": "#0f172a"
  },
  "surface": {
    "background": "#ffffff",
    "foreground": "#0f172a",
    "muted":      "#f1f5f9",
    "border":     "#e2e8f0"
  }
}
```

## WCAG Contrast Check

All palette entries with `on` values must pass WCAG 2.2 AA (4.5:1 for normal text, 3:1 for large text):

```python
def relative_luminance(hex_color):
    r, g, b = (int(hex_color[i:i+2], 16) / 255 for i in (1, 3, 5))
    def linearize(c):
        return c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4
    r, g, b = linearize(r), linearize(g), linearize(b)
    return 0.2126 * r + 0.7152 * g + 0.0722 * b

def contrast_ratio(fg, bg):
    l1 = relative_luminance(fg)
    l2 = relative_luminance(bg)
    lighter, darker = max(l1, l2), min(l1, l2)
    return (lighter + 0.05) / (darker + 0.05)

# AA normal text requires >= 4.5
# AA large text requires >= 3.0
ratio = contrast_ratio("#2563eb", "#ffffff")
assert ratio >= 4.5, f"FAIL: contrast {ratio:.2f} < 4.5 for #2563eb on #ffffff"
```

**If any pair fails AA, adjust the shade until it passes. Never ship a palette with failing contrast.**

## Typography Structure

`docs/design/identity/typography.json`:

```json
{
  "families": {
    "sans":  "Inter, system-ui, -apple-system, sans-serif",
    "mono":  "JetBrains Mono, 'Fira Code', monospace",
    "serif": null
  },
  "scale": {
    "xs":   { "size": "0.75rem",  "line": "1rem" },
    "sm":   { "size": "0.875rem", "line": "1.25rem" },
    "base": { "size": "1rem",     "line": "1.5rem" },
    "lg":   { "size": "1.125rem", "line": "1.75rem" },
    "xl":   { "size": "1.25rem",  "line": "1.75rem" },
    "2xl":  { "size": "1.5rem",   "line": "2rem" },
    "3xl":  { "size": "1.875rem", "line": "2.25rem" },
    "4xl":  { "size": "2.25rem",  "line": "2.5rem" }
  },
  "weights": {
    "regular": 400,
    "medium":  500,
    "semibold": 600,
    "bold":    700
  }
}
```

## Spacing + Radius Scales

Written directly into `tokens.json` (see `design-tokens` skill for full schema):

```json
{
  "spacing": {
    "1":  "0.25rem",
    "2":  "0.5rem",
    "3":  "0.75rem",
    "4":  "1rem",
    "6":  "1.5rem",
    "8":  "2rem",
    "12": "3rem",
    "16": "4rem"
  },
  "radius": {
    "none": "0",
    "sm":   "0.125rem",
    "md":   "0.375rem",
    "lg":   "0.5rem",
    "xl":   "0.75rem",
    "full": "9999px"
  }
}
```

## Write palette.json

```bash
mkdir -p docs/design/identity
# Write palette.json from the template above, populated with chosen values
# Then validate all contrast ratios before writing
python3 scripts/validate-palette.py docs/design/identity/palette.json
```

## Approval Gate

Mark `docs/design/identity/palette.json` approved before `design-tokens` runs:

Add to the top of `palette.json`:
```json
{ "_meta": { "status": "draft", "approved_by": null, "approved_at": null }, ... }
```

Check in `design-tokens` skill:
```bash
STATUS=$(python3 -c "import json; print(json.load(open('docs/design/identity/palette.json'))['_meta']['status'])")
[ "$STATUS" = "approved" ] || { echo "BLOCKED: palette not approved"; exit 1; }
```

## Failure Modes

| Condition | Action |
|---|---|
| Contrast ratio fails AA | Adjust shade. Re-check. Never skip. |
| `existing_tokens` provided | Merge: only fill missing roles; do not overwrite approved values. |
| No brief or keywords provided | Generate a neutral/professional default palette. Log `DEFAULT_PALETTE`. |
| `palette.json` exists and is approved | Do not overwrite. Log `SKIP — approved identity exists`. |

## Logging

```
[visual-identity] PALETTE    docs/design/identity/palette.json  contrast=all-pass
[visual-identity] TYPOGRAPHY docs/design/identity/typography.json
[visual-identity] SKIP       docs/design/identity/palette.json  (approved — locked)
[visual-identity] CONTRAST_FAIL  #2563eb on #ffffff  ratio=3.8  required=4.5
```
