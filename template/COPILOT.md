# COPILOT.md — MAPLE Runtime Contract

## Session Start Protocol (mandatory)

Before responding to any implementation request, check:

```bash
python3 -c "import json; s=json.load(open('.claude/state/maple.json')); print(s.get('status',''))" 2>/dev/null || echo "none"
```

- **`RUNNING`, `PAUSED`, or `RATE_LIMITED`** — pipeline is active (RATE_LIMITED = paused on a rate limit). Continue within it; resume a RATE_LIMITED run rather than starting new work.
- **anything else** — route through `/pipeline-runner` before writing to `app/` or `tests/`.

Never write implementation code outside a running pipeline stage.

---

## Scope

This file defines mandatory runtime behavior for Copilot harness executions launched by MAPLE (especially TAFFY and `/pipeline-runner` flows).

## Required Inputs

Before executing pipeline work, read and enforce:

- `COPILOT.md` (this file)
- `AGENTS.md`
- `.github/copilot-instructions.md`
- `.github/instructions/stories.instructions.md` (when story files are in scope)

## Pipeline Runner Contract

Use:

```text
/pipeline-runner <name>
```

Resolution and execution behavior is defined by:

- `.claude/skills/pipeline-runner/SKILL.md` (and harness mirrors)

Treat that contract as authoritative for:

- workflow/skill/agent dispatch
- stage ordering and gates
- failure handling
- state file ownership and merge semantics

## Heartbeats (Mandatory)

While a TAFFY run is active:

1. Send an immediate kickoff update before first long-running call.
2. Send a concise progress heartbeat every 60–120 seconds.
3. Refresh `.claude/state/maple.json` (`stage`, `status`, `updated_at`) on each heartbeat.
4. Include concrete progress evidence on every heartbeat:
   - changed files/artifacts since last update (explicit paths), or
   - a specific blocker.
5. Use this structure:
   - Progress
   - Done since last update
   - Current action
   - Blockers
   - Next update (ETA)

Do not emit heartbeat-only timestamp churn.

## BusinessRepo and Test Boundaries

- Preserve BusinessRepo structure and phase gates.
- Required implementation artifacts must include app/domain changes plus tests in `/tests`, with Gherkin assets in `/tests/features` when applicable.
- Runtime/test code must not import from `docs/`, `.github/`, or `.claude/`.
- Copying/adapting reviewed artifacts from docs into app/test code is allowed; path-based imports/references are not.

## Approval Loop

At human-approval gates:

- write `PAUSED` state to `.claude/state/maple.json`
- write stage to `.claude/state/approval-pending.txt`
- wait for approval signal before advancing
- process `.claude/state/design-feedback.json` (including `attachments`) before resume when status indicates requested changes or rejection

For design review gates, also keep `.claude/state/design-artifacts.json` updated with previewable artifact paths so the MAPLE review portal reflects progress continuously.

**Canonical design artifact paths (never deviate from these):**
- Wireframes → `docs/design/wireframes/<story-id>.wireframe.{md,html,excalidraw}` — **all three files are required every run; producing only `.md` is incomplete**
- Mockups → `docs/design/mockups/<story-id>.mockup.{tsx,html}`
- Visual identity → `docs/design/identity/`
- **Never write to `docs/wireframes/`, `docs/identity/`, `docs/mockups/`, or any path outside `docs/design/`.**
