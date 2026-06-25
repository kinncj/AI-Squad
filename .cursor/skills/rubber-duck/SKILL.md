---
name: rubber-duck
description: Invoke the rubber duck reviewer for a second opinion. Use after planning, after complex multi-file implementations, or after writing tests. Produces a focused CRITICAL/WARN/INFO list and a APPROVE/REQUEST_CHANGES verdict.
---

# rubber-duck skill

Invoke `@rubber-duck` to get an independent second opinion on a plan, implementation, or test suite. This is the same pattern as GitHub Copilot CLI's built-in Rubber Duck feature — a different perspective to catch what the primary agent missed.

## When to invoke

| Checkpoint | What to pass | What you get back |
|---|---|---|
| After Phase 3 (plan complete) | `plan.md` + `test-plan.md` | PLAN REVIEW |
| After complex multi-file implementation | Changed files / diff | CODE REVIEW |
| After tests written, before running | Test files | TEST REVIEW |
| Any time you feel uncertain | Anything relevant | Focused critique |

## How to call it

```
@rubber-duck Mode: PLAN REVIEW

Plan: <paste plan.md content or path>
Story: <paste story frontmatter + scenarios>
```

```
@rubber-duck Mode: CODE REVIEW

Files changed:
- src/scheduler.ts
- src/jobs/notify.ts
- src/queue/index.ts

<paste diff or file content>
```

```
@rubber-duck Mode: TEST REVIEW

Test files:
- tests/unit/scheduler.test.ts
- tests/e2e/notify.spec.ts
- features/scheduler.feature

<paste test content>
```

## Verdict handling

| Verdict | Orchestrator action |
|---|---|
| APPROVE | Proceed to next phase |
| REQUEST_CHANGES | Send findings to the responsible agent for revision; re-review once |

CRITICAL findings always block. WARN findings are surfaced to the human for a decision. INFO findings are logged but do not block.

## Tips

- You do not need to prepare anything special — just pass the relevant content.
- The rubber duck will not rewrite your code. It only surfaces findings. Acting on them is your job.
- For Copilot CLI users: the built-in `/experimental` Rubber Duck activates automatically at the same checkpoints. This skill adds equivalent coverage when using Claude Code or OpenCode.
