# CLAUDE.md — Project Rules

## Session Start Protocol (mandatory)

Before responding to any implementation request, run:

```bash
python3 -c "import json; s=json.load(open('.claude/state/maple.json')); print(s.get('status',''))" 2>/dev/null || echo "none"
```

- **`RUNNING`, `PAUSED`, or `RATE_LIMITED`** — a pipeline is active (RATE_LIMITED = paused on a rate limit). Continue within it; do not start a parallel one. For RATE_LIMITED, resume it — do not begin new work.
- **anything else** — no pipeline is active. Route through `/pipeline-runner` before touching `app/` or `tests/`:

```
/pipeline-runner implement-stories   — implement existing approved stories
/pipeline-runner new-ui-feature      — full UI pipeline with design gates
/pipeline-runner api-endpoint        — API feature pipeline
/pipeline-runner bugfix              — reproduce → fix → validate
```

Never write to `app/` or `tests/` outside a running pipeline stage. The `PreToolUse` hook enforces this and will hard-block the attempt.

---

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

## RTK Token Optimizer

`rtk` is installed alongside `maple` and wired via a `PreToolUse` hook. It intercepts Bash tool calls and compresses output (build logs, grep results, test output, git diffs) before they reach the LLM context window — reducing token consumption by 60–90% on common dev commands.

This is transparent: no commands change. If `rtk` is not installed, the hook is a no-op.

To disable for a single session: `RTK_DISABLE=1 claude`

Pipeline rules:
1. Orchestrator never writes code. Delegates to specialist agents.
2. **A Gherkin story file must exist in `docs/stories/` before any implementation begins.** The story IS the spec.
3. Tests are written before implementation (TDD enforced).
4. QA writes failing tests. Implementation agents make them pass.
5. 3 consecutive failures on any task → escalate to human.
6. `make test-all` must pass before Phase 8 gate.
7. Every feature gets a GitHub issue. Agents update it via `gh` CLI.
8. Conventional Commits: `feat:`, `fix:`, `test:`, `docs:`, `infra:`, `refactor:`.

**Gate enforcement:**
- No implementation without a Gherkin story file (orchestrator HALTS and REFUSES if absent)
- `ui: true` stories require approved wireframe + mockup before IMPLEMENT phase
- A11y audit required after IMPLEMENT for all `ui: true` stories

**Canonical design artifact paths (do not deviate):**
- Wireframes → `docs/design/wireframes/<story-id>.wireframe.{md,html,excalidraw}` — required files depend on `design.target` (**web** = md+html+excalidraw, **tui** = md+excalidraw, no html). See `docs/design/design-targets.md`.
- Mockups → `docs/design/mockups/<story-id>.mockup.{tsx,html,md}` — **web** = code (.tsx/.html) + .md; **tui** = .md only (lipgloss-annotated render).
- Visual identity → `docs/design/identity/` (palette.json, tokens.json, typography.json, …)
- Design system components → `docs/design/system/components/`
- Research → `docs/design/research/`
- **Never write design artifacts to `docs/wireframes/`, `docs/identity/`, `docs/mockups/`, or any path outside `docs/design/`.**

---

## Communication Style

- Short sentences. Structured formatting (bullets, tables, sections).
- No marketing language, hype, filler, or motivational tone.
- Explicit about assumptions, constraints, and trade-offs.
- Audience: senior engineers, staff+, VP/Director level.
- Do not explain fundamentals unless asked.

---

## BusinessRepo Model (Always On)

This repository is a **BusinessRepo** — a domain-centric repo that owns everything required to build, test, deploy, operate, and evolve one business capability end-to-end.

**Ownership scope per repo:**
- Application code
- Domain-specific shared libraries
- Infrastructure (Terraform, Kubernetes, cloud configs)
- CI/CD definitions
- All test layers (unit, integration, e2e, contract)
- Documentation

**Naming:** `<domain>`, `<domain>-service`, `<domain>-api`, `<domain>-app`.
Examples: `payments`, `payments-api`, `identity-service`, `export-app`.

**Canonical layout:**
```
/app        # application code (modularized internally)
/common     # domain-scoped shared libraries (no cross-domain imports)
/infra      # Terraform, K8s manifests, cloud configs
/tests      # all test layers
/docs       # specs, ADRs, runbooks, stories, design artifacts
Makefile    # all CI/CD calls Makefile targets
```

**Anti-patterns — reject any design that:**
- Creates horizontal shared repos ("shared-utils", "platform-common")
- Hides ownership behind tool-driven names ("terraform-repo", "k8s-manifests")
- Introduces cross-domain data coupling
- Requires coordinated deploys across unrelated domains

---

## Architecture Standards

- **Clean Architecture** — domain logic has zero framework/infrastructure imports
- **SOLID** — every module. Call out violations explicitly.
  - Single Responsibility: one reason to change
  - Open/Closed: extend without modifying
  - Liskov Substitution: subtypes honor contracts
  - Interface Segregation: no fat interfaces
  - Dependency Inversion: depend on abstractions
- Composition over inheritance
- Testability, observability, reliability, security by default

---

## Code Review Standard (Staff+)

- Call out boundary violations and architectural risk.
- Prefer long-term maintainability over short-term convenience.
- No politeness padding.
- Distinguish tactical vs strategic trade-offs.
- Explain reasoning and second-order effects.

---

## ADR / RFC Format

1. Title
2. Context
3. Goals / Non-goals
4. Proposal
5. Alternatives
6. Trade-offs and Risks
7. Impact (FinOps, SRE, Security, Team)
8. Decision
9. Next Steps

---

## Cost and Ops (Every ADR)

**FinOps:** Cost drivers, scaling characteristics, cost visibility.
**SRE:** Failure modes, blast radius, observability, recovery.

---

## MCP Servers

- `context7`: Library documentation lookup (`use context7` in prompts)
- New MCP servers require an ADR. Use `docs/architecture/adr-template.md` and add a row to `docs/architecture/README.md`.

---

## Skills

Read skills from `.claude/skills/` before executing tasks.

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
