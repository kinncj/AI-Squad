# CURSOR.md — Cursor IDE Configuration

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

Read skills from `.cursor/skills/` before executing tasks.

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

## Cursor-Specific Configuration

`.cursor/cursor.rules.json` is auto-discovered and applied by Cursor IDE. Customize per-project agent preferences here.

---

## Git Conventions

- Conventional Commits: `feat:`, `fix:`, `test:`, `docs:`, `infra:`, `refactor:`
- Branch naming: `feat/{slug}`, `fix/{slug}`
- Squash merge to main
- Never co-author commits with AI attribution

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
- Wireframes → `docs/design/wireframes/<story-id>.wireframe.{md,html,excalidraw}` — **all three files required every run; `.md` only is incomplete**
- Mockups → `docs/design/mockups/<story-id>.mockup.{tsx,html}`
- Visual identity → `docs/design/identity/`
- **Never write to `docs/wireframes/`, `docs/identity/`, `docs/mockups/`, or any path outside `docs/design/`.**

---

## Design Tokens

`docs/design/identity/tokens.json` is the canonical source (W3C DTCG format).
CSS vars, Tailwind config, Mantine theme are generated from it — never manually edit.
Token naming: `{category}.{group}.{role}` e.g. `color.brand.primary`.
