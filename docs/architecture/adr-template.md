---
adr: "NNNN"
title: "{Short Decision Title}"
status: proposed    # proposed | accepted | deprecated | superseded
date: "YYYY-MM-DD"
supersedes: null    # NNNN of the ADR this replaces, if any
superseded_by: null # NNNN of the ADR that replaces this, if any
deciders:
  - "{name or role}"
---

# NNNN — {Short Decision Title}

## 1. Context

<!-- What is the situation forcing this decision?
     Include constraints, prior state, and why this matters now. -->

## 2. Goals / Non-Goals

**Goals:**
- {what this decision must achieve}

**Non-Goals:**
- {what this decision explicitly does not address}

## 3. Proposal

<!-- Describe the chosen approach in concrete terms.
     Be specific enough that an engineer can implement it without asking questions. -->

## 4. Alternatives Considered

| Option | Pros | Cons | Why Rejected |
|---|---|---|---|
| {option A} | {pros} | {cons} | {reason} |
| {option B} | {pros} | {cons} | {reason} |

## 5. Trade-offs and Risks

<!-- Call out second-order effects, migration cost, operational burden,
     and anything that could go wrong. No sugar-coating. -->

## 6. Impact

**FinOps:** {cost drivers, scaling characteristics, visibility}

**SRE:** {failure modes, blast radius, observability, recovery procedure}

**Security:** {threat surface changes, data exposure, auth implications}

**Team:** {skill requirements, onboarding burden, ownership}

## 7. Decision

<!-- One paragraph. What was decided and why.
     Write it so a new team member can understand without reading the full doc. -->

Status: **proposed**

## 8. Next Steps

- [ ] {action item with owner}
- [ ] {action item with owner}
