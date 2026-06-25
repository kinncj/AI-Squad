---
description: Run a single RED → GREEN → REFACTOR TDD cycle for a specific requirement
---

Invoke the @orchestrator agent with the following instruction:

"Run a single TDD cycle for this requirement: $ARGUMENTS

1. RED: @qa writes a failing test.
   - Test file must not exist yet or test must currently fail.
   - Run the test — confirm it fails.
   - Report: test file path, failure message.
2. GREEN: Route to appropriate specialist to make the test pass.
   - Implement minimal code — no extras.
   - Run the test — confirm it passes.
3. REFACTOR: If code needs cleanup (duplication, naming), specialist cleans up while keeping test green.
   - Run test again — must still pass.
4. COMMIT: git commit -m 'feat: {requirement description}'
5. Report: test path, implementation files changed, final test result."
