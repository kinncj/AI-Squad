---
name: karpathy-audit
description: Audit code changes against Karpathy's 4 principles (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution). Auto-called after Phase 5 IMPLEMENT; can also be invoked manually. Produces scored compliance report and blocks advancement if threshold not met.
---

# karpathy-audit skill

Audit PRs and code changes against Andrej Karpathy's 4 principles for reducing LLM coding mistakes.

## When to use

- Auto-invoked after Phase 5 (IMPLEMENT) before advancing to Phase 6 (VALIDATE)
- Manual invocation: `/karpathy-audit` or `@karpathy-audit` in chat
- At any phase when you want to assess code quality against the principles

## The 4 Principles

### 1. Think Before Coding
**Don't assume. Don't hide confusion. Surface tradeoffs.**

- State assumptions explicitly. If uncertain, ask.
- Present multiple interpretations—don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- Stop when confused. Name what's unclear and ask.

### 2. Simplicity First
**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If 200 lines could be 50, rewrite it.

### 3. Surgical Changes
**Touch only what you must. Clean up only your own mess.**

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- Only remove imports/variables/functions that YOUR changes made unused.

### 4. Goal-Driven Execution
**Define success criteria. Loop until verified.**

- Transform tasks into verifiable goals: tests before code.
- State a brief plan with explicit success criteria.
- Loop independently until verified.

## How to invoke

**Auto-call (Phase 5 → Phase 6 gate):**
Orchestrator automatically calls this skill after IMPLEMENT phase to audit the diff.

**Manual call:**
```
/karpathy-audit
```
or
```
@karpathy-audit
```

## Audit output

The skill produces a compliance report written to `.claude/state/karpathy-report.json`:

```json
{
  "phase": "IMPLEMENT",
  "timestamp": "2026-05-18T21:25:00Z",
  "spec_file": "docs/stories/user-auth/Story.md",
  "audit": {
    "think_before_coding": {
      "score": 85,
      "violations": ["Assumed DB schema without asking"],
      "evidence": ["Line 42: const User = model('User')"]
    },
    "simplicity_first": {
      "score": 92,
      "violations": [],
      "evidence": ["Reduced 180→120 lines in refactor"]
    },
    "surgical_changes": {
      "score": 78,
      "violations": ["Modified logging in unrelated file"],
      "scope_creep_files": ["src/logging/index.ts"],
      "out_of_spec": true
    },
    "goal_driven": {
      "score": 100,
      "violations": [],
      "evidence": ["All tests passing. RED→GREEN→REFACTOR complete."]
    },
    "overall": 89,
    "gate_decision": "PASS_WITH_APPROVAL",
    "recommendation": "Score 89: Pass scope and simplicity checks. Recommend human approval before advancing to Phase 6."
  }
}
```

## Scoring & Gate Decisions

| Overall Score | Decision | Action |
|---|---|---|
| ≥90 | 🟢 PASS | Auto-advance to next phase |
| 70-89 | 🟡 PASS_WITH_APPROVAL | Pause. Require human approval to advance. |
| <70 | 🔴 FAIL | BLOCK. Orchestrator halts. Requires remediation + re-audit. |

## Scope creep detection

Compares two sources:

1. **Original spec** — From `docs/stories/{story}/Story.md` (required scope)
2. **PR diff** — Files and line changes in the current PR

Violations flagged:
- Files modified outside the spec boundary
- Feature branches added beyond the story requirement
- Unrelated cleanup/refactoring

## Phase 5 → Phase 6 Gate

**Orchestrator behavior:**

```
Phase 5 (IMPLEMENT) complete
  ↓
Auto-call: /karpathy-audit
  ↓
Analyze spec + PR diff + 4 principles
  ↓
Write `.claude/state/karpathy-report.json`
  ↓
Score ≥90?      → Auto-advance to Phase 6
Score 70-89?    → Pause. Display report. Require [a]pprove or [c]cancel
Score <70?      → HALT. Show violations. Require /karpathy-audit re-run after fixes
```

## Dashboard integration

The TUI displays:
- **Phase 5 detail view** — Shows karpathy-report.json if present
- **[P] overlay (Pipeline pane)** — Per-principle scores + violations
- **Red blocking indicator** — If score <70, blocks manual phase advance
- **Approval gate** — If 70-89, shows "awaiting approval" state

## Remediation workflow

If audit fails or scores <70:

```
1. Orchestrator shows violation details
2. Specialist agent re-assesses work against the principle
3. Fix or remove out-of-scope code
4. Manually re-run: /karpathy-audit
5. If all 4 principles ≥70 now, approve advancement
```

## Example invocation

```
User: /feature "add password reset via email"
Orchestrator: [Phase 1-5 complete]
Orchestrator: Auditing against Karpathy principles...
karpathy-audit: Running audit...
[Analyze spec vs PR diff]
[Score each principle]
karpathy-audit: 📋 Report written to .claude/state/karpathy-report.json
karpathy-audit: 🟡 Score 82/100 — PASS_WITH_APPROVAL
karpathy-audit: ⚠️ Scope creep: modified src/logging (out of spec)

Dashboard: [Shows approval gate, awaiting [a] key]
User: a  [approve]
Dashboard: [Phase 6 VALIDATE begins]
```

## Further reading

- [Karpathy's observations on LLM coding mistakes](https://x.com/karpathy/status/2015883857489522876)
- [MAPLE 8-Phase Pipeline](../../docs/pipeline.md)
