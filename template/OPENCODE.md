# OPENCODE.md — OpenCode Configuration

## Session Start Protocol (mandatory)

Before responding to any implementation request, run:

```bash
python3 -c "import json; s=json.load(open('.claude/state/maple.json')); print(s.get('status',''))" 2>/dev/null || echo "none"
```

- **`RUNNING`, `PAUSED`, or `RATE_LIMITED`** — a pipeline is active (RATE_LIMITED = paused on a rate limit). Continue within it; do not start a parallel one. For RATE_LIMITED, resume it — do not begin new work.
- **anything else** — no pipeline is active. Route through `/pipeline-runner` before touching `app/` or `tests/`.

Never write to `app/` or `tests/` outside a running pipeline stage.

## Agent System

Default agent: `@orchestrator`. It never writes code — delegates everything to specialist agents via the Task tool.

Commands:
- `/feature {description}` — Full 8-phase pipeline
- `/build-feature {description}` — Alias for `/feature`
- `/bugfix {description}` — Reproduce → fix → validate → CHANGELOG
- `/validate` — Run full test suite
- `/tdd {requirement}` — Single RED → GREEN → REFACTOR cycle
- `/pipeline-runner {name}` — Launch a named taffy workflow (e.g. `new-ui-feature`, `api-endpoint`, `bugfix`, `design-refresh`)
- `/ship-safe` — Run `npx ship-safe audit .` security scan before shipping (**optional** — disabled by default; enable by setting repo variable `ENABLE_SHIP_SAFE=true`)

## Skills

Read skills from `.opencode/skills/` before executing tasks.

**Core skills:** `karpathy-audit`, `humanizer`, `tdd-workflow`, `playwright-cli`, `github-cli`, `mermaid-diagrams`, `pipeline-runner`, `ship-safe`.

### Karpathy Audit (Phase 5 → Phase 6 Gate)

After Phase 5 IMPLEMENT, orchestrator auto-calls `/karpathy-audit` to enforce 4 principles:

- **Think Before Coding** — Assumptions stated, ambiguities surfaced, no silent interpretations
- **Simplicity First** — Minimum code, no speculation/abstractions, 200→50 lines if possible
- **Surgical Changes** — Only requested changes, no unrelated refactoring/cleanup
- **Goal-Driven Execution** — Tests first, success criteria explicit, every line traces to requirement

Scoring: ≥90 auto-advance, 70-89 require approval, <70 **BLOCK**.

Manual: `/karpathy-audit` at any phase. Detects scope creep, over-engineering, hidden assumptions.

### Humanizer Skill

After Phase 7 DOCUMENT, invoke `/humanizer` to remove AI-isms from prose:

- Removes 29 AI-writing patterns (significance inflation, hedging, notability name-dropping, etc.)
- Voice calibration: provide sample of your writing for style matching
- Use before finalizing documentation, commit messages, comments

---

## Pipeline Phases

1. DISCOVER → 2. ARCHITECT → 3. PLAN → 4. INFRA → 5. IMPLEMENT → **[Karpathy Gate]** → 6. VALIDATE → 7. DOCUMENT → 8. FINAL GATE

**[Karpathy Gate]** — After Phase 5 IMPLEMENT, orchestrator auto-calls karpathy-audit to enforce code quality principles.
Score ≥90 auto-advance, 70-89 require approval, <70 BLOCK.

After Phase 7 DOCUMENT, call `/humanizer` to remove AI-isms from prose before merge.

---

## Git Conventions

- Conventional Commits: `feat:`, `fix:`, `test:`, `docs:`, `infra:`, `refactor:`
- Branch naming: `feat/{slug}`, `fix/{slug}`
- Squash merge to main
- Never co-author commits with AI attribution

---

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

## Story + Spec Context

Story files live at `docs/stories/<epic>-<story>-<timestamp>-NNNN.md`.
Specs live at `docs/specs/<epic>-<slug>/`.

When completing code for a story, respect the Gherkin scenarios in the story file as the source of truth for behavior.

---

## Test Expectations

- Unit tests for all pure functions and domain logic.
- Integration tests for infrastructure adapters.
- Gherkin/Cucumber scenarios for user-facing behavior (extracted into `tests/features/` at build time).
- A11y audit required for any component with `ui: true` in story frontmatter.

**Canonical design artifact paths (never deviate):**
- Wireframes → `docs/design/wireframes/<story-id>.wireframe.{md,html,excalidraw}` — required files depend on `design.target` (**web** = md+html+excalidraw, **tui** = md+excalidraw, no html). See `docs/design/design-targets.md`.
- Mockups → `docs/design/mockups/<story-id>.mockup.{tsx,html,md}` — **web** = code (.tsx/.html) + .md; **tui** = .md only (lipgloss-annotated render).
- Visual identity → `docs/design/identity/`
- **Never write to `docs/wireframes/`, `docs/identity/`, `docs/mockups/`, or any path outside `docs/design/`.**

---

## Design Tokens

`docs/design/identity/tokens.json` is the canonical source (W3C DTCG format).
CSS vars, Tailwind config, Mantine theme are generated from it — never manually edit.
Token naming: `{category}.{group}.{role}` e.g. `color.brand.primary`.
