---
description: Alias for /feature — run the full 8-phase feature development pipeline
---

Invoke the @orchestrator agent with the following instruction:

"Run the full 8-phase pipeline for this feature request: $ARGUMENTS

Phase sequence:
1. DISCOVER: @product-owner creates stories.md + acceptance-criteria.md → human gate
2. ARCHITECT: @architect creates architecture.md + adr.md + contracts/ + threat-model.md → human gate
3. PLAN: Create plan.md + test-plan.md (tests precede implementation for every task)
4. INFRA: @docker + @kubernetes + @terraform + @postgresql + @redis as needed → containers healthy
5. IMPLEMENT: TDD loop per task (QA writes RED → specialist makes GREEN) → all tests pass
6. VALIDATE: @qa runs full suite → 100% pass
7. DOCUMENT: @docs creates feature docs + CHANGELOG → complete
8. FINAL GATE: make test-all → exit 0 → create PR

Update GitHub issue at every phase transition."
