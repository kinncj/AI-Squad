# AGENTS.md — Multi-Agent Squad Roster

## Quick Reference

| # | Agent | Role |
|---|---|---|
| 1 | orchestrator | Pipeline control (never codes) |
| 2 | product-owner | User stories, acceptance criteria |
| 3 | architect | ADR, contracts, threat models |
| 4 | qa | Write tests (RED) + validate (GREEN) |
| 4b | qa-cucumber | BDD: extract Gherkin → feature files, generate step stubs, run suite |
| 5 | dotnet | .NET backend implementation |
| 6 | javascript | Node.js / vanilla JS |
| 7 | typescript | TypeScript backend/libraries |
| 8 | react-vite | React + Vite + TypeScript SPA |
| 9 | nextjs | Next.js full-stack |
| 10 | java | Java backend (non-Spring) |
| 11 | springboot | Spring Boot applications |
| 12 | kubernetes | K8s manifests, Kustomize, Helm |
| 13 | terraform | Terraform IaC |
| 14 | docker | Dockerfiles, Compose |
| 15 | postgresql | Schema, migrations, RLS |
| 16 | redis | Caching, pub/sub, streams |
| 17 | supabase | Auth, RLS, Edge Functions |
| 18 | vercel | Deployment, edge, config |
| 19 | stripe | Payments, webhooks, billing |
| 20 | data-science | EDA, stats, visualization |
| 21 | data-engineer | Pipelines, ETL, orchestration |
| 22 | tensorflow | TF/Keras models, training |
| 23 | pytorch | PyTorch models, training |
| 24 | pandas-numpy | Data manipulation, arrays |
| 25 | scikit | Classical ML, pipelines |
| 26 | jupyter | Notebooks, papermill |
| 27 | docs | Feature docs, CHANGELOG, Mermaid |
| 28 | spec-kit | Gherkin story author — writes story file to `docs/stories/`, halts for approval |
| 29 | ux-researcher | Personas, journey maps, research summaries — feeds wireframe-architect |
| 30 | wireframe-architect | Wireframes per `design.target` (web: md+html+excalidraw; tui: md+excalidraw) — requires human approval |
| 31 | visual-identity-designer | Brand palette, typography, spacing — outputs `palette.json` + `tokens.json` |
| 32 | design-system-author | Design token system — writes `tokens.json`, CSS vars, Tailwind config, Mantine theme |
| 33 | ui-mockup-builder | High-fidelity mockups per `design.target` (web: React/HTML; tui: lipgloss md) — requires wireframe approval |
| 34 | a11y-auditor | WCAG 2.2 AA audit — blocks merge on critical/serious violations for `ui:true` stories |
| 35 | rubber-duck | Second-opinion reviewer — surfaces bugs, design flaws, edge cases (no style comments) |

## Pipeline Phases
1. DISCOVER → 2. ARCHITECT → 3. PLAN → 4. INFRA → 5. IMPLEMENT → **[Karpathy Gate]** → 6. VALIDATE → 7. DOCUMENT → 8. FINAL GATE

**[Karpathy Gate]** — After Phase 5 IMPLEMENT, orchestrator auto-calls karpathy-audit to enforce:
- Think Before Coding
- Simplicity First
- Surgical Changes
- Goal-Driven Execution

Score ≥90 auto-advance, 70-89 require approval, <70 BLOCK.

After Phase 7 DOCUMENT, call `/humanizer` to remove AI-isms from prose before merge.

---
All agents use: `make build`, `make test`, `make test-integration`, `make test-e2e`,
`make test-contract`, `make test-all`, `make lint`, `make security-scan`, `make fmt`,
`make containers-up`, `make containers-down`, `make migrate`.

## Git Conventions
- Conventional Commits: `feat:`, `fix:`, `test:`, `docs:`, `infra:`, `refactor:`
- Branch naming: `feat/{slug}`, `fix/{slug}`
- Squash merge to main

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
