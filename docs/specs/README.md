# docs/specs/

Feature specification artifacts live here. Subdirectories hold architecture, ADRs, contracts, and phase artifacts produced by architect and other agents.

## Structure

```
docs/specs/
└── <epic>-<feature-slug>/
    ├── architecture.md          # System design — 10 sections, Mermaid diagrams
    ├── contracts/
    │   ├── openapi.yaml         # API contract
    │   ├── events.md            # Event schema
    │   ├── schema.sql           # Database schema
    │   └── seed-data.sql        # Test data seeds
    └── threat-model.md          # STRIDE threat model
```

## Source of Truth

The **story file** in `docs/stories/` is the spec. Architecture artifacts here are produced by the `@architect` agent in Phase 2 after the story is approved.

## Key Rules

- The `@spec-kit` agent writes story files directly to `docs/stories/` — not here.
- Architecture docs live here after ARCHITECT phase.
- Spec-Kit is **skipped** for `spike/*` and `chore/*` branches and `type:bug` stories.
- ADRs live in `docs/architecture/` — use `docs/architecture/adr-template.md`.
