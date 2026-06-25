---
name: component-scaffold
description: "Generate a complete component file tree wired to design tokens with tests and Gherkin spec. Use when scaffolding a new UI component."
---

# SKILL: component-scaffold

## Purpose

Generate a complete component file tree wired to design tokens. Each component gets an implementation file, Storybook story, unit test, and Gherkin spec file. Skeletons are runnable with `TODO` stubs — not empty files. Stack is auto-detected from `project.config.yaml`.

## Inputs

| Field | Source | Example |
|---|---|---|
| `component_name` | PascalCase | `PasswordResetForm` |
| `story_id` | story frontmatter `id` | `auth-reset-0001` |
| `tokens_file` | `docs/design/identity/tokens.json` | required |
| `stack` | `project.config.yaml` | `react-mantine` \| `react-tailwind` \| `html` |
| `mockup_file` | approved mockup | `docs/design/mockups/auth-reset-0001.mockup.tsx` |

## Output Tree

For `PasswordResetForm` with stack `react-mantine`:

```
app/
└── components/
    └── PasswordResetForm/
        ├── index.tsx              ← component implementation
        ├── PasswordResetForm.stories.tsx  ← Storybook story
        ├── PasswordResetForm.test.tsx     ← unit tests (Vitest / Jest)
        └── PasswordResetForm.spec.ts      ← Gherkin step binding
```

## Generate Component Index (react-mantine)

`app/components/{ComponentName}/index.tsx`:

```tsx
// {ComponentName} — generated from story {story_id}
// Tokens: docs/design/identity/mantine.theme.ts
// TODO: implement from mockup docs/design/mockups/{story_id}.mockup.tsx

import { Stack, TextInput, Button } from '@mantine/core';
import { useState } from 'react';

export interface {ComponentName}Props {
  // TODO: define props from story acceptance criteria
  onSubmit?: (data: Record<string, unknown>) => void;
}

export function {ComponentName}({ onSubmit }: {ComponentName}Props) {
  // TODO: implement
  return (
    <Stack>
      {/* TODO: build from approved mockup */}
    </Stack>
  );
}

export default {ComponentName};
```

## Generate Storybook Story

`app/components/{ComponentName}/{ComponentName}.stories.tsx`:

```tsx
import type { Meta, StoryObj } from '@storybook/react';
import { {ComponentName} } from '.';

const meta: Meta<typeof {ComponentName}> = {
  title: 'Components/{ComponentName}',
  component: {ComponentName},
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};
export default meta;
type Story = StoryObj<typeof {ComponentName}>;

export const Default: Story = {
  args: {},
};

export const WithError: Story = {
  args: {
    // TODO: add error props
  },
};
```

## Generate Unit Test

`app/components/{ComponentName}/{ComponentName}.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { {ComponentName} } from '.';

describe('{ComponentName}', () => {
  it('renders without crashing', () => {
    render(<{ComponentName} />);
    // TODO: assert presence of key elements from wireframe
    expect(document.body).toBeTruthy();
  });

  it('calls onSubmit when form is submitted', () => {
    // TODO: implement interaction test
  });

  it('shows validation error when input is invalid', () => {
    // TODO: implement error state test
  });
});
```

## Generate Gherkin Step Binding

`app/components/{ComponentName}/{ComponentName}.spec.ts`:

```ts
// Gherkin step bindings for story {story_id}
// Feature file: tests/features/{epic}/{story_slug}.feature
import { Given, When, Then } from '@cucumber/cucumber';

// TODO: copy matching steps from cucumber-automation output
// Example:
// Given('the user is on the {string} page', async (page: string) => {
//   throw new Error('Pending');
// });
```

## Python Stack Alternative

For `stack=python` (e.g., Django / HTMX):

```
app/
└── components/
    └── password_reset_form/
        ├── __init__.py
        ├── template.html
        ├── test_password_reset_form.py
        └── password_reset_form_steps.py
```

## Scaffold Script

```bash
COMPONENT="PasswordResetForm"
STORY_ID="auth-reset-0001"
STACK="react-mantine"
DIR="app/components/$COMPONENT"

mkdir -p "$DIR"

# Check for existing files — never overwrite
for f in "index.tsx" "${COMPONENT}.stories.tsx" "${COMPONENT}.test.tsx" "${COMPONENT}.spec.ts"; do
  if [ -f "$DIR/$f" ]; then
    echo "[component-scaffold] SKIP  $DIR/$f  (already exists)"
  else
    echo "[component-scaffold] CREATE  $DIR/$f"
    # write the appropriate template (see above)
  fi
done
```

## Token Wiring

After scaffold, inject token import into `index.tsx`:

```bash
# Verify mantine.theme.ts exists
[ -f "docs/design/identity/mantine.theme.ts" ] || \
  echo "[component-scaffold] WARN  mantine.theme.ts missing — run design-tokens skill"
```

## Failure Modes

| Condition | Action |
|---|---|
| `ComponentName` not PascalCase | Warn and convert: `password-reset-form` → `PasswordResetForm`. |
| `app/components/` does not exist | Create it. Log the creation. |
| File already exists | Skip with `SKIP` log — never overwrite existing implementations. |
| `tokens.json` missing | Scaffold without token import. Add `// TODO: wire to design tokens` comment. |
| Stack `unknown` | Default to `react-mantine`. Log warning. |

## Logging

```
[component-scaffold] CREATE  app/components/PasswordResetForm/index.tsx
[component-scaffold] CREATE  app/components/PasswordResetForm/PasswordResetForm.stories.tsx
[component-scaffold] CREATE  app/components/PasswordResetForm/PasswordResetForm.test.tsx
[component-scaffold] CREATE  app/components/PasswordResetForm/PasswordResetForm.spec.ts
[component-scaffold] SKIP    app/components/PasswordResetForm/index.tsx  (already exists)
[component-scaffold] WARN    mantine.theme.ts missing
```
