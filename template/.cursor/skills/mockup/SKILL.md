---
name: mockup
description: "Scaffold high-fidelity UI component mockups using the project UI stack consuming approved wireframes. Use when implementing approved wireframes."
---

# SKILL: mockup

## Purpose

Scaffold high-fidelity UI component mockups using the project's declared UI stack (Mantine, Tailwind/shadcn, or plain HTML). Mockups consume approved wireframes and design tokens. Output is runnable code — not a design file. Human approval required before the mockup is treated as the implementation target.

## Inputs

| Field | Source | Example |
|---|---|---|
| `story_file` | path to story markdown | `docs/stories/auth-reset-0001.md` |
| `wireframe_file` | path to approved wireframe | `docs/design/wireframes/auth-reset-0001.wireframe.md` |
| `tokens_file` | canonical token file | `docs/design/identity/tokens.json` |
| `stack` | `project.config.yaml` | `react-mantine` \| `react-tailwind` \| `html` |

## Supported Stacks (web target)

| Stack value | Framework | Token source |
|---|---|---|
| `react-mantine` | `@mantine/core` | `mantine.theme.ts` |
| `react-tailwind` | Tailwind CSS + shadcn/ui | `tailwind.tokens.js` |
| `react-shadcn` | shadcn/ui + Tailwind | `tailwind.tokens.js` |
| `html` | Vanilla HTML + `tokens.css` | `tokens.css` |

## Outputs

| File | Location | Targets |
|---|---|---|
| `<story-id>.mockup.md` | `docs/design/mockups/` | web + tui — metadata + approval (tui: also the terminal render + lipgloss styles) |
| `<story-id>.mockup.tsx` | `docs/design/mockups/` | web only — React stacks |
| `<story-id>.mockup.html` | `docs/design/mockups/` | web only — html stack |

## Target awareness

`design.target` (`project.config.yaml`, default `web`):

- **web** — generate the code mockup (`.tsx`/`.html`) for the configured `ui_library`, plus `.mockup.md`.
- **tui** — generate ONLY `<id>.mockup.md`: a fenced monospace terminal render at the target width
  covering default/selected/empty/error/loading states, followed by a "Styles" section mapping each
  region to lipgloss `Foreground`/`Background`/`Border`/`Padding` taken from
  `docs/design/identity/terminal-theme.json`. Skip the stack detector and the React templates below.

## Pre-flight Checks

Before generating:

```bash
WIREFRAME="docs/design/wireframes/${STORY_ID}.wireframe.md"
TOKENS="docs/design/identity/tokens.json"

# 1. Wireframe must be approved
STATUS=$(python3 -c "
import re
m = re.search(r'^status:\s*(\w+)', open('$WIREFRAME').read(), re.MULTILINE)
print(m.group(1) if m else 'draft')
")
[ "$STATUS" = "approved" ] || { echo "BLOCKED: wireframe not approved"; exit 1; }

# 2. Tokens must exist
[ -f "$TOKENS" ] || { echo "BLOCKED: tokens.json missing — run visual-identity skill first"; exit 1; }
```

## Mockup Template: react-mantine

```tsx
// AUTO-GENERATED MOCKUP — docs/design/mockups/{story-id}.mockup.tsx
// Story: {story title}
// Wireframe: {wireframe file}
// Tokens: mantine.theme.ts
// Status: draft — awaiting approval

import { MantineProvider, Stack, TextInput, Button, Text, Alert } from '@mantine/core';
import { theme } from '../../docs/design/identity/mantine.theme';
import { IconAlertCircle } from '@tabler/icons-react';

interface Props {
  error?: string;
  onSubmit: (email: string) => void;
}

export function {ComponentName}({ error, onSubmit }: Props) {
  const [email, setEmail] = useState('');

  return (
    <MantineProvider theme={theme}>
      <Stack gap="md" maw={400} mx="auto" mt="xl">
        <Text fw={700} size="xl">{Screen Title}</Text>

        <TextInput
          label="Email"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={error}
        />

        {error && (
          <Alert icon={<IconAlertCircle />} color="red" variant="light">
            {error}
          </Alert>
        )}

        <Button fullWidth onClick={() => onSubmit(email)}>
          Send Reset Link
        </Button>
      </Stack>
    </MantineProvider>
  );
}
```

## Mockup Template: react-tailwind

```tsx
// AUTO-GENERATED MOCKUP — docs/design/mockups/{story-id}.mockup.tsx
import { useState } from 'react';

interface Props {
  error?: string;
  onSubmit: (email: string) => void;
}

export function {ComponentName}({ error, onSubmit }: Props) {
  const [email, setEmail] = useState('');

  return (
    <div className="max-w-sm mx-auto mt-16 space-y-4">
      <h1 className="text-2xl font-bold text-foreground">{Screen Title}</h1>

      <div>
        <label className="block text-sm font-medium mb-1">Email</label>
        <input
          type="email"
          className={`w-full border rounded-md px-3 py-2 text-sm ${error ? 'border-error' : 'border-border'}`}
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        {error && <p className="text-error text-sm mt-1">{error}</p>}
      </div>

      <button
        className="w-full bg-brand-primary text-white rounded-md py-2 font-medium hover:opacity-90"
        onClick={() => onSubmit(email)}
      >
        Send Reset Link
      </button>
    </div>
  );
}
```

## Mockup Metadata File

Always write a `.mockup.md` alongside the code file:

```markdown
---
story_id: "{story_id}"
wireframe: "docs/design/wireframes/{story_id}.wireframe.md"
stack: "{stack}"
status: draft        # draft | approved | rejected
approved_by: null
approved_at: null
---

## Mockup: {story title}

### States

- Default
- Error: {describe}
- Success: {describe}
- Loading: {describe if applicable}

### Token Usage

- Primary action: `color.brand.primary`
- Error text: `color.semantic.error`
- Border: `color.surface.border`

### Approval

- [ ] Matches approved wireframe
- [ ] Design tokens applied correctly
- [ ] All interaction states present
- [ ] Approved by product owner / UX lead
```

## Failure Modes

| Condition | Action |
|---|---|
| Wireframe not approved | Stop. Output `BLOCKED`. Do not generate mockup. |
| `tokens.json` missing | Stop. Run `visual-identity` first. |
| Stack `unknown` | Default to `html`. Log warning. |
| Mockup exists and is `approved` | Do not overwrite. Log `SKIP — approved mockup locked`. |
| Component name collision | Append story ID suffix to component name. |

## Logging

```
[mockup] CREATE   docs/design/mockups/auth-reset-0001.mockup.tsx  stack=react-mantine
[mockup] CREATE   docs/design/mockups/auth-reset-0001.mockup.md   status=draft
[mockup] BLOCKED  auth-reset-0001  wireframe not approved
[mockup] SKIP     auth-reset-0001  mockup approved — locked
```
