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

## Version & Issue Tracking (mandatory)

Every change we work on is tracked — automatic, not optional. When you start a piece of work (and before you finish it):

1. **Gate on a GitHub repo.** If `project.repo` is empty and there is no git remote, skip this section silently.
2. **Classify the SemVer bump** — **major** (breaking) / **minor** (new backward-compatible feature) / **patch** (bug fix, no API change), per the scheme in `project.config.yaml`.
3. **Milestone — major & minor only.** Read `github.milestone_granularity` (an absent key counts as `null`). `null` → ask the user once and persist the answer (`minor` = yes / `none` = no). `minor` → ensure `vX.Y.0` exists; a patch attaches to its minor's `vX.Y.0`, never its own. `patch` → patches get their own too. `none` → skip milestones.
4. **Issue — every change.** Create a GitHub issue labelled the matching **`type:*`** label (`type:bug` fix · `type:feature` feature · `type:docs`/`type:refactor`/`type:chore`, per `gh-labels-milestones`), assigned to the target major/minor milestone when milestones are enabled. Never `bug`/`enhancement`.
5. **Project board.** Read `github.project_number` (absent = `null`). `null` → ask the user once; if yes, bootstrap (`maple project` / `gh-projects`) and write the number back to config; if no, persist `0`. `0` → skip the board. `> 0` → add the issue and set **Status** (`In Progress` while working, `Done` when shipped) — read the number from `project.config.yaml`, never hard-code it.
6. **Link & close.** The PR references the issue (`Closes #N`); merging closes it. On release the milestone version equals the release tag. **No Claude/Anthropic attribution** anywhere — commits, PR bodies, issues, releases.

Mechanics live in the maple skills — use them, don't hand-roll: `gh-labels-milestones` (milestone/label upsert + granularity gate), `gh-issues` (create/label/milestone), `gh-projects` (board add + Status). Config: `project.config.yaml → github`.

---

## Approval Loop

At human-approval gates:

- write `PAUSED` state to `.claude/state/maple.json`
- write stage to `.claude/state/approval-pending.txt`
- wait for approval signal before advancing
- process `.claude/state/design-feedback.json` (including `attachments`) before resume when status indicates requested changes or rejection

For design review gates, also keep `.claude/state/design-artifacts.json` updated with previewable artifact paths so the MAPLE review portal reflects progress continuously.

**Canonical design artifact paths (never deviate from these):**
- Wireframes → `docs/design/wireframes/<story-id>.wireframe.{md,html,excalidraw}` — required files depend on `design.target` (**web** = md+html+excalidraw, **tui** = md+excalidraw, no html). See `docs/design/design-targets.md`.
- Mockups → `docs/design/mockups/<story-id>.mockup.{tsx,html,md}` — **web** = code (.tsx/.html) + .md; **tui** = .md only (lipgloss-annotated render).
- Visual identity → `docs/design/identity/`
- **Never write to `docs/wireframes/`, `docs/identity/`, `docs/mockups/`, or any path outside `docs/design/`.**
