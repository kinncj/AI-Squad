---
description: Reproduce a bug with a failing test, fix it, validate, and update CHANGELOG
---

Invoke the @orchestrator agent with the following abbreviated pipeline:

"Fix this bug: $ARGUMENTS

Steps:
1. REPRODUCE: @qa writes a failing test that demonstrates the bug.
   - Test must fail before the fix.
   - Test must describe the expected behavior.
2. FIX: Route to the appropriate specialist agent to make the test pass.
   - The fix must be minimal — do not refactor surrounding code.
3. VALIDATE: @qa runs make test to verify no regressions.
4. DOCUMENT: @docs adds a CHANGELOG entry under ### Fixed.
5. COMMIT: git commit -m 'fix: {description}'

Update GitHub issue throughout."
