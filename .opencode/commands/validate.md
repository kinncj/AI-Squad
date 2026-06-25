---
description: Run the full test suite without discovery or architecture phases
---

Invoke the @qa agent with the following instruction:

"Run complete validation:
1. make test (unit tests)
2. make containers-up → make seed-test → make test-integration
3. make test-e2e (Playwright browser + API)
4. make test-contract (schema validation)
5. Smoke test all health endpoints.
6. Report: total pass/fail count per category.
7. If all pass: report VALIDATED.
8. If any fail: list failing tests with file paths and failure messages."
