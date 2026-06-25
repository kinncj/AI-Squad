---
name: playwright-cli
description: "Use Playwright CLI for interactive browser exploration and snapshot-based interaction. Use when writing or running browser automation tests."
---

# SKILL: Playwright CLI

## When to Use
Use Playwright CLI (`playwright-cli`) for interactive browser exploration and snapshot-based interaction. Use `npx playwright test` for running actual test suites.

## Installation
```bash
npm install -g @playwright/cli
npx playwright install  # installs browsers
```

## Core CLI Commands

### Browser Exploration
```bash
# Open browser to URL
playwright-cli open http://localhost:3000

# Take accessibility snapshot (YAML with element references)
playwright-cli snapshot
# Output: saves to disk as snapshot.yml — review before acting

# Interact by element reference (e.g., e21 from snapshot)
playwright-cli click e21
playwright-cli fill e35 "test@example.com"
playwright-cli press e35 Enter
playwright-cli screenshot  # saves to disk
```

### Session Management
```bash
# Named sessions for auth state
playwright-cli --session=auth open http://localhost:3000/login
playwright-cli --session=auth fill e10 "admin@example.com"
playwright-cli --session=auth fill e11 "password"
playwright-cli --session=auth click e20  # submit button
playwright-cli session-list
```

## Snapshot-to-Test Workflow
1. `playwright-cli open http://localhost:3000`
2. `playwright-cli snapshot` → review element refs in snapshot.yml
3. Interact: `playwright-cli click e21`, `playwright-cli fill e35 "value"`
4. Observe results
5. Convert to test file:

```typescript
// tests/e2e/feature.spec.ts
import { test, expect } from '@playwright/test';

test('should {outcome} when {action}', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('button', { name: 'Sign In' }).click();
  await page.getByLabel('Email').fill('test@example.com');
  await page.getByLabel('Password').fill('password123');
  await page.getByRole('button', { name: 'Login' }).click();
  await expect(page).toHaveURL('/dashboard');
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
});
```

## Test Execution
```bash
# Run all E2E tests
npx playwright test tests/e2e/

# Run specific test file
npx playwright test tests/e2e/auth.spec.ts

# Visual debugger
npx playwright test --ui

# Headed mode (visible browser)
npx playwright test --headed

# Generate tests by recording
npx playwright codegen http://localhost:3000

# View HTML report
npx playwright show-report
```

## API Testing (Playwright request fixture)
```typescript
test('POST /api/users returns 201', async ({ request }) => {
  const response = await request.post('/api/users', {
    data: {
      email: 'test@example.com',
      name: 'Test User'
    }
  });
  expect(response.status()).toBe(201);
  const body = await response.json();
  expect(body).toHaveProperty('id');
});
```

## Token Efficiency Note
Use `playwright-cli snapshot` (saves to disk) instead of screenshots in context.
The compact YAML element refs (~1,350 tokens) vs accessibility tree injection (~5,700 tokens).
