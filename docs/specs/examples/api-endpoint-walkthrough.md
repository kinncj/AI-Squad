# Example: API Endpoint — No Design Gates

This example shows an API-only story (`ui: false`) moving through the full 8-phase pipeline without triggering design intake or a11y gates.

---

## Story File

Create `docs/stories/data-export-csv-0001.md`:

```yaml
---
id: "data-export-0001"
title: "API endpoint: export activity as CSV"
epic: "data-export"
priority: "medium"
ui: false
adr_required: false
phase: discover
labels:
  - "type:feature"
  - "priority:medium"
status: draft
---
```

```gherkin
@story:data-export-0001 @priority:medium
Feature: Export activity as CSV

  Scenario: successful export returns CSV
    Given the user provides a valid API key
    When they send GET /api/v1/exports/activity.csv
    Then the response status is 200
    And the Content-Type header is "text/csv"
    And the body contains rows for the last 90 days of activity

  Scenario: missing API key returns 401
    Given no API key is provided
    When they send GET /api/v1/exports/activity.csv
    Then the response status is 401

  Scenario: API key with no activity returns empty CSV
    Given the user has no recorded activity
    When they send GET /api/v1/exports/activity.csv
    Then the response status is 200
    And the body contains only the CSV header row
```

---

## Phase Gates (no design gates)

Because `ui: false`, the orchestrator **skips**:
- UX Research
- Wireframe (and human approval pause)
- Visual Identity
- Design Tokens
- Mockup (and human approval pause)
- Component Scaffold
- A11y audit

The pipeline runs straight through: DISCOVER → ARCHITECT → PLAN → INFRA → IMPLEMENT → VALIDATE → DOCUMENT → FINAL GATE.

---

## IMPLEMENT loop (TDD)

For each task in `plan.md`:

1. `@qa` writes a failing test for the endpoint.
2. Rubber Duck reviews the tests.
3. Implementation agent (`@typescript`, `@python`, etc.) makes the test pass.
4. `@qa` validates GREEN.

No design approvals needed. No a11y hook fires.

---

## Taffy shortcut

```
/pipeline-runner api-endpoint
```

Runs: spec-kit → 8-phase pipeline.
