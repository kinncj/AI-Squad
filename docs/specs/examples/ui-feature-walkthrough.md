# Example: UI-Bearing Feature — End to End

This example traces a `ui: true` story from intake through merge using the full
Spec-Kit + Design Suite + 8-phase pipeline + TUI workflow.

---

## Story: Export Filtered Results as CSV

### 1. Write the Story

Run `/feature export filtered CSV` in Claude Code. The orchestrator dispatches
`@spec-kit` to write a Gherkin story at `docs/stories/export-filtered-csv.md`:

```yaml
---
id: "export-0001"
title: "Export filtered results as CSV"
epic: "export"
priority: "high"
ui: true
adr_required: false
phase: discover
labels:
  - "type:feature"
  - "priority:high"
status: draft
---
```

```gherkin
Feature: Export filtered results as CSV

  Scenario: successful export downloads CSV
    Given the analyst has applied filters to the results table
    When they click "Export CSV"
    Then a CSV file is downloaded
    And the file contains only the filtered rows from the last 90 days

  Scenario: export with no results returns empty CSV
    Given no rows match the active filters
    When they click "Export CSV"
    Then a CSV file is downloaded with only the header row

  Scenario: unauthenticated user cannot export
    Given the user is not logged in
    When they attempt to access the export endpoint
    Then the response is 401 Unauthorized
```

Human approves story → `status: approved` in frontmatter.

---

### 2. Design Intake Gate (triggered by `ui: true`)

Orchestrator detects `ui: true` and dispatches the design sub-pipeline before IMPLEMENT:

**Wireframe** saved at `docs/design/wireframes/export-0001.wireframe.md`:

```
┌─ Table Header ──────────────────────────────────────────────┐
│  Showing 47 results  (filter: status=shipped, last 90d)     │
│                                          [ Export CSV ↓ ]   │
└─────────────────────────────────────────────────────────────┘
```

Human approves wireframe → story phase advances.

Since no `tokens.json` exists, `@visual-identity-designer` is dispatched,
producing `docs/design/identity/palette.json` and `docs/design/identity/tokens.json`.

**Mockup** built by `@ui-mockup-builder` using Mantine + tokens:
`docs/design/mockups/export-0001.mockup.tsx`

Human approves mockup → `component-scaffold` skill runs:

```
app/components/ExportButton/
├── index.tsx
├── ExportButton.stories.tsx
├── ExportButton.test.tsx
└── ExportButton.spec.ts
```

---

### 3. Standard 8-Phase Pipeline

DISCOVER → ARCHITECT → PLAN → INFRA → IMPLEMENT → VALIDATE → DOCUMENT → GATE

Key checkpoints:
- `@architect` writes ADR if needed: `docs/architecture/0001-streaming-csv.md`
- `@qa` writes failing Cucumber scenarios before implementation
- `@typescript` implements to make tests green
- `@a11y-auditor` runs WCAG 2.2 AA audit; audit saved at `docs/design/mockups/export-0001.a11y.json`
- `@orchestrator` creates PR when `make test-all` exits 0

Story's `phase` frontmatter field advances as gates pass:
`discover` → `architect` → `plan` → `infra` → `implement` → `validate` → `document` → `done`

---

### 4. Monitoring in the TUI

```
┌─ Stories ─────────────────────┐  ┌─ Recent Agents ────────────────────┐
│ ▸ export-0001   implement      │  │ typescript   ExportButton.tsx      │
│   export-0002   discover       │  │ qa           export.spec.ts        │
│                                │  │ a11y-auditor contrast check        │
└────────────────────────────────┘  └────────────────────────────────────┘
┌─ PRs ─────────────────────────┐  ┌─ QA / Gherkin ─────────────────────┐
│ ● #131 export-0001   ● open   │  │  1 feature file(s)                 │
│                               │  │  3 scenario(s) total               │
└────────────────────────────────┘  └────────────────────────────────────┘
```

Press `d` to view wireframes and tokens in the Design pane.
Press `P` to show the active taffy pipeline status.
Press `x` to launch the `new-ui-feature` taffy workflow on a new story.

---

### Artifacts Produced

| Artifact | Location |
|---|---|
| Story file | `docs/stories/export-filtered-csv.md` |
| Wireframe | `docs/design/wireframes/export-0001.wireframe.md` |
| Design tokens | `docs/design/identity/tokens.json` |
| Mockup | `docs/design/mockups/export-0001.mockup.tsx` |
| A11y report | `docs/design/mockups/export-0001.a11y.json` |
| Architecture | `docs/specs/export-filtered-csv/architecture.md` |
| ADR | `docs/architecture/0001-streaming-csv.md` |
| Component scaffold | `app/components/ExportButton/` |
| Feature files | `tests/features/export-filtered-csv.feature` |
| PR | `#131 feat/export-filtered-csv` |
