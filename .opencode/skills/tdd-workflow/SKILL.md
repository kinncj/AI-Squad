---
name: tdd-workflow
description: "Drive development with a red-green-refactor TDD cycle, ensuring tests are written before implementation. Use when implementing any new functionality."
---

# SKILL: TDD Workflow

## The RED → GREEN → REFACTOR Cycle

### RED Phase (QA Agent)
1. Read the acceptance criterion or task description.
2. Write a test that will fail because the implementation doesn't exist.
3. Run the test. It MUST fail with a meaningful error (not a syntax error).
4. If the test passes immediately → the test is wrong. Rewrite it.
5. Report: file path, test name, failure message.

### GREEN Phase (Specialist Agent)
1. Read the failing test file at the provided path.
2. Implement the MINIMUM code to make ONLY that test pass.
3. Do not implement anything not required by the test.
4. Run the test. It must pass.
5. Run the full unit suite to verify no regressions.
6. If the test fails after 3 attempts → escalate to Orchestrator.

### REFACTOR Phase (Specialist Agent)
1. Look for: duplication, poor naming, long functions, complex conditionals.
2. Clean up without changing behavior.
3. Run the test again. It must still pass.
4. Commit: `git commit -m "refactor: {what was cleaned up}"`

## ATDD Pattern (Acceptance Test-Driven Development)
1. Write the E2E/acceptance test from the acceptance criterion (Given/When/Then).
2. Watch it fail (RED).
3. Drive out unit tests and implementation to make it pass (GREEN).
4. Acceptance test goes green last.

## Integration Test Container Lifecycle
```bash
# Before integration tests
docker compose -f docker-compose.test.yml up -d --wait

# Seed test data
./scripts/seed-test.sh
# or
make seed-test

# Run integration tests
make test-integration

# After tests
docker compose -f docker-compose.test.yml down -v
```

## Rules
- Test file is created BEFORE implementation file.
- Test name format: "should {expected outcome} when {condition}".
- Never weaken an assertion to make a test pass.
- Never mock what you can test with a real dependency (use containers).
- Implementation agent receives FILE PATH, not requirement text.
- Gate: test must fail before implementation, pass after.
