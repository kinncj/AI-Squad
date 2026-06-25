---
name: pipeline-runner
description: "Universal dispatcher: run a named taffy workflow (.claude/taffy/<name>.yaml), a skill (/skill-name), or a sub-agent (@agent-name). Falls back to skills.sh registry when a skill is not found locally. Tracks all runs in .claude/state/maple.json so the maple TUI shows live progress."
---

# SKILL: pipeline-runner

## What It Does

Dispatches any named workflow, skill, or agent from a single entry point. Resolution order:

1. **Taffy workflow** — look for `.claude/taffy/<name>.yaml`; if found, execute each stage in order
2. **Skill (local)** — look for `.claude/skills/<name>/`; if found, invoke the skill
3. **Agent** — look for `.claude/agents/<name>.md`; if found, delegate to `@<name>`
4. **Skill (registry)** — if not found locally, try to install from skills.sh, then retry step 2

Pipeline state is written to `.claude/state/maple.json` at every transition.

## Usage

```
/pipeline-runner <name>
```

Examples:
```
/pipeline-runner new-ui-feature
/pipeline-runner api-endpoint
/pipeline-runner implement-stories
/pipeline-runner tdd-workflow
/pipeline-runner orchestrator
```

List available taffy workflows:
```bash
ls .claude/taffy/*.yaml | grep -v schema
```

List available skills:
```bash
ls .claude/skills/
```

## Dispatch Protocol

### Step 1: Resolve the target

```bash
# Check taffy first
[ -f ".claude/taffy/<name>.yaml" ] && dispatch=taffy
# Then local skill
[ -d ".claude/skills/<name>" ] && dispatch=skill
# Then agent
[ -f ".claude/agents/<name>.md" ] && dispatch=agent
# Fallback: fetch from skills.sh registry, then retry
if [ -z "$dispatch" ] && command -v npx &>/dev/null; then
  echo "pipeline-runner: '<name>' not found locally — checking skills.sh…"
  npx --yes skills add kinncj/maple@<name> -a claude-code -y 2>/dev/null \
    || npx --yes skills add <name> -a claude-code -y 2>/dev/null \
    || true
  [ -d ".claude/skills/<name>" ] && dispatch=skill
fi
```

If nothing matches after the registry fallback, report:
`pipeline-runner: no taffy workflow, skill, or agent named '<name>' (also checked skills.sh registry)`

### Step 2: Initialise state

Write to `.claude/state/maple.json` (merge — do not overwrite unowned fields):

```json
{
  "taffy": "<name>",
  "stage": "<first-stage or skill-name>",
  "status": "RUNNING",
  "awaiting_approval": null,
  "started_at": "<iso8601>",
  "updated_at": "<iso8601>"
}
```

### Step 2b: Runtime policy enforcement (mandatory)

Before any stage execution, read and enforce:

- Harness-specific root markdown:
  - Claude harness → `CLAUDE.md`
  - OpenCode harness → `OPENCODE.md`
  - Cursor harness → `CURSOR.md`
  - Copilot harness → `COPILOT.md`
- `AGENTS.md`
- `.github/copilot-instructions.md`
- `.github/instructions/stories.instructions.md` (when touching story files)

When the launch prompt contains a `<maple-gherkin-handoff>` block:

1. Treat it as hard scope: implement only listed story paths / IDs.
2. Do not run Spec-Kit or regenerate stories.
3. Preserve the repository's current Cucumber stack:
   - If generated stories include `cucumber/*_steps.py`, use Python behave-style steps.
   - Do **not** introduce TypeScript `@cucumber/cucumber` unless the repository already uses it as the active standard.
4. Keep BusinessRepo structure and phase gates exactly as defined by instruction files.
5. Treat test layout as mandatory:
   - Gherkin feature files must live under `/tests/features/`.
   - Step definitions must use the repository's active Cucumber stack and live under `/tests`.
   - Do not place acceptance tests under `/app` or story directories.
6. Enforce module boundaries independent of language:
   - Runtime/source files must not import from `docs/`, `.github/`, or `.claude/`.
   - Copying/adapting approved artifact content into app/test source is allowed.
   - Design/spec artifacts are references, not runtime code dependencies.

### Step 3a: Taffy workflow execution

Load `.claude/taffy/<name>.yaml`, parse `stages:`, resolve `depends_on` order.

If the workflow defines an `orchestrator-kickoff` stage, execute it first before all other stages. This stage must publish the initial plan and heartbeat cadence.

For each stage:

**`when:` guard:**
- `when: ui:true` — skip if story has `ui: false`, **unless** `.claude/state/maple.json` has `force_ui: true` with `launch_source: maple-x` (MAPLE `[x]` quick-launch override).
- `when: ui:false` — skip if story has `ui: true`
- `when: always` — always run
- UI-related scope includes web, mobile, desktop, and TUI stories. Any such run must include design-review human-approval stages before implementation phases.

**`depends_on`:** all listed stages must be `DONE` before this one starts.

**Dispatch:**
- `agent: <name>` → delegate to `@<name>` with current story context
- `skill: <name>` → invoke the skill
- `pipeline: standard` → run the full 8-phase orchestrator pipeline

After each stage: update `maple.json` with current stage + `RUNNING`.

**Progress heartbeats (mandatory):**
- Send an immediate kickoff status before the first long-running tool/agent call.
- While a taffy run is active, send a concise progress update at least every 60-120 seconds.
- On each heartbeat, refresh `maple.json` `updated_at` and current `stage`.
- Every heartbeat must include concrete progress evidence:
  - changed files/artifacts since last update (explicit paths), or
  - a specific blocker that prevented changes.
- Use this status format:
  - Progress: `<stage / phase>`
  - Done since last update: `<brief>`
  - Current action: `<brief>`
  - Blockers: `<none or blocker>`
  - Next update: `<ETA>`
- Do not send heartbeat-only timestamp churn with no artifact/blocker details.
- If a stage requires writing artifacts and write access/tools are unavailable, set `maple.json` to `FAILED` with an explicit error and stop.
- If blocked/waiting, report what is pending and continue heartbeats until unblocked.

**Completion artifact gate (mandatory):**
- Before marking `DONE`, verify the run produced concrete story-linked artifacts under the BusinessRepo layout.
- Required for implementation runs:
  - application changes in `/app` (or existing domain folders),
  - tests in `/tests` (unit/integration/e2e as applicable),
  - Gherkin assets in `/tests/features` plus matching step implementations.
- Boundary check:
  - fail the run if generated runtime code imports paths under `docs/`, `.github/`, or `.claude/`.
- If required test/gherkin artifacts are missing, set `maple.json` to `FAILED` and report missing paths explicitly.

### Step 3b: Skill invocation

Invoke the skill directly. Update `maple.json` on start and completion.

### Step 3c: Agent delegation

Delegate to `@<name>`. Update `maple.json` on start and completion.

### Step 4: Human-approval gates (taffy only)

When a stage has `gate: human-approval`:

1. Complete stage work (produce artifact).
   - For design review stages (`wireframe`, `visual-identity`, `design-tokens`, `ui-mockup-builder`, `design-refresh`), artifact production is mandatory:
      - create at least one previewable artifact (`.excalidraw`, `.html`, `.svg`, `.png`, `.jpg`, `.jpeg`, `.webp`, or `.md`) under docs/design (or approved artifact dirs), and
      - for `wireframe`, `ui-mockup-builder`, and `design-refresh`, all three formats are required: `.md`, `.html`, and `.excalidraw` — a run that produces only `.md` is incomplete, and
      - update `.claude/state/design-artifacts.json` with current stage artifact paths so the review portal can update live.
   - **Path + completeness gate for `wireframe` stage (mandatory before PAUSING):**
     ```bash
     # Verify wireframes are in the canonical location and all three formats exist
     CANONICAL=$(find docs/design/wireframes -name "*.wireframe.*" 2>/dev/null | wc -l | tr -d ' ')
     MISPLACED=$(find docs -name "*.wireframe.*" -not -path "*/docs/design/wireframes/*" 2>/dev/null)
     MISSING_HTML=$(find docs/design/wireframes -name "*.wireframe.md" 2>/dev/null | while read md; do
       base="${md%.wireframe.md}"
       [ ! -f "${base}.wireframe.html" ] && echo "${base}.wireframe.html"
     done)
     MISSING_EXCALIDRAW=$(find docs/design/wireframes -name "*.wireframe.md" 2>/dev/null | while read md; do
       base="${md%.wireframe.md}"
       [ ! -f "${base}.wireframe.excalidraw" ] && echo "${base}.wireframe.excalidraw"
     done)
     if [ "$CANONICAL" -eq 0 ]; then
       if [ -n "$MISPLACED" ]; then
         echo "PIPELINE GATE FAILED: wireframes found at wrong path(s): $MISPLACED"
         echo "Canonical path is docs/design/wireframes/ — move files there before this stage can complete."
       else
         echo "PIPELINE GATE FAILED: no wireframe artifacts in docs/design/wireframes/"
       fi
       # set maple.json FAILED and stop
     fi
     if [ -n "$MISSING_HTML" ]; then
       echo "PIPELINE GATE FAILED: missing .wireframe.html for: $MISSING_HTML — generate it now."
       # set maple.json FAILED and stop
     fi
     if [ -n "$MISSING_EXCALIDRAW" ]; then
       echo "PIPELINE GATE FAILED: missing .wireframe.excalidraw for: $MISSING_EXCALIDRAW — generate it now."
       # set maple.json FAILED and stop
     fi
     ```
     If the gate fails, produce the missing files before re-running the gate check.
   - If no reviewable artifact exists for a design gate, set `maple.json` to `FAILED` and stop.
2. Write PAUSED state:
```json
{ "stage": "<name>", "status": "PAUSED", "awaiting_approval": "<name>", "updated_at": "<iso8601>" }
```
3. Write stage name to `.claude/state/approval-pending.txt`.
4. Output:
```
TAFFY PAUSED — awaiting human approval
Stage:    <stage-name>
Artifact: <artifact path or description>

Approve via the maple TUI ([P] pipeline → [a] approve) or reply "approved" / "continue".
I will not advance to the next stage until approval is confirmed.
```
5. Poll: `timeout 540 bash -c 'until [ ! -f .claude/state/approval-pending.txt ]; do sleep 2; done'`
   - On timeout (exit 124), re-run the same poll. The Bash tool caps at 10 min per call; re-polling across calls lets approval delays exceed that bound.
   - Also accept an explicit "approved" / "continue" reply in chat.
   - When the user approves via the maple TUI ([P] → [a]), the TUI deletes the pending file **and** sends a "continue" keystroke to the agent's pane via the active multiplexer (outer tmux/zellij, or a detached `tmux new-session` wrapper). Either signal is sufficient to resume.
6. On resume: update to `RUNNING`, advance to next stage.
7. While paused, monitor `.claude/state/design-feedback.json`:
   - `status: requested_changes` or `status: rejected` means apply the requested updates before advancing.
   - Treat `attachments` as required review inputs (uploaded files such as `.excalidraw`, images, HTML, text), typically under `docs/design/review-input/`.
   - Summarize how each feedback item and attachment was addressed before continuing.

### Step 5: Completion

```json
{ "taffy": "<name>", "stage": "DONE", "status": "DONE", "awaiting_approval": null, "updated_at": "<iso8601>" }
```

Output:
```
TAFFY COMPLETE — <name>
Stages run: N
Duration:   <elapsed>
```

## Failure Handling

After 3 consecutive failures on any stage:

```json
{ "stage": "<name>", "status": "FAILED", "error": "<summary>", "updated_at": "<iso8601>" }
```

Stop. Report failed stage and error to human. Do not proceed.

## Session Context

On startup, read `.claude/state/sessions.json` if it exists:

```json
{ "claude": "<uuid>", "opencode": "<id>", "copilot": "<id>" }
```

Use the matching session ID when resuming work within an existing agent session.

## State File Reference

All state in `.claude/state/`. TUI and skill share these files.

### `.claude/state/maple.json`

| Field | Owner | Values |
|---|---|---|
| `taffy` | skill | workflow/skill/agent name |
| `stage` | skill | current stage name |
| `status` | skill | `RUNNING`, `PAUSED`, `DONE`, `FAILED` |
| `awaiting_approval` | skill | local MAPLE stage name (e.g., `spec-kit` = Specification Knowledge & Integration Toolkit) or `null` — not an external package reference |
| `pipeline` | skill | `standard` if running 8-phase |
| `started_at` | skill | ISO 8601 |
| `updated_at` | skill | ISO 8601 |
| `state` | TUI | `running` or `exited` |
| `ts` | TUI | ISO 8601 |

**Merge-not-overwrite:** read existing file, update only owned fields, re-write.

### `.claude/state/approval-pending.txt`

Skill writes stage name. TUI deletes when user presses `[a]`.

### `.claude/state/sessions.json`

TUI writes harness→session-ID map. Skill reads for session resume.

## Skip Conditions

- `spike/*` and `chore/*` branches: skip Spec-Kit stages, run implementation stages.
- Stage `when: ui:true` on a `ui: false` story: skip silently, log `[pipeline-runner] SKIP stage=<name> reason=ui:false` unless quick-launch override is active (`force_ui=true`, `launch_source=maple-x`).
