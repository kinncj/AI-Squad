# Example: Spike — Performance Investigation

Spikes bypass Spec-Kit, skip design gates, and are exempt from most SDLC
enforcement gates. They live on `spike/*` branches.

---

## When to Use a Spike

- Time-boxed technical investigation with no production code output
- Proof-of-concept before committing to an architectural direction
- Performance profiling / bottleneck identification
- Evaluating a library or approach before writing an ADR

---

## How to Start a Spike

Create a branch:
```bash
git checkout -b spike/perf-audit-export-query
```

Create a story with `type: spike` in frontmatter:

```markdown
---
id: "spike-perf-export-query"
title: "Spike: profile export query performance"
epic: "export"
priority: "high"
ui: false
adr_required: false
type: spike
labels:
  - "type:spike"
  - "priority:high"
issue_number: null
created_at: "2026-04-20T10:00:00Z"
---

# Spike: Profile export query performance

## Goal
Identify why the export query takes > 3s on datasets > 10k rows.
Time-box: 1 day.

## Scenarios

No Gherkin required for spikes — outcomes are documented in findings below.

## Findings

(filled in after investigation)

## Definition of Done
- [ ] Root cause identified
- [ ] Findings documented here
- [ ] ADR filed if architectural change recommended
- [ ] Spike branch closed (do not merge to main)
```

---

## Gates That Are Skipped for Spikes

| Gate | Status |
|---|---|
| Spec-Kit (PROBLEM → TASKS) | ⊘ skipped |
| Design intake (wireframe/mockup) | ⊘ skipped |
| A11y audit | ⊘ skipped |
| Feature frontmatter validation | ⊘ skipped |
| `make test-all` before push | ⊘ skipped |

lefthook detects `spike/*` branches and skips the above hooks automatically.

---

## Gates That Still Apply

| Gate | Status |
|---|---|
| `no-secrets` pre-commit hook | ✓ always on |
| Branch naming (`spike/<slug>`) | ✓ enforced |
| Issue creation | ✓ recommended |

---

## Closing a Spike

Spikes are **not merged to main**. When the investigation is done:

1. Document findings in the story's `## Findings` section.
2. If an ADR is warranted, file it in `docs/specs/adrs/`.
3. Close the PR (or don't open one).
4. Delete the spike branch.

```bash
git push origin --delete spike/perf-audit-export-query
```

If the spike leads to a feature, create a new story (non-spike) and use the
findings to inform the `@spec-kit` agent when writing the Gherkin story file.

---

## TUI Behavior for Spikes

The dashboard shows spike stories with a `◌` prefix to distinguish them from
feature stories:

```
┌─ Stories ──────────────────────────────┐
│   0042 export-csv          implement   │
│ ◌ spike-perf-export-query  spike       │
└────────────────────────────────────────┘
```
