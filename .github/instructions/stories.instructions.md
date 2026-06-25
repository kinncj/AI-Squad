---
applyTo: "docs/stories/**/*.md"
---

# Story File Rules

Every story file must include valid YAML frontmatter. Required fields: `id`, `title`, `epic`, `priority`, `ui`, `adr_required`, `labels`. Optional: `milestone`, `issue_number`, `issue_url`, `created_at`.

```yaml
---
id: "<slug>-<NNNN>"
title: "Human-readable title"
epic: "<epic-slug>"
priority: "critical | high | medium | low"
ui: true   # true if ANY rendered UI element is involved
adr_required: false
milestone: null
labels:
  - "type:feature"
  - "priority:<level>"
  - "phase:discover"
issue_number: null
issue_url: null
created_at: "<ISO8601>"
---
```

## `ui` field rules
- `ui: true` — page, card, modal, form, button, navigation, chart, visual component
- `ui: false` — pure API, CLI, pipeline, migration, infrastructure, background job

## After writing or editing this file
```bash
bash scripts/sdlc/validate-frontmatter.sh <this-file>
```
