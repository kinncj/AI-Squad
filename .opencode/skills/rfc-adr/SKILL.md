---
name: rfc-adr
description: "Author and manage Architecture Decision Records (ADRs) in docs/specs/. Use when a significant architectural decision needs to be documented."
---

# SKILL: RFC / Architecture Decision Records

## ADR Format
Every significant architectural decision gets an ADR. Store in: `docs/specs/{feature}/adr.md`

```markdown
# ADR-{N}: {Short Title}

## Status
Proposed | Accepted | Deprecated | Superseded by ADR-{N}

## Context
What is the problem or situation that requires a decision?
What constraints exist?

## Goals
- {Specific, measurable goal}

## Non-goals
- {What this ADR does NOT address}

## Proposal
{Detailed technical proposal. Include code snippets if helpful.}

## Alternatives Considered

### Option A: {Name}
**Description:** {What it is}
**Pros:** {list}
**Cons:** {list}

### Option B: {Name}
**Description:** {What it is}
**Pros:** {list}
**Cons:** {list}

## Trade-offs and Risks
{Analysis of trade-offs. What could go wrong?}

## Impact

### Cost (FinOps)
{Estimated cloud cost impact. Monthly estimate if possible.}

### Operations (SRE)
{New runbook requirements. Alert thresholds. On-call implications.}

### Security
{Changes to attack surface. New threat vectors. Compliance implications.}

### Team
{Skill requirements. Training needed. Hiring implications.}

## Decision
{Final decision and rationale. Who made it and when.}

## Next Steps
- [ ] {Action item with owner}
- [ ] {Action item with owner}
```

## When to Write an ADR
- Choosing between two or more non-trivial technical approaches.
- Adopting a new technology or framework.
- Changing a significant existing pattern.
- Making a trade-off with known long-term implications.

## When NOT to Write an ADR
- Obvious choices with no real alternatives.
- Implementation details within an agreed-upon approach.
- Temporary decisions expected to change soon.
