---
id: "__STORY_ID__"
title: "__TITLE__"
epic: "__EPIC__"
priority: "medium"
ui: false
adr_required: false
milestone: null
phase: discover
labels:
  - "type:feature"
  - "priority:medium"
status: draft
issue_number: null
---

## Story

**As a** __ROLE__,
**I want** __ACTION__,
**so that** __OUTCOME__.

## Acceptance Criteria

```gherkin
@story:__STORY_ID__ @epic:__EPIC__ @priority:medium
Feature: __FEATURE_TITLE__

  Scenario: __HAPPY_PATH__
    Given __PRECONDITION__
    When __ACTION__
    Then __EXPECTED_OUTCOME__

  Scenario: __EDGE_CASE__
    Given __PRECONDITION__
    When __EDGE_ACTION__
    Then __EDGE_OUTCOME__
```

## Definition of Done

- [ ] Unit tests green
- [ ] Integration tests green
- [ ] Cucumber/Behave scenarios green
- [ ] Wireframe approved (required when `ui: true`)
- [ ] Mockup approved (required when `ui: true`)
- [ ] A11y audit passed (required when `ui: true`)
- [ ] ADRs linked where required
- [ ] CHANGELOG entry added
- [ ] PR description references this story

## ADR Links

<!-- populated by architect agent when adr_required: true -->
