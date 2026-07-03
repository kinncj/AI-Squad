# Copilot CLI Instructions — MAPLE

## Session Start Protocol (mandatory)

Before responding to any implementation request, check:

```bash
python3 -c "import json; s=json.load(open('.claude/state/maple.json')); print(s.get('status',''))" 2>/dev/null || echo "none"
```

- **`RUNNING` or `PAUSED`** — pipeline is active. Continue within it.
- **anything else** — no pipeline active. Route through `/pipeline-runner` before writing to `app/` or `tests/`:

```
/pipeline-runner implement-stories   — implement existing approved stories
/pipeline-runner new-ui-feature      — full UI pipeline with design gates
/pipeline-runner api-endpoint        — API feature pipeline
/pipeline-runner bugfix              — reproduce → fix → validate
```

Never write implementation code outside a running pipeline stage.

---

## Agent System

Default agent: `@orchestrator`. It never writes code — delegates everything to specialist agents.

## Commands

| Command | What it does |
|---|---|
| `/feature {description}` | Full 8-phase pipeline |
| `/bugfix {description}` | Reproduce → fix → validate → CHANGELOG |
| `/validate` | Run full test suite |
| `/tdd {requirement}` | Single RED → GREEN → REFACTOR cycle |
| `/pipeline-runner {name}` | Named taffy workflow: `new-ui-feature`, `api-endpoint`, `bugfix`, `design-refresh` |
| `/ship-safe` | Run `npx ship-safe audit .` security scan before shipping |

## RTK Token Optimizer

`rtk` is installed alongside `maple`. It hooks into Bash tool calls and compresses output before it reaches the LLM — reducing token use 60–90% on common dev commands (git, grep, build logs, test output). No command changes required. Disable per-session with `RTK_DISABLE=1`.

## Specialist Agents

| Agent | Responsibility |
|---|---|
| `@orchestrator` | Decomposes features, delegates, gates phases |
| `@architect` | ADRs, system design, threat models, pre-ship audit |
| `@qa` | Writes failing tests before implementation |
| `@rubber-duck` | Second-opinion reviewer at plan/impl/test checkpoints |
| `@spec-kit` | Requirement intake → Gherkin story files |
| `@ux-researcher` | Personas, journey maps |
| `@wireframe-architect` | Low-fi wireframes |
| `@visual-identity-designer` | Design tokens, brand system |
| `@ui-mockup-builder` | High-fidelity code mockups |
| `@a11y-auditor` | WCAG 2.2 AA audit |
| `@docs` | Technical documentation |
| `@product-owner` | Story prioritization |

## Skills (read before tasks)

Read skills from `.claude/skills/` before executing tasks:

| Skill | When to use |
|---|---|
| `karpathy-audit` | After Phase 5 IMPLEMENT (gates Phase 6 advancement); detects scope creep + over-engineering |
| `humanizer` | After Phase 7 DOCUMENT (polish prose, remove AI-isms) |
| `tdd-workflow` | Before any implementation (Phase 5) |
| `playwright-cli` | E2E / browser testing |
| `github-cli` | Issue + PR management via `gh` |
| `gh-issues` | Issue CRUD |
| `gh-projects` | GitHub Project board management |
| `gherkin-authoring` | Writing Gherkin story files |
| `rubber-duck` | Second-opinion review |
| `wireframe` | Before dispatching wireframe-architect |
| `a11y-audit` | Before dispatching a11y-auditor |
| `mermaid-diagrams` | Architecture / flow diagrams |
| `rfc-adr` | ADR format |
| `threat-modeling` | STRIDE threat analysis |
| `ship-safe` | Pre-ship security scan (`/ship-safe`) |

## Hard Rules (Non-Negotiable)

1. **TDD is mandatory.** `@qa` writes a failing test FIRST. Implementation follows. Never write implementation before a failing test exists.
2. **Orchestrator never writes code.** It decomposes, delegates, and gates.
3. **3 failures → escalate.** After 3 consecutive failures on any task, stop and surface to the human.
4. **No secrets in code.** Never write API keys, passwords, or tokens directly in source files.
5. **Conventional Commits.** All commits use `feat:`, `fix:`, `test:`, `docs:`, `infra:`, `refactor:`.
6. **`make test-all` must pass** before any PR is created (Phase 8 gate).
7. **Karpathy principles enforced at Phase 5→6 gate.** After IMPLEMENT, `/karpathy-audit` scores code; <70 blocks Phase 6. Never bypass.

## Karpathy Principles (Phase 5 → Phase 6 Gate)

After Phase 5 IMPLEMENT is complete, Orchestrator auto-calls `@karpathy-audit` to score against 4 principles:

| Principle | Enforces |
|-----------|----------|
| **Think Before Coding** | Assumptions stated. Ambiguities surfaced. No silent interpretations. |
| **Simplicity First** | Minimum code. No speculation, abstractions, or over-engineering. |
| **Surgical Changes** | Only requested changes. No unrelated refactoring or cleanup. |
| **Goal-Driven Execution** | Tests first. Success criteria explicit. Every line traces to requirement. |

**Scoring:** ≥90 auto-advance, 70-89 require approval, <70 BLOCK.

Manual invocation: `/karpathy-audit` or `@karpathy-audit` at any phase.

---

Read `project.config.yaml`:
- Check `stack:` — if any key is null, run tech-stack discovery (ask the user).
- Check `sdlc.mode` — if `standard`, Spec-Kit intake applies before DISCOVER.

## After Writing Code

Run these in order:
```bash
make lint
make test
```

If either fails, **fix the failure before proceeding**. Do not move to the next task with a red build.

## Before `git commit` or `git push`

Run SDLC gates:
```bash
bash scripts/sdlc/spec-kit-gate.sh
bash scripts/sdlc/validate-frontmatter.sh $(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null)
bash scripts/sdlc/design-approved-gate.sh $(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null)
bash scripts/sdlc/a11y-gate.sh $(find docs/stories -name "*.md" ! -name "_template.md" 2>/dev/null)
bash scripts/sdlc/adr-required-gate.sh
# ship-safe is optional — only run if ENABLE_SHIP_SAFE=true is set
[ "${ENABLE_SHIP_SAFE}" = "true" ] && npx ship-safe audit .
```

If any gate fails, **do not commit**. Report the failure and wait for the human to resolve.

## Story Files (`docs/stories/**/*.md`)

Every story file must have valid YAML frontmatter:
- `id`, `title`, `epic`, `priority`, `ui`, `adr_required`, `labels` (milestone optional, nullable)
- `ui: true` for ANY story involving a rendered UI element (pages, cards, modals, forms, navigation)
- `priority` must be `critical | high | medium | low`

After writing or editing a story file, validate:
```bash
bash scripts/sdlc/validate-frontmatter.sh <file>
```

## Rubber Duck — Second Opinion

GitHub Copilot CLI has a built-in **Rubber Duck** reviewer (experimental). Enable it with `/experimental`.

When Rubber Duck is active, it automatically provides a second opinion using a complementary model family at the three highest-value checkpoints:

1. **After planning** (Phase 3) — before implementation begins
2. **After complex multi-file implementations** — before tests run
3. **After writing tests** — before executing them

You can also trigger it manually at any time: say "critique your work" or "get a second opinion" and Copilot will invoke Rubber Duck, incorporate the feedback, and show you what changed and why.

**To enable:**
```
/experimental
```
Then select a Claude model from the model picker. Rubber Duck will use GPT-5.4 as the reviewer.

If Rubber Duck is not available (no `/experimental` access), the `@rubber-duck` agent defined in this project provides equivalent coverage — the orchestrator invokes it at the same three checkpoints.

## Phase Gates

| Phase | Gate condition |
|---|---|
| DISCOVER → ARCHITECT | Human approves stories |
| ARCHITECT → PLAN | Human approves architecture + ADR |
| PLAN → INFRA | plan.md complete, every impl task has a preceding test task |
| INFRA → IMPLEMENT | `docker compose up --wait` exits 0 |
| IMPLEMENT loop | RED (test fails) → GREEN (test passes) per task |
| IMPLEMENT → VALIDATE | All tests pass |
| VALIDATE → DOCUMENT | 100% test pass across all categories |
| DOCUMENT → FINAL GATE | Docs complete, CHANGELOG updated |
| FINAL GATE → PR | `make test-all` exits 0 |

## UI Stories (`ui: true`)

Insert design sub-pipeline before IMPLEMENT:
1. UX Research → personas + journey map
2. Wireframe → **PAUSE for human approval**
3. Visual Identity (if `docs/design/identity/tokens.json` missing) → **PAUSE for human approval**
4. Design Tokens → CSS vars, Tailwind config, Mantine theme
5. Mockup → **PAUSE for human approval**
6. Component Scaffold
7. After IMPLEMENT: A11y audit — no critical/serious WCAG 2.2 AA violations

**Canonical design artifact paths:**
| Artifact | Canonical Path |
|---|---|
| Wireframes | `docs/design/wireframes/<story-id>.wireframe.{md,html,excalidraw}` — **all three required per run** |
| Mockups | `docs/design/mockups/<story-id>.mockup.{tsx,html}` |
| Visual identity | `docs/design/identity/` |
| Design system | `docs/design/system/components/` |

Never write design artifacts to `docs/wireframes/`, `docs/identity/`, `docs/mockups/`, or any location outside `docs/design/`.

## Version & Issue Tracking (mandatory)

Every change we work on is tracked — automatic, not optional. When you start a piece of work (and before you finish it):

1. **Gate on a GitHub repo.** If `project.repo` is empty and there is no git remote, skip this section silently.
2. **Classify the SemVer bump** — **major** (breaking) / **minor** (new backward-compatible feature) / **patch** (bug fix, no API change), per the scheme in `project.config.yaml`.
3. **Milestone — major & minor only.** Read `github.milestone_granularity` (an absent key counts as `null`). `null` → ask the user once and persist the answer (`minor` = yes / `none` = no). `minor` → ensure `vX.Y.0` exists; a patch attaches to its minor's `vX.Y.0`, never its own. `patch` → patches get their own too. `none` → skip milestones.
4. **Issue — every change.** Create a GitHub issue labelled the matching **`type:*`** label (`type:bug` fix · `type:feature` feature · `type:docs`/`type:refactor`/`type:chore`, per `gh-labels-milestones`), assigned to the target major/minor milestone when milestones are enabled. Never `bug`/`enhancement`.
5. **Project board.** Read `github.project_number` (absent = `null`). `null` → ask the user once; if yes, bootstrap (`maple project` / `gh-projects`) and write the number back to config; if no, persist `0`. `0` → skip the board. `> 0` → add the issue and set **Status** (`In Progress` while working, `Done` when shipped) — read the number from `project.config.yaml`, never hard-code it.
6. **Link & close.** The PR references the issue (`Closes #N`); merging closes it. On release the milestone version equals the release tag. **No Claude/Anthropic attribution** anywhere — commits, PR bodies, issues, releases.

Mechanics live in the maple skills — use them, don't hand-roll: `gh-labels-milestones` (milestone/label upsert + granularity gate), `gh-issues` (create/label/milestone), `gh-projects` (board add + Status). Config: `project.config.yaml → github`.
