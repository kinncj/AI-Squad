# Architecture Decision Records

ADRs document significant architectural decisions made during this project's lifetime. Each decision is captured as a numbered Markdown file in this directory.

## When to Write an ADR

Write an ADR whenever a decision:
- Introduces a new external dependency or service
- Changes the data model or storage layer
- Crosses Clean Architecture boundaries
- Requires a coordinated deploy or migration
- Has a non-obvious trade-off that future engineers need to understand

The orchestrator and architect agents will flag `adr_required: true` on stories that trigger ADRs.

## File Naming

```
docs/architecture/NNNN-short-decision-title.md
```

- `NNNN` is a zero-padded sequential number starting at `0001`
- Use the template at `docs/architecture/adr-template.md`

## Status Values

| Status | Meaning |
|---|---|
| `proposed` | Under discussion — not yet decided |
| `accepted` | Decision made and in effect |
| `deprecated` | Was accepted, no longer applies |
| `superseded` | Replaced by a later ADR (link to it) |

## Index

| # | Title | Status | Date |
|---|---|---|---|
| [0001](0001-herdr-primary-multiplexer.md) | herdr as the primary split-pane backend | accepted | 2026-07-07 |
| [0002](0002-go-native-design-review-portal.md) | Go-native design review portal | accepted | 2026-07-07 |

<!-- Add a row for each ADR as you create them -->
