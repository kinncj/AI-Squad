---
name: design-tokens
description: "Read and write design tokens in W3C DTCG format (tokens.json) and emit CSS, Tailwind, and Mantine outputs. Use when modifying or generating design tokens."
---

# SKILL: design-tokens

## Purpose

Read and write design tokens in W3C Design Token Community Group (DTCG) format (`tokens.json`). Emit framework-specific outputs: CSS custom properties, Tailwind config, and Mantine theme object. The `tokens.json` in `docs/design/identity/` is the canonical source of truth; all framework outputs are derived and regenerated — never manually edited.

## Token File: `docs/design/identity/tokens.json`

W3C DTCG format. Each token is an object with `$value` and `$type`:

```json
{
  "$schema": "https://design-tokens.org/schema/v1.0.0",
  "_meta": { "status": "draft", "version": "1.0.0" },

  "color": {
    "brand": {
      "primary":   { "$value": "#2563eb", "$type": "color" },
      "secondary": { "$value": "#7c3aed", "$type": "color" },
      "accent":    { "$value": "#0ea5e9", "$type": "color" }
    },
    "semantic": {
      "success": { "$value": "#16a34a", "$type": "color" },
      "warning": { "$value": "#d97706", "$type": "color" },
      "error":   { "$value": "#dc2626", "$type": "color" },
      "info":    { "$value": "#0284c7", "$type": "color" }
    },
    "neutral": {
      "50":  { "$value": "#f8fafc", "$type": "color" },
      "500": { "$value": "#64748b", "$type": "color" },
      "900": { "$value": "#0f172a", "$type": "color" }
    },
    "surface": {
      "background": { "$value": "#ffffff", "$type": "color" },
      "foreground": { "$value": "#0f172a", "$type": "color" },
      "muted":      { "$value": "#f1f5f9", "$type": "color" },
      "border":     { "$value": "#e2e8f0", "$type": "color" }
    }
  },

  "typography": {
    "fontFamily": {
      "sans": { "$value": "Inter, system-ui, sans-serif", "$type": "fontFamily" },
      "mono": { "$value": "JetBrains Mono, monospace",   "$type": "fontFamily" }
    },
    "fontSize": {
      "sm":   { "$value": "0.875rem", "$type": "dimension" },
      "base": { "$value": "1rem",     "$type": "dimension" },
      "lg":   { "$value": "1.125rem", "$type": "dimension" },
      "xl":   { "$value": "1.25rem",  "$type": "dimension" },
      "2xl":  { "$value": "1.5rem",   "$type": "dimension" },
      "4xl":  { "$value": "2.25rem",  "$type": "dimension" }
    }
  },

  "spacing": {
    "1": { "$value": "0.25rem", "$type": "dimension" },
    "2": { "$value": "0.5rem",  "$type": "dimension" },
    "4": { "$value": "1rem",    "$type": "dimension" },
    "8": { "$value": "2rem",    "$type": "dimension" }
  },

  "radius": {
    "sm":   { "$value": "0.125rem", "$type": "dimension" },
    "md":   { "$value": "0.375rem", "$type": "dimension" },
    "lg":   { "$value": "0.5rem",   "$type": "dimension" },
    "full": { "$value": "9999px",   "$type": "dimension" }
  },

  "shadow": {
    "sm": { "$value": "0 1px 2px 0 rgb(0 0 0 / 0.05)", "$type": "shadow" },
    "md": { "$value": "0 4px 6px -1px rgb(0 0 0 / 0.1)", "$type": "shadow" },
    "lg": { "$value": "0 10px 15px -3px rgb(0 0 0 / 0.1)", "$type": "shadow" }
  }
}
```

## Emit: CSS Custom Properties

Output to `docs/design/identity/tokens.css`:

```python
import json, re

def flatten(obj, prefix=""):
    result = {}
    for k, v in obj.items():
        if k.startswith("$") or k.startswith("_"):
            continue
        key = f"{prefix}-{k}" if prefix else k
        if isinstance(v, dict) and "$value" in v:
            result[key] = v["$value"]
        elif isinstance(v, dict):
            result.update(flatten(v, key))
    return result

tokens = json.load(open("docs/design/identity/tokens.json"))
flat = flatten(tokens)

lines = [":root {"]
for name, value in sorted(flat.items()):
    css_var = "--" + name.replace(".", "-")
    lines.append(f"  {css_var}: {value};")
lines.append("}")

with open("docs/design/identity/tokens.css", "w") as f:
    f.write("\n".join(lines) + "\n")

print(f"[design-tokens] CSS  docs/design/identity/tokens.css  vars={len(flat)}")
```

## Emit: Tailwind Config

Output to `docs/design/identity/tailwind.tokens.js`:

```python
import json

tokens = json.load(open("docs/design/identity/tokens.json"))

def extract_colors():
    result = {}
    for group, values in tokens.get("color", {}).items():
        result[group] = {}
        for name, token in values.items():
            if "$value" in token:
                result[group][name] = token["$value"]
    return result

colors = extract_colors()
font_family = {
    k: v["$value"]
    for k, v in tokens.get("typography", {}).get("fontFamily", {}).items()
}

config = f"""/** @type {{import('tailwindcss').Config}} */
// AUTO-GENERATED — edit docs/design/identity/tokens.json, not this file
module.exports = {{
  theme: {{
    extend: {{
      colors: {json.dumps(colors, indent(6))},
      fontFamily: {json.dumps(font_family, indent(6))},
    }},
  }},
}};
"""

with open("docs/design/identity/tailwind.tokens.js", "w") as f:
    f.write(config)

print("[design-tokens] TAILWIND  docs/design/identity/tailwind.tokens.js")
```

## Emit: Mantine Theme

Output to `docs/design/identity/mantine.theme.ts`:

```python
import json

tokens = json.load(open("docs/design/identity/tokens.json"))

primary = tokens["color"]["brand"]["primary"]["$value"]
secondary = tokens["color"]["brand"]["secondary"]["$value"]

theme = f"""// AUTO-GENERATED — edit docs/design/identity/tokens.json, not this file
import {{ createTheme }} from '@mantine/core';

export const theme = createTheme({{
  primaryColor: 'brand',
  colors: {{
    brand: [
      '{tokens["color"]["neutral"]["50"]["$value"]}',
      '{tokens["color"]["neutral"]["100"]["$value"] if "100" in tokens["color"]["neutral"] else "#f1f5f9"}',
      '{tokens["color"]["neutral"]["200"]["$value"] if "200" in tokens["color"]["neutral"] else "#e2e8f0"}',
      '{tokens["color"]["neutral"]["300"]["$value"] if "300" in tokens["color"]["neutral"] else "#cbd5e1"}',
      '{tokens["color"]["neutral"]["400"]["$value"] if "400" in tokens["color"]["neutral"] else "#94a3b8"}',
      '{tokens["color"]["neutral"]["500"]["$value"]}',
      '{primary}',
      '{tokens["color"]["neutral"]["700"]["$value"] if "700" in tokens["color"]["neutral"] else "#334155"}',
      '{tokens["color"]["neutral"]["800"]["$value"] if "800" in tokens["color"]["neutral"] else "#1e293b"}',
      '{tokens["color"]["neutral"]["900"]["$value"]}',
    ],
  }},
  fontFamily: '{tokens["typography"]["fontFamily"]["sans"]["$value"]}',
  fontFamilyMonospace: '{tokens["typography"]["fontFamily"]["mono"]["$value"]}',
}});
"""

with open("docs/design/identity/mantine.theme.ts", "w") as f:
    f.write(theme)

print("[design-tokens] MANTINE  docs/design/identity/mantine.theme.ts")
```

## Emit: Terminal Theme (tui target)

When `design.target` is `tui`, emit `docs/design/identity/terminal-theme.json` from `tokens.json`
instead of the CSS/Tailwind/Mantine outputs. Map each color role to an ANSI-256 index (and the hex)
plus a lipgloss style name, and record the foreground/background pairs with their contrast ratios so
the a11y auditor can flag any pair under WCAG 2.2 AA. Shape:

```json
{
  "roles": {
    "primary":    { "fg": "#7aa2f7", "ansi256": 111, "lipgloss": "primary" },
    "muted":      { "fg": "#565f89", "ansi256": 60,  "lipgloss": "muted" },
    "accent":     { "fg": "#bb9af7", "ansi256": 141, "lipgloss": "accent" },
    "error":      { "fg": "#f7768e", "ansi256": 204, "lipgloss": "error" },
    "success":    { "fg": "#9ece6a", "ansi256": 149, "lipgloss": "success" },
    "background": { "bg": "#1a1b26", "ansi256": 234 },
    "foreground": { "fg": "#c0caf5", "ansi256": 189 }
  },
  "pairs": [
    { "role": "foreground", "on": "background", "contrast": 12.6 },
    { "role": "muted",      "on": "background", "contrast": 3.4 }
  ]
}
```

## Run All Emitters

```bash
python3 - <<'EOF'
# Run all three emitters in sequence
# (inline the three scripts above, or import from scripts/emit-tokens.py)
EOF
```

## Update Tokens (read-write)

To update a single token value:

```python
import json, sys

tokens = json.load(open("docs/design/identity/tokens.json"))
# Set tokens["color"]["brand"]["primary"]["$value"] = "#1d4ed8"
# Write back, then re-run all emitters
json.dump(tokens, open("docs/design/identity/tokens.json", "w"), indent=2)
```

Always re-emit all framework outputs after any token change.

## Failure Modes

| Condition | Action |
|---|---|
| `tokens.json` malformed | Log parse error with line number. Do not emit partial output. |
| Missing required token key | Log `MISSING_TOKEN: {path}`. Emit with placeholder `TODO` value. |
| Emitted file differs from committed | Acceptable — these are always regenerated. |
| `_meta.status != approved` | Log warning. Emit anyway (dev workflow). Gate is in `visual-identity` approval, not here. |

## Logging

```
[design-tokens] READ    docs/design/identity/tokens.json  tokens=47
[design-tokens] CSS     docs/design/identity/tokens.css   vars=47
[design-tokens] TAILWIND docs/design/identity/tailwind.tokens.js
[design-tokens] MANTINE  docs/design/identity/mantine.theme.ts
[design-tokens] MISSING_TOKEN  color.neutral.100  — using placeholder
```
