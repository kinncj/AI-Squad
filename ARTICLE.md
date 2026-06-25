# Building MAPLE: Orchestrated Multi-Agent Systems with Claude Code and OpenCode

*A practical guide to running an orchestrated, phase-gated, TDD-enforced development pipeline with 27 specialist agents using your GitHub Copilot Enterprise and Claude Code Max subscriptions.*

---

## The Problem with Single-Agent Coding

Single-agent AI coding breaks down at scale. One model, one context window, one session — the moment you ask it to gather requirements, design architecture, write tests, implement code, and document the result, context pollution takes over. The model skips tests. It jumps to "obvious" implementations. It forgets the acceptance criteria it wrote ten minutes ago.

The fix is structural: split the work across specialist agents with enforced boundaries, exactly like a real engineering team. A product owner who never writes code. A QA engineer who never touches production files. A Kubernetes specialist who only handles manifests. An orchestrator who delegates everything and writes nothing.

This article walks through a complete orchestrated multi-agent system with **27 specialist agents** running on two platforms: **Claude Code** (Claude Code Max) and **OpenCode** (GitHub Copilot Enterprise). Same pipeline, same quality gates, same agent prompt bodies — different frontmatter.

---

## The Two Platforms

### Claude Code (Claude Code Max)

Anthropic's terminal-based agentic coding tool. Agent definitions live in `.claude/agents/`, commands in `.claude/commands/`, skills in `.claude/skills/`.

Key characteristics: agents defined with `name`, `description`, `model` in YAML frontmatter. Subagents invoked via `@agent-name`. Commands via `/command-name`. Supports **swarm mode** for parallel execution via Zellij.

### OpenCode (GitHub Copilot Enterprise)

Open-source terminal coding agent supporting multiple LLM providers including GitHub Copilot Enterprise. Agent definitions live in `.opencode/agents/`, commands in `.opencode/commands/`, skills in `.opencode/skills/`.

Key characteristics: richer frontmatter with `model`, `temperature`, `mode`, `tools` (write/edit/bash per agent), and granular `permission` blocks (including `permission.task` to restrict subagent invocation). Explicit model routing per agent using `provider/model-id` format. Project config in `opencode.json`. Subagents run in navigable child sessions.

### License Mapping

| Capability | Claude Code Max | OpenCode (GitHub Copilot Enterprise) |
|---|---|---|
| Orchestrator / Architect | `claude-opus-4-6` | `anthropic/claude-opus-4-6` |
| Implementation agents | `claude-sonnet-4-6` | `github-copilot/claude-sonnet-4.5` |
| Infrastructure agents | `claude-sonnet-4-6` | `copilot/gpt-4.1` |
| Small/fast model | `claude-haiku-4-5-20251001` | `copilot/gpt-5-nano` |

Model IDs follow OpenCode's `provider/model-id` convention. Check your provider's documentation for the latest version strings and update accordingly.

---

## System Architecture

```mermaid
graph TB
    Human["👤 Human"] -->|Feature request| Orch

    subgraph "🎯 Primary Agent"
        Orch["Orchestrator<br/><i>Never writes code</i>"]
    end

    subgraph "📋 Discovery & Design"
        PO["Product Owner"]
        Arch["Architect"]
    end

    subgraph "⚙️ Backend Specialists"
        DotNet[".NET"]
        JS["JavaScript"]
        TS["TypeScript"]
        Java["Java"]
        Spring["Spring Boot"]
        DS["Data Science"]
        DE["Data Engineer"]
    end

    subgraph "🎨 Frontend Specialists"
        ReactVite["React + Vite"]
        NextJS["Next.js"]
    end

    subgraph "🐳 Infrastructure Specialists"
        K8s["Kubernetes"]
        Terra["Terraform"]
        Docker["Docker"]
    end

    subgraph "🗄️ Data & Services Specialists"
        PG["PostgreSQL"]
        Redis["Redis"]
        Supa["Supabase"]
        Vercel["Vercel"]
        Stripe["Stripe"]
    end

    subgraph "🧬 ML/AI Specialists"
        TensorFlow["TensorFlow"]
        PyTorch["PyTorch"]
        Pandas["Pandas/NumPy"]
        Scikit["Scikit-learn"]
        Jupyter["Jupyter"]
    end

    subgraph "🧪 Quality & Docs"
        QA["QA Agent"]
        Docs["Documentation"]
    end

    Orch --> PO
    Orch --> Arch
    Orch --> DotNet
    Orch --> JS
    Orch --> TS
    Orch --> Java
    Orch --> Spring
    Orch --> ReactVite
    Orch --> NextJS
    Orch --> K8s
    Orch --> Terra
    Orch --> Docker
    Orch --> PG
    Orch --> Redis
    Orch --> Supa
    Orch --> Vercel
    Orch --> Stripe
    Orch --> DS
    Orch --> DE
    Orch --> TensorFlow
    Orch --> PyTorch
    Orch --> Pandas
    Orch --> Scikit
    Orch --> Jupyter
    Orch --> QA
    Orch --> Docs
```

---

## The 8-Phase Pipeline

```mermaid
flowchart LR
    P1["Phase 1<br/>DISCOVER<br/><i>@product-owner</i>"]
    P2["Phase 2<br/>ARCHITECT<br/><i>@architect</i>"]
    P3["Phase 3<br/>PLAN<br/><i>Orchestrator</i>"]
    P4["Phase 4<br/>INFRA<br/><i>@docker @k8s @terraform</i>"]
    P5["Phase 5<br/>IMPLEMENT<br/><i>TDD Loop</i>"]
    P6["Phase 6<br/>VALIDATE<br/><i>@qa</i>"]
    P7["Phase 7<br/>DOCUMENT<br/><i>@docs</i>"]
    P8["Phase 8<br/>FINAL GATE<br/><i>Orchestrator</i>"]

    P1 -->|"Human gate"| P2
    P2 -->|"Human gate"| P3
    P3 --> P4
    P4 --> P5
    P5 --> P6
    P6 -->|"100% pass"| P7
    P7 --> P8
    P8 -->|"All green"| Done["✅ Done"]
    P6 -->|"Failure"| P5
    P8 -->|"Failure"| P5
```

Each phase produces artifacts and has a quality gate. No skipping.

| Phase | Agent(s) | Gate | Artifacts |
|---|---|---|---|
| 1. DISCOVER | `@product-owner` | Human approval | `stories.md`, `acceptance-criteria.md` |
| 2. ARCHITECT | `@architect` | Human approval + BusinessRepo check | `architecture.md`, `adr.md`, `contracts/`, `threat-model.md` |
| 3. PLAN | Orchestrator | Tests precede implementation | `plan.md`, `test-plan.md` |
| 4. INFRA | `@docker`, `@k8s`, `@terraform` | Containers healthy | `docker-compose.yml`, manifests, modules |
| 5. IMPLEMENT | `@qa` + specialist agents | All tests pass per task | Source code, test files |
| 6. VALIDATE | `@qa` | 100% pass across all categories | Validation report |
| 7. DOCUMENT | `@docs` | Docs match implementation | Feature docs, CHANGELOG, runbooks |
| 8. FINAL GATE | Orchestrator | `make test-all` exits 0 | Completion summary |

### The TDD Loop (Phase 5)

```mermaid
stateDiagram-v2
    [*] --> RED: QA writes failing test
    RED --> VERIFY_FAIL: Run test
    VERIFY_FAIL --> RED: Test passes — wrong test
    VERIFY_FAIL --> GREEN: Test fails — correct
    GREEN --> VERIFY_PASS: Specialist implements
    VERIFY_PASS --> GREEN: Test fails — fix code
    VERIFY_PASS --> REFACTOR: Test passes
    REFACTOR --> VERIFY_STILL: Clean up
    VERIFY_STILL --> REFACTOR: Test fails
    VERIFY_STILL --> [*]: Passes → next task
```

---

## The Complete Agent Roster

### Model Assignment Strategy

| Agent Category | Model | Reasoning |
|---|---|---|
| Orchestrator, Architect | `anthropic/claude-opus-4-6` (Claude Code: `claude-opus-4-6`) | Highest reasoning for coordination + design |
| Product Owner, QA, Docs | `github-copilot/claude-sonnet-4.5` | Creative + structured output |
| Backend/Frontend specialists | `github-copilot/claude-sonnet-4.5` | Fast, high-quality code gen |
| Infrastructure specialists | `copilot/gpt-4.1` | Strong at config/manifest generation |
| ML/AI specialists | `github-copilot/claude-sonnet-4.5` | Good at Python/ML patterns |

---

### 1. Orchestrator

**Role:** Primary agent. Controls the entire pipeline. Never writes code — enforced via `tools.write=false`, `tools.edit=false` in OpenCode config.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/orchestrator.md` | `claude-opus-4-6` |
| OpenCode | `.opencode/agents/orchestrator.md` | `anthropic/claude-opus-4-6` |

**OpenCode tools:** `write: false`, `edit: false`, `bash: true`. Bash: ask (except `git`, `cat`, `ls`, `find`, `mkdir`, `make`, `gh` = allow). Uses `permission.task` to control which subagents can be invoked.

Owns the 8-phase workflow, TDD loop, quality gates, escalation rules (3 failures = human). Delegates to all specialist agents by task type. Manages issue lifecycle: assigns agents, updates phase labels, posts progress comments.

---

### 2. Product Owner

**Role:** Translates raw feature requests into testable specifications.

| Platform | File | Model | Temperature |
|---|---|---|---|
| Claude Code | `.claude/agents/product-owner.md` | `claude-sonnet-4-6` | — |
| OpenCode | `.opencode/agents/product-owner.md` | `github-copilot/claude-sonnet-4.5` | `0.3` |

**OpenCode tools:** `write: true`, `edit: true`, `bash: false`. Edit: `docs/**` only. Bash allow (read-only): `cat`, `ls`, `gh issue create`, `gh issue list`, `gh issue view`.

Produces: user stories (As a/I want/So that), acceptance criteria (Given/When/Then), definition of done. Covers happy paths, edge cases, error scenarios, non-functional requirements. Never designs solutions. Creates GitHub issues for each story with labels and milestone assignment.

---

### 3. Architect

**Role:** Staff+ system design. ADR, contracts, threat models, diagrams.

| Platform | File | Model | Temperature |
|---|---|---|---|
| Claude Code | `.claude/agents/architect.md` | `claude-opus-4-6` | — |
| OpenCode | `.opencode/agents/architect.md` | `anthropic/claude-opus-4-6` | `0.1` |

**OpenCode tools:** `write: true`, `edit: true`, `bash: false`. Edit: `docs/**` only. Bash allow (read-only): `cat`, `ls`, `find`, `gh issue comment`.

Produces: `architecture.md` (10 required sections + 4 Mermaid diagrams minimum), `adr.md` (RFC format), `contracts/` (OpenAPI, events, DB schema, seed data), `threat-model.md` (STRIDE). Evaluates through BusinessRepo lens. Rejects cross-domain coupling.

---

### 4. QA Agent (Unified)

**Role:** Two modes — write failing tests (Red phase) and validate (full suite).

| Platform | File | Model | Temperature |
|---|---|---|---|
| Claude Code | `.claude/agents/qa.md` | `claude-sonnet-4-6` | — |
| OpenCode | `.opencode/agents/qa.md` | `github-copilot/claude-sonnet-4.5` | `0.05` |

**OpenCode tools:** `write: true`, `edit: true`, `bash: true`. Edit: `tests/**`, `**/*.test.*`, `**/*.spec.*`, `playwright.config.*`, `docker-compose*.yml`, `scripts/seed*`, `Makefile`. Bash: all allowed (needs to run tests and update issues).

Handles 6 test types: unit, integration, E2E browser (Playwright), E2E API, contract, smoke. Stack-specific patterns for xUnit, JUnit, Vitest, pytest, Playwright. Manages container lifecycle and seed data.

```mermaid
graph LR
    QA["🧪 QA Agent"]
    QA --> Unit["Unit<br/><i>xUnit/JUnit/Vitest/pytest</i>"]
    QA --> Int["Integration<br/><i>TestContainers</i>"]
    QA --> E2E["E2E Browser<br/><i>Playwright</i>"]
    QA --> API["E2E API<br/><i>Playwright request</i>"]
    QA --> Contract["Contract<br/><i>Schema validation</i>"]
    QA --> Smoke["Smoke<br/><i>Health checks</i>"]
```

---

### 5. .NET Specialist

**Role:** Implements .NET backend code. Receives failing tests, makes them pass.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/dotnet.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/dotnet.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** .NET 8+, ASP.NET Core, Entity Framework Core, Clean Architecture (Api → Application → Domain → Infrastructure), MediatR, FluentValidation, Result pattern, xUnit + FluentAssertions + TestContainers.

**OpenCode bash:** `dotnet *`, `make *`, `git *`, `docker compose *` allowed.

---

### 6. JavaScript Specialist

**Role:** Implements vanilla JavaScript / Node.js code.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/javascript.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/javascript.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Node.js 22+, Express/Hono/Fastify, Vitest, ESM modules, no TypeScript. For vanilla JS APIs, serverless functions, scripts, and tooling.

**OpenCode bash:** `npm *`, `npx *`, `node *`, `make *`, `git *` allowed.

---

### 7. TypeScript Specialist

**Role:** Implements TypeScript backend/library code.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/typescript.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/typescript.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** TypeScript 5.x strict mode, Zod for validation, Drizzle/Prisma for ORM, Vitest, `bun` or `npm`. No `any` types. Strict `tsconfig.json`.

**OpenCode bash:** `bun *`, `npm *`, `npx *`, `node *`, `make *`, `git *` allowed.

---

### 8. React (Vite + TypeScript) Specialist

**Role:** Implements React SPA frontends with Vite.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/react-vite.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/react-vite.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** React 19+, Vite 6+, TypeScript strict, Tailwind CSS, shadcn/ui + Magic UI, Zod + React Hook Form, Tanstack Router + Query, Zustand for state management. Vitest + Testing Library + axe-core.

**OpenCode bash:** `bun *`, `npm *`, `npx *`, `node *`, `make *`, `git *` allowed.

---

### 9. Next.js Specialist

**Role:** Implements Next.js full-stack applications.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/nextjs.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/nextjs.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Next.js 15+ App Router, Server Components by default, `use client` only when needed, Server Actions for mutations, Tailwind CSS, shadcn/ui, Zod + React Hook Form, Tanstack Query, Playwright for E2E.

**OpenCode bash:** `npm *`, `npx *`, `node *`, `make *`, `git *` allowed.

---

### 10. Java Specialist

**Role:** Implements Java backend code (non-Spring).

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/java.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/java.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Java 21+, virtual threads, records, sealed classes, pattern matching. JUnit 5 + AssertJ + Testcontainers. Maven or Gradle.

**OpenCode bash:** `mvn *`, `gradle *`, `java *`, `make *`, `git *`, `docker compose *` allowed.

---

### 11. Spring Boot Specialist

**Role:** Implements Spring Boot applications.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/springboot.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/springboot.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Spring Boot 3.x, Java 21+, Hexagonal Architecture, Spring Data JPA, Flyway/Liquibase, Spring Security, Spring Cloud, JUnit 5 + Mockito + Testcontainers.

**OpenCode bash:** `mvn *`, `gradle *`, `java *`, `make *`, `git *`, `docker compose *` allowed.

---

### 12. Kubernetes Specialist

**Role:** Creates and manages Kubernetes manifests.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/kubernetes.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/kubernetes.md` | `copilot/gpt-4.1` |

**Stack:** Kubernetes 1.30+, Kustomize overlays (base/dev/staging/prod), Helm charts, resource limits, PodDisruptionBudgets, HPA, NetworkPolicies, RBAC, health probes, cert-manager. `kubectl apply --dry-run=client` for validation.

**OpenCode bash:** `kubectl apply --dry-run*`, `kubectl get *`, `make *`, `git *` allowed. `kubectl apply` (without dry-run) requires ask.

---

### 13. Terraform Specialist

**Role:** Creates and manages Terraform infrastructure-as-code.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/terraform.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/terraform.md` | `copilot/gpt-4.1` |

**Stack:** Terraform 1.9+, modules per environment, state locking, workspaces, `terraform validate` + `terraform plan` for verification. AWS/Azure/GCP provider patterns. FinOps cost annotations.

**OpenCode bash:** `terraform validate`, `terraform plan *`, `terraform fmt *` allowed. `terraform apply` requires ask.

---

### 14. Docker Specialist

**Role:** Creates Dockerfiles, Compose configs, and container orchestration.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/docker.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/docker.md` | `copilot/gpt-4.1` |

**Stack:** Multi-stage builds, `docker compose` with health checks, `.dockerignore`, BuildKit cache mounts, distroless/alpine base images, non-root users, security scanning with Trivy/Grype.

**OpenCode bash:** `docker *`, `docker compose *`, `make *`, `git *` allowed. **OpenCode edit:** `docker-compose*`, `Dockerfile*`, `.dockerignore`, `scripts/**`.

---

### 15. PostgreSQL Specialist

**Role:** Database schema design, migrations, query optimization, RLS policies.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/postgresql.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/postgresql.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** PostgreSQL 16+, DDL migrations (Flyway/EF Core/Prisma/Drizzle), indexing strategy, `EXPLAIN ANALYZE`, partitioning, Row-Level Security, connection pooling (PgBouncer), backup/restore patterns.

**OpenCode bash:** `psql *`, `pg_dump *`, `make *`, `git *` allowed. **OpenCode edit:** `**/migrations/**`, `docs/**`.

---

### 16. Redis Specialist

**Role:** Caching, session management, pub/sub, rate limiting with Redis.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/redis.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/redis.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Redis 7+, Redis Stack (JSON, Search, TimeSeries), Streams, Lua scripting, Sentinel/Cluster, eviction policies, key design patterns, TTL strategies.

**OpenCode bash:** `redis-cli *`, `make *`, `git *` allowed.

---

### 17. Supabase Specialist

**Role:** Implements Supabase-backed applications.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/supabase.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/supabase.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Supabase Auth (email, OAuth, MFA), Row-Level Security on all tables, Edge Functions (Deno runtime), Realtime subscriptions, Storage buckets, database migrations in `supabase/migrations/`, local dev with `supabase start`.

**OpenCode bash:** `supabase *`, `deno *`, `make *`, `git *` allowed.

---

### 18. Vercel Specialist

**Role:** Deployment, edge functions, and platform configuration for Vercel.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/vercel.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/vercel.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** `vercel.json` config, Edge/Serverless/ISR runtime selection, environment variables, preview deployments, Vercel KV/Blob/Postgres, cron jobs, middleware, headers/redirects, framework auto-detection.

**OpenCode bash:** `vercel *`, `npm *`, `make *`, `git *` allowed.

---

### 19. Stripe Specialist

**Role:** Payment integration, webhooks, billing flows.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/stripe.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/stripe.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Stripe API (latest), webhook signature verification on ALL endpoints, idempotency keys, restricted API keys per service, Checkout Sessions, Payment Intents, Subscriptions, Customer Portal, Stripe CLI for local testing (`stripe listen --forward-to`).

**OpenCode bash:** `stripe *`, `make *`, `git *` allowed.

---

### 20. Data Science Specialist

**Role:** Exploratory data analysis, statistical modeling, visualization.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/data-science.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/data-science.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Python 3.12+, pandas, NumPy, scikit-learn, matplotlib/seaborn/plotly, statsmodels, SciPy. Focus on EDA, feature engineering, statistical testing, visualization best practices.

**OpenCode bash:** `python *`, `pip *`, `pytest *`, `jupyter *`, `make *`, `git *` allowed.

---

### 21. Data Engineer Specialist

**Role:** Data pipelines, ETL/ELT, data modeling, orchestration.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/data-engineer.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/data-engineer.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** Python, SQL, dbt, Apache Airflow/Dagster, Spark (PySpark), data lake patterns (Delta Lake/Iceberg), schema evolution, data quality (Great Expectations), dimensional modeling.

**OpenCode bash:** `python *`, `pip *`, `dbt *`, `pytest *`, `make *`, `git *`, `docker compose *` allowed.

---

### 22. TensorFlow Specialist

**Role:** Implements TensorFlow/Keras ML models and training pipelines.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/tensorflow.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/tensorflow.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** TensorFlow 2.x, Keras 3, `tf.data` pipelines, mixed precision training, TensorBoard, SavedModel export, TFLite conversion, TF Serving. GPU memory management, distributed training with `tf.distribute`.

**OpenCode bash:** `python *`, `pip *`, `pytest *`, `make *`, `git *` allowed.

---

### 23. PyTorch Specialist

**Role:** Implements PyTorch ML models and training loops.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/pytorch.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/pytorch.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** PyTorch 2.x, `torch.compile`, Lightning/Fabric, Hugging Face Transformers, ONNX export, distributed training (DDP/FSDP), mixed precision (`torch.amp`), custom datasets and dataloaders.

**OpenCode bash:** `python *`, `pip *`, `pytest *`, `make *`, `git *` allowed.

---

### 24. Pandas/NumPy Specialist

**Role:** Data manipulation, numerical computing, array operations.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/pandas-numpy.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/pandas-numpy.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** pandas 2.x (Arrow backend), NumPy 2.x, vectorized operations over loops, method chaining, `pd.eval`/`query`, memory-efficient dtypes, chunked processing for large datasets.

**OpenCode bash:** `python *`, `pip *`, `pytest *`, `make *`, `git *` allowed.

---

### 25. Scikit-learn Specialist

**Role:** Classical ML: classification, regression, clustering, preprocessing.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/scikit.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/scikit.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** scikit-learn 1.x, Pipeline + ColumnTransformer, cross-validation, hyperparameter tuning (GridSearchCV/Optuna), model persistence (joblib), feature selection, metrics/evaluation, SHAP for interpretability.

**OpenCode bash:** `python *`, `pip *`, `pytest *`, `make *`, `git *` allowed.

---

### 26. Jupyter Notebook Specialist

**Role:** Creates and manages Jupyter notebooks for analysis and experimentation.

| Platform | File | Model |
|---|---|---|
| Claude Code | `.claude/agents/jupyter.md` | `claude-sonnet-4-6` |
| OpenCode | `.opencode/agents/jupyter.md` | `github-copilot/claude-sonnet-4.5` |

**Stack:** JupyterLab/Notebook, `nbformat` for programmatic creation, `papermill` for parameterized execution, `nbconvert` for export (HTML/PDF), cell organization patterns (setup → data → analysis → visualization → conclusions), clean notebook hygiene (restart & run all).

**OpenCode bash:** `python *`, `pip *`, `jupyter *`, `papermill *`, `make *`, `git *` allowed.

---

### 27. Documentation Agent

**Role:** Technical documentation with Mermaid diagrams.

| Platform | File | Model | Temperature |
|---|---|---|---|
| Claude Code | `.claude/agents/docs.md` | `claude-sonnet-4-6` | — |
| OpenCode | `.opencode/agents/docs.md` | `github-copilot/claude-sonnet-4.5` | `0.2` |

**OpenCode tools:** `write: true`, `edit: true`, `bash: false`. Edit: `docs/**`, `README.md`, `CHANGELOG.md`, `**/*.md`. Bash allow (read-only): `cat`, `ls`, `find`, `git log`, `git diff`, `gh issue comment`.

Produces: feature documentation, CHANGELOG entries, runbooks, Mermaid diagrams (component, sequence, ER, state, flowchart). Reads implementation before documenting — never documents from spec.

---

## Agent Interaction Sequence

```mermaid
sequenceDiagram
    actor H as Human
    participant O as Orchestrator
    participant PO as Product Owner
    participant A as Architect
    participant D as Docker
    participant PG as PostgreSQL
    participant QA as QA Agent
    participant BE as .NET / Spring / TS
    participant FE as React / Next.js
    participant Docs as Docs

    H->>O: /feature user registration with OAuth

    Note over O: Phase 1: DISCOVER
    O->>PO: Decompose into stories
    PO-->>O: stories.md + acceptance-criteria.md
    O->>H: Review?
    H-->>O: Approved ✅

    Note over O: Phase 2: ARCHITECT
    O->>A: Design architecture
    A-->>O: ADR + contracts + threat model
    O->>H: Review?
    H-->>O: Approved ✅

    Note over O: Phase 3: PLAN
    O->>O: Create plan.md + test-plan.md

    Note over O: Phase 4: INFRA
    O->>D: Create docker-compose + Dockerfile
    O->>PG: Create schema + seed data
    D-->>O: Containers healthy ✅
    PG-->>O: Schema migrated ✅

    Note over O: Phase 5: IMPLEMENT
    loop For each task
        O->>QA: Write failing test (RED)
        QA-->>O: Test fails ✅
        O->>BE: Make test pass (GREEN)
        BE-->>O: Implementation done
        O->>QA: Verify
        QA-->>O: ✅ PASS
    end

    O->>QA: Playwright E2E tests
    O->>FE: Fix E2E failures
    QA-->>O: All E2E pass ✅

    Note over O: Phase 6: VALIDATE
    O->>QA: Full suite
    QA-->>O: All green ✅

    Note over O: Phase 7: DOCUMENT
    O->>Docs: Document feature
    Docs-->>O: Docs + CHANGELOG ✅

    Note over O: Phase 8: FINAL GATE
    O->>O: make test-all → ✅
    O->>H: Complete ✅
```

### GitHub Issue Lifecycle

Every feature flows through GitHub issues. The `gh` CLI is the interface — no browser context switching, no MCP overhead.

```mermaid
stateDiagram-v2
    [*] --> Created: PO runs gh issue create
    Created --> Triaged: Orchestrator adds phase:discover label
    Triaged --> InProgress: Orchestrator adds in-progress label

    state InProgress {
        [*] --> Discover
        Discover --> Architect: PO comments stories done
        Architect --> Plan: Architect comments design done
        Plan --> Implement: Orchestrator updates to phase:implement
        Implement --> Validate: All TDD loops green
        Validate --> Document: QA comments validation passed
        Document --> FinalGate: Docs comments docs complete
    }

    InProgress --> Blocked: Agent hits 3 failures
    Blocked --> InProgress: Human unblocks

    InProgress --> ReadyForReview: Orchestrator creates PR, adds label
    ReadyForReview --> Validated: QA closes issue
    Validated --> [*]: gh pr merge --squash
```

Each state transition is a `gh issue edit` or `gh issue comment` call — a small payload (tens of tokens, varies by body length).

---

## CLI + Skills First, MCP Only When Necessary

### Why Skills Over MCP

MCP servers inject tool schemas, accessibility trees, and verbose response payloads into the agent's context window on every call. For a typical browser automation task, Playwright MCP consumes ~114,000 tokens versus ~27,000 tokens with CLI — a 4x reduction. That's context budget that could hold your entire codebase, test suite, and reasoning traces.

The principle: **if the agent has filesystem and shell access (which both Claude Code and OpenCode do), use CLI tools exposed as Skills.** Reserve MCP for situations where the agent lacks shell access or needs persistent state across a structured protocol.

```mermaid
flowchart TB
    Decision{"Does the agent<br/>have shell access?"}
    Decision -->|Yes| CLI["✅ CLI + Skills<br/><i>Token-efficient, file-based,<br/>stateless commands</i>"]
    Decision -->|No| MCP["MCP Server<br/><i>Structured protocol,<br/>persistent state</i>"]

    CLI --> PW_CLI["Playwright CLI<br/><code>playwright-cli snapshot</code><br/><code>playwright-cli click e21</code>"]
    CLI --> Native["Native CLIs<br/><code>dotnet test</code><br/><code>kubectl apply</code><br/><code>terraform plan</code>"]
    CLI --> Third["Third-party CLIs<br/><code>stripe listen</code><br/><code>supabase start</code><br/><code>vercel deploy</code>"]

    MCP --> C7["Context7 MCP<br/><i>Doc lookup — low token cost,<br/>high value per call</i>"]
```

### What Stays as MCP: Context7

Context7 is the one MCP server we keep. Its tool calls are lightweight (a library ID + query in, a focused doc snippet out) and the value-per-token ratio is excellent — it eliminates hallucinated APIs and outdated patterns in a single call.

**Package:** `@upstash/context7-mcp`
**Tools:** `resolve-library-id`, `get-library-docs`

```bash
# Claude Code setup
claude mcp add context7 -- npx -y @upstash/context7-mcp@latest

# OpenCode setup (opencode.json)
{
  "mcp": {
    "context7": {
      "type": "local",
      "command": ["npx", "-y", "@upstash/context7-mcp@latest"],
      "enabled": true
    }
  }
}
```

Usage: agents append `use context7` to prompts or reference a library ID directly (`use library /supabase/supabase`). The Orchestrator instructs implementation agents to use Context7 when working with unfamiliar library versions.

Available libraries: Next.js, React, Supabase, Stripe, Playwright, TensorFlow, PyTorch, scikit-learn, pandas, FastAPI, Spring Boot, ASP.NET Core, Prisma, Drizzle, Tailwind CSS, shadcn/ui, and thousands more.

### What Moves to CLI + Skills: Playwright

For coding agents, we prefer Playwright CLI + file-based snapshots because it avoids injecting large accessibility trees and screenshots into the model context. The `@playwright/cli` package saves snapshots and screenshots to disk instead of injecting them into context. The agent decides what to read.

**Install:**

```bash
npm install -g @playwright/cli
npx playwright install  # install browsers
```

**CLI commands the QA agent uses:**

```bash
# Browser exploration (saves YAML snapshot to disk, not context)
playwright-cli open http://localhost:3000
playwright-cli snapshot                    # compact YAML with element refs (e21, e35)
playwright-cli click e21                   # click by element reference
playwright-cli fill e35 "test@example.com" # fill input
playwright-cli screenshot                  # saves to disk, not context

# Session management
playwright-cli --session=auth open http://localhost:3000/login
playwright-cli session-list

# Test execution (standard Playwright runner)
npx playwright test tests/e2e/{feature}.spec.ts
npx playwright test --ui                   # visual debugger
npx playwright test --headed               # visible browser
npx playwright codegen http://localhost:3000  # record → generate test
```

**The Playwright Skill** (`.claude/skills/playwright-cli/SKILL.md` / `.opencode/skills/playwright-cli/SKILL.md`) teaches agents the CLI commands, element reference patterns, and snapshot-to-test workflow — without loading MCP tool schemas into context on every turn.

### What Was Always CLI: Everything Else

Every other tool in the system is a native CLI invoked through bash. No MCP overhead. No tool schema injection. Just shell commands.

| Category | CLI Tools | Key Commands |
|---|---|---|
| **.NET** | `dotnet` | `dotnet build`, `dotnet test --filter "Category=Unit"`, `dotnet ef database update`, `dotnet format --verify-no-changes` |
| **Java/Spring** | `mvn`, `gradle` | `mvn test -Dtest=ClassName`, `mvn spring-boot:run`, `gradle test` |
| **TypeScript/JS** | `bun`, `npm`, `npx`, `node` | `bun test`, `npm run build`, `npx vitest run`, `npx tsc --noEmit` |
| **React/Next.js** | `npm`, `npx` | `npm run dev`, `npx next build`, `npx next lint` |
| **Python/ML** | `python`, `pip`, `pytest` | `pytest -v`, `python -m jupyter nbconvert`, `pip install -e .` |
| **Docker** | `docker`, `docker compose` | `docker compose up -d --wait`, `docker compose down -v`, `docker build --target production` |
| **Kubernetes** | `kubectl`, `helm`, `kustomize` | `kubectl apply --dry-run=client -f`, `kubectl get pods`, `helm template`, `kustomize build` |
| **Terraform** | `terraform` | `terraform validate`, `terraform plan -out=plan.tfplan`, `terraform fmt -check` |
| **PostgreSQL** | `psql`, `pg_dump` | `psql -c "EXPLAIN ANALYZE ..."`, `pg_dump --schema-only` |
| **Redis** | `redis-cli` | `redis-cli ping`, `redis-cli info memory`, `redis-cli monitor` |
| **Supabase** | `supabase` | `supabase start`, `supabase db push`, `supabase functions deploy`, `supabase gen types typescript` |
| **Vercel** | `vercel` | `vercel build`, `vercel deploy --prebuilt`, `vercel env pull .env.local` |
| **Stripe** | `stripe` | `stripe listen --forward-to localhost:3000/api/webhooks`, `stripe trigger payment_intent.succeeded`, `stripe logs tail` |
| **Jupyter** | `jupyter`, `papermill` | `jupyter nbconvert --execute notebook.ipynb`, `papermill input.ipynb output.ipynb -p param value` |
| **Quality** | `make` | `make test`, `make lint`, `make security-scan`, `make test-all` |
| **GitHub** | `gh` | `gh issue create --title "..." --body "..." --label "..."`, `gh issue edit 23 --add-label "in-progress"`, `gh pr create --title "..." --base main`, `gh pr merge --squash` |
| **Git** | `git` | `git add -A`, `git commit -m "feat: ..."`, `git diff --staged` |

### Token Budget Comparison

These numbers are from a representative 20-step browser automation flow; results vary by page complexity and tool verbosity. The relative magnitude (~4x CLI advantage) is consistent across measurements.

| Approach | Tokens per browser interaction | Tokens for 20-step flow |
|---|---|---|
| Playwright MCP | ~5,700 (full accessibility tree + screenshot) | ~114,000 |
| Playwright CLI | ~1,350 (file path + compact status) | ~27,000 |
| Native CLI (`dotnet test`) | ~200 (exit code + summary) | ~4,000 |
| Context7 MCP | ~800 (focused doc snippet) | N/A (on-demand) |

The savings compound over a feature lifecycle. Over many tasks, MCP-style browser interactions can consume millions of tokens of context in UI telemetry alone, while CLI + file-based snapshots typically stay in the hundreds of thousands — freeing substantial context budget for code, tests, and reasoning.

---

## Skills

Skills are reusable knowledge packages (markdown files) that agents read before executing. They're loaded on-demand, not injected into every turn like MCP tool schemas. This is why skills are fundamentally more token-efficient than MCP: the agent reads a skill file once, internalizes the patterns, then executes CLI commands that return compact output.

```mermaid
graph LR
    subgraph "Process Skills (How)"
        TDD["tdd-workflow"]
        RFC["rfc-adr"]
        TM["threat-modeling"]
    end

    subgraph "Output Skills (What)"
        Mermaid["mermaid-diagrams"]
        FinOps["finops-review"]
        SRE["sre-review"]
    end

    subgraph "Tool Skills (With)"
        PW["playwright-cli"]
        Docker_S["docker-patterns"]
        K8s_S["kubernetes-patterns"]
        TF_S["terraform-patterns"]
        Supa_S["supabase-patterns"]
        Stripe_S["stripe-patterns"]
        Vercel_S["vercel-patterns"]
        PG_S["postgresql-patterns"]
        Redis_S["redis-patterns"]
        Jupyter_S["jupyter-patterns"]
        GH_S["github-cli"]
    end
```

### Process Skills

#### Skill 1: TDD Workflow

**File:** `skills/tdd-workflow/SKILL.md`

Defines TDD (RED → GREEN → REFACTOR) and ATDD (acceptance criteria → test → implement) patterns. Container lifecycle for integration tests (`docker compose up -d --wait` → run tests → `docker compose down -v`). Rules: test file before implementation, test must fail initially, never weaken assertions, implementation agent receives test path not requirement text.

#### Skill 2: RFC/ADR

**File:** `skills/rfc-adr/SKILL.md`

Architecture Decision Record format: Title, Context, Goals/Non-goals, Proposal, Alternatives Considered, Trade-offs and Risks, Impact (cost, ops, security, team), Decision, Next Steps. Must include FinOps cost impact section and SRE operability section.

#### Skill 3: Threat Modeling

**File:** `skills/threat-modeling/SKILL.md`

STRIDE framework: Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege. Identifies assets and trust boundaries. Maps threats to components with risk levels and mitigations. Output to `docs/specs/{feature}/threat-model.md`.

### Output Skills

#### Skill 4: Mermaid Diagrams

**File:** `skills/mermaid-diagrams/SKILL.md`

Required diagrams per feature: component (`graph TB`), sequence (`sequenceDiagram`), ER (`erDiagram`), deployment, state machine (`stateDiagram-v2`). 30-node limit for readability. Syntax reference for all 5 diagram types.

#### Skill 5: FinOps Review

**File:** `skills/finops-review/SKILL.md`

Cost driver identification, scaling characteristics, cost visibility. Evaluates cloud spend implications of architectural decisions. Annotates Terraform modules with estimated cost ranges.

#### Skill 6: SRE Review

**File:** `skills/sre-review/SKILL.md`

Failure modes, blast radius, observability (metrics/logs/traces/alerts), recovery plan, runbook requirements. Evaluates operational readiness of new features.

### Tool Skills (CLI Wrappers)

These skills teach agents how to use CLI tools effectively — the commands, flags, output patterns, and common workflows. The agent reads the skill once, then issues lightweight bash commands.

#### Skill 7: Playwright CLI

**File:** `skills/playwright-cli/SKILL.md`

Teaches the QA agent `playwright-cli` commands: `open`, `snapshot`, `click`, `fill`, `press`, `screenshot`, `session-list`. Element reference patterns (e.g., `e21`, `e35` from YAML snapshots). Workflow: snapshot → identify elements → interact → assert → promote to test file. Also covers `npx playwright test`, `--ui`, `--headed`, `codegen` for test generation.

#### Skill 8: Docker Patterns

**File:** `skills/docker-patterns/SKILL.md`

Multi-stage Dockerfile patterns (development/production targets). `docker compose` health check patterns. `.dockerignore` best practices. BuildKit cache mounts. Distroless/alpine base image selection. Non-root user setup. `docker compose -f docker-compose.test.yml up -d --wait` for test infra.

#### Skill 9: Kubernetes Patterns

**File:** `skills/kubernetes-patterns/SKILL.md`

Kustomize overlay structure (base/dev/staging/prod). Resource limits and requests. PodDisruptionBudgets. HPA configuration. NetworkPolicies. RBAC templates. Health probe patterns (startup/liveness/readiness). `kubectl apply --dry-run=client` validation. `kustomize build overlays/dev | kubectl apply -f -`.

#### Skill 10: Terraform Patterns

**File:** `skills/terraform-patterns/SKILL.md`

Module structure per environment. State locking patterns. Variable and output conventions. `terraform validate` + `terraform plan -out=plan.tfplan` workflow. Provider-specific patterns (AWS/Azure/GCP). Cost estimation annotations. `terraform fmt -check` for CI.

#### Skill 11: Supabase Patterns

**File:** `skills/supabase-patterns/SKILL.md`

Local dev workflow: `supabase start` → `supabase db push` → `supabase gen types typescript`. RLS policy patterns. Edge Function patterns (Deno runtime). Auth configuration (email, OAuth, MFA). Migration workflow. `supabase functions deploy` for production.

#### Skill 12: Stripe Patterns

**File:** `skills/stripe-patterns/SKILL.md`

Local testing: `stripe listen --forward-to localhost:3000/api/webhooks`. Event triggering: `stripe trigger payment_intent.succeeded`. Webhook signature verification patterns. Idempotency key usage. Checkout Session flow. Subscription lifecycle. `stripe logs tail` for debugging.

#### Skill 13: Vercel Patterns

**File:** `skills/vercel-patterns/SKILL.md`

`vercel.json` configuration patterns. Runtime selection (Edge/Serverless/ISR). Environment variable management: `vercel env pull .env.local`. Preview deployment workflow. Vercel KV/Blob/Postgres integration. Middleware patterns. `vercel build` + `vercel deploy --prebuilt` for CI.

#### Skill 14: PostgreSQL Patterns

**File:** `skills/postgresql-patterns/SKILL.md`

Migration patterns per ORM (Flyway/EF Core/Prisma/Drizzle). Indexing strategy (B-tree, GIN, GiST). `EXPLAIN ANALYZE` interpretation. Partitioning patterns. Row-Level Security policy templates. Connection pooling (PgBouncer). `pg_dump --schema-only` for schema extraction.

#### Skill 15: Redis Patterns

**File:** `skills/redis-patterns/SKILL.md`

Key design conventions. Cache-aside pattern. Session store pattern. Rate limiting with sorted sets. Pub/sub patterns. Redis Streams for event sourcing. TTL strategies. `redis-cli monitor` for debugging. Sentinel/Cluster topology.

#### Skill 16: Jupyter Patterns

**File:** `skills/jupyter-patterns/SKILL.md`

Notebook structure (setup → data loading → EDA → modeling → visualization → conclusions). `nbformat` for programmatic creation. `papermill` for parameterized execution: `papermill input.ipynb output.ipynb -p dataset "train.csv"`. `nbconvert` for export. Clean notebook hygiene (restart & run all before commit).

#### Skill 17: GitHub CLI

**File:** `skills/github-cli/SKILL.md`

Teaches agents how to manage GitHub issues, PRs, labels, milestones, and workflows using the `gh` CLI. This is the backbone of the MAPLE team's project management — agents create and update issues as they progress through the pipeline.

**Issue lifecycle managed by agents:**

```
PO creates issues → Orchestrator assigns & labels → Agents update status → QA closes on pass
```

**Key commands by agent role:**

| Agent | Command | Example |
|---|---|---|
| Product Owner | `gh issue create` | `gh issue create --title "Story: User registration" --body-file docs/specs/user-reg/stories.md --label "story,must-have" --milestone "v1.0"` |
| Orchestrator | `gh issue edit` | `gh issue edit 42 --add-label "in-progress" --add-assignee "@me"` |
| Orchestrator | `gh issue comment` | `gh issue comment 42 --body "Phase 5: TDD loop started. 3/7 tasks complete."` |
| QA | `gh issue close` | `gh issue close 42 --comment "All acceptance criteria passing. Validation report attached."` |
| Backend/Frontend | `gh issue comment` | `gh issue comment 42 --body "Task 3 GREEN: tests/unit/UserRegistrationTests.cs passing."` |
| Docs | `gh issue comment` | `gh issue comment 42 --body "Documentation complete: docs/features/user-registration.md"` |
| Any agent | `gh issue list` | `gh issue list --label "in-progress" --assignee "@me"` |
| Any agent | `gh issue view` | `gh issue view 42` |

**PR workflow:**

```bash
# Create feature branch and PR
git checkout -b feat/user-registration
gh pr create --title "feat: user registration with OAuth" \
  --body "Closes #42" --base main --draft

# Update PR when ready
gh pr ready
gh pr edit --add-label "ready-for-review" --add-reviewer teamlead

# Merge (squash, conventional commit)
gh pr merge --squash --subject "feat: user registration with OAuth (#42)"
```

**Label conventions the skill enforces:**

| Label | Applied by | Meaning |
|---|---|---|
| `story` | PO | User story issue |
| `bug` | PO / Human | Bug report |
| `must-have`, `should-have`, `could-have` | PO | MoSCoW priority |
| `phase:discover`, `phase:architect`, `phase:implement`, etc. | Orchestrator | Current pipeline phase |
| `in-progress` | Orchestrator | Work started |
| `blocked` | Any agent | Blocked, needs human |
| `ready-for-review` | Orchestrator | PR ready for human review |
| `validated` | QA | Full test suite passing |

**Setup:**

```bash
# Authenticate (one-time)
gh auth login

# Bootstrap labels for a new repo
gh label create "story" --color "0E8A16" --description "User story"
gh label create "must-have" --color "B60205" --description "MoSCoW: Must"
gh label create "should-have" --color "FBCA04" --description "MoSCoW: Should"
gh label create "could-have" --color "C5DEF5" --description "MoSCoW: Could"
gh label create "phase:discover" --color "D4C5F9"
gh label create "phase:architect" --color "D4C5F9"
gh label create "phase:implement" --color "D4C5F9"
gh label create "phase:validate" --color "D4C5F9"
gh label create "in-progress" --color "FBCA04"
gh label create "blocked" --color "B60205"
gh label create "validated" --color "0E8A16"
gh label create "ready-for-review" --color "0E8A16"
```

**Extended labels (phase granularity + agent tracking):**

```bash
# Per-phase labels (more granular than above)
gh label create "phase:plan"       --color "bfd4f2" --description "Phase 3: Task planning"
gh label create "phase:infra"      --color "f9d0c4" --description "Phase 4: Infrastructure"
gh label create "phase:implement"  --color "fef2c0" --description "Phase 5: TDD implementation"
gh label create "phase:document"   --color "c2e0c6" --description "Phase 7: Documentation"
gh label create "phase:done"       --color "0e8a16" --description "Phase 8: Final gate passed"

# Agent assignment labels
gh label create "agent:backend"    --color "1d76db"
gh label create "agent:frontend"   --color "5319e7"
gh label create "agent:qa"         --color "e11d48"
gh label create "agent:infra"      --color "f97316"

# TDD state labels
gh label create "tdd:red"          --color "b60205" --description "Failing test written"
gh label create "tdd:green"        --color "0e8a16" --description "Test passing"
```

**Workflow run monitoring (used by QA during validation):**

```bash
gh run list --workflow=ci.yml --limit 5
gh run view <run-id>
gh run watch <run-id>    # live stream logs
gh pr checks 99          # check CI status on PR
```

**Agent-to-issue mapping** — the Orchestrator creates one issue per task in Phase 3, labels it with the responsible agent, and each agent updates the issue as it progresses through the TDD loop:

```mermaid
stateDiagram-v2
    [*] --> Created: PO creates issue
    Created --> Planned: Orchestrator adds plan
    Planned --> Red: QA writes failing test
    Red --> Green: Impl agent passes test
    Green --> Validated: QA confirms pass
    Validated --> Documented: Docs agent writes docs
    Documented --> Closed: Final gate passes
```

---

## Commands

| Command | Platform | Description |
|---|---|---|
| `/feature {desc}` | Both (Claude Code alias: `/build-feature`) | Full 8-phase lifecycle |
| `/bugfix {desc}` | Both | Abbreviated: reproduce → fix → validate → CHANGELOG |
| `/validate` | Both | Run full test suite (skip discovery/architecture) |
| `/tdd {requirement}` | Both | Single RED → GREEN → REFACTOR cycle |

---

## The Makefile Contract

```mermaid
graph LR
    subgraph "Build"
        B[make build]
    end

    subgraph "Test Pyramid"
        TU[make test]
        TI[make test-integration]
        TE[make test-e2e]
        TC[make test-contract]
        TA[make test-all]
    end

    subgraph "Quality"
        L[make lint]
        SS[make security-scan]
        F[make fmt]
    end

    subgraph "Infrastructure"
        CU[make containers-up]
        CD[make containers-down]
        M[make migrate]
    end

    TA --> TU
    TA --> TI
    TA --> TE
    TA --> TC
```

All 12 targets are required. CI/CD, the Orchestrator, and QA all use the same `make` interface.

---

## OpenCode Permission Model

OpenCode enforces agent boundaries through two mechanisms: `tools` (write/edit/bash toggles per agent) and `permission` (granular bash command patterns). The Orchestrator's "never writes code" rule is not a social convention — it's `tools.write=false, tools.edit=false` in frontmatter.

```mermaid
graph TB
    subgraph "No Write/Edit — tools.write=false, tools.edit=false"
        Orch_P["Orchestrator<br/><i>bash only: git, make, gh</i>"]
    end

    subgraph "Docs Only — tools.write=true, tools.bash=false"
        PO_P["Product Owner<br/><i>docs/**</i>"]
        A_P["Architect<br/><i>docs/**</i>"]
        D_P["Docs Agent<br/><i>docs/**, README, CHANGELOG</i>"]
    end

    subgraph "Tests + Bash — tools.bash=true"
        QA_P["QA<br/><i>tests/**, *.test.*, *.spec.*</i>"]
    end

    subgraph "Infra Only — tools.bash=true"
        K8s_P["Kubernetes<br/><i>infra/**, k8s/**</i>"]
        TF_P["Terraform<br/><i>infra/**</i>"]
        Docker_P["Docker<br/><i>docker-compose*, Dockerfile*</i>"]
    end

    subgraph "Full Edit — TDD Contract + tools.bash=true"
        DotNet_P[".NET"]
        JS_P["JavaScript"]
        TS_P["TypeScript"]
        Java_P["Java"]
        Spring_P["Spring Boot"]
        React_P["React"]
        Next_P["Next.js"]
        DS_P["Data Science / ML"]
    end
```

**How it's enforced:**

| Mechanism | What it controls | Example |
|---|---|---|
| `tools.write` | Can the agent create new files? | Orchestrator: `false` |
| `tools.edit` | Can the agent modify existing files? | Orchestrator: `false` |
| `tools.bash` | Can the agent run shell commands? | PO, Architect, Docs: `false` (with specific overrides) |
| `permission.bash` | Per-command bash allow/ask/deny patterns | `"terraform apply*": "ask"`, `"kubectl apply --dry-run*": "allow"` |
| `permission.task` | Which subagents can this agent invoke? | Orchestrator: all 26; Specialists: none |
| TDD contract | Behavioral constraint (not config) | Specialists only write code to pass a specific failing test |
| `make test-all` | CI gate (the real enforcement) | Nothing merges without 100% pass |

Path-scoping edits (e.g., "QA can only edit `tests/**`") is a prompt convention, not a native OpenCode config feature. The `tools.edit` toggle is binary (on/off). For hard path enforcement, use branch protection rules and CI checks.

---

## File Structure

```
{domain-name}/
├── AGENTS.md
├── CLAUDE.md                          # Claude Code project rules
├── opencode.json                      # OpenCode project config
├── Makefile
├── docker-compose.test.yml
├── playwright.config.ts
│
├── .claude/
│   ├── agents/                        # 27 agent .md files
│   │   ├── orchestrator.md
│   │   ├── product-owner.md
│   │   ├── architect.md
│   │   ├── qa.md
│   │   ├── dotnet.md
│   │   ├── javascript.md
│   │   ├── typescript.md
│   │   ├── react-vite.md
│   │   ├── nextjs.md
│   │   ├── java.md
│   │   ├── springboot.md
│   │   ├── kubernetes.md
│   │   ├── terraform.md
│   │   ├── docker.md
│   │   ├── postgresql.md
│   │   ├── redis.md
│   │   ├── supabase.md
│   │   ├── vercel.md
│   │   ├── stripe.md
│   │   ├── data-science.md
│   │   ├── data-engineer.md
│   │   ├── tensorflow.md
│   │   ├── pytorch.md
│   │   ├── pandas-numpy.md
│   │   ├── scikit.md
│   │   ├── jupyter.md
│   │   └── docs.md
│   ├── commands/
│   │   ├── feature.md
│   │   ├── build-feature.md             # Alias → feature.md
│   │   ├── bugfix.md
│   │   ├── validate.md
│   │   └── tdd.md
│   └── skills/
│       ├── tdd-workflow/SKILL.md
│       ├── mermaid-diagrams/SKILL.md
│       ├── rfc-adr/SKILL.md
│       ├── threat-modeling/SKILL.md
│       ├── finops-review/SKILL.md
│       ├── sre-review/SKILL.md
│       ├── playwright-cli/SKILL.md
│       ├── docker-patterns/SKILL.md
│       ├── kubernetes-patterns/SKILL.md
│       ├── terraform-patterns/SKILL.md
│       ├── supabase-patterns/SKILL.md
│       ├── stripe-patterns/SKILL.md
│       ├── vercel-patterns/SKILL.md
│       ├── postgresql-patterns/SKILL.md
│       ├── redis-patterns/SKILL.md
│       ├── jupyter-patterns/SKILL.md
│       └── github-cli/SKILL.md
│
├── .opencode/                         # Mirrored structure
│   ├── agents/                        # 27 agent .md files (OpenCode frontmatter + tools config)
│   ├── commands/
│   └── skills/
│
├── app/
│   ├── backend/
│   └── frontend/
├── common/
├── infra/
├── tests/
│   ├── unit/
│   ├── integration/
│   ├── e2e/
│   ├── contract/
│   └── smoke/
├── docs/
│   ├── specs/{feature-slug}/
│   │   ├── stories.md
│   │   ├── acceptance-criteria.md
│   │   ├── architecture.md
│   │   ├── adr.md
│   │   ├── threat-model.md
│   │   ├── plan.md
│   │   ├── test-plan.md
│   │   └── contracts/
│   ├── features/
│   ├── architecture/
│   └── runbooks/
├── infra/
│   └── scripts/
├── scripts/
│   └── maple
└── CHANGELOG.md
```

---

## How the Orchestrator Routes to Specialists

The Orchestrator selects the right specialist based on task type, file path, and technology stack:

```mermaid
flowchart TB
    Task["Incoming Task"]

    Task --> IsTest{Test task?}
    IsTest -->|Yes| QA["@qa"]
    IsTest -->|No| IsInfra{Infrastructure?}

    IsInfra -->|Docker/Compose| Docker["@docker"]
    IsInfra -->|K8s manifests| K8s["@kubernetes"]
    IsInfra -->|Terraform| TF["@terraform"]
    IsInfra -->|No| IsBackend{Backend?}

    IsBackend -->|.cs files| DotNet["@dotnet"]
    IsBackend -->|.java + Spring| Spring["@springboot"]
    IsBackend -->|.java (other)| Java["@java"]
    IsBackend -->|.ts backend| TS["@typescript"]
    IsBackend -->|.js backend| JS["@javascript"]
    IsBackend -->|No| IsFrontend{Frontend?}

    IsFrontend -->|Next.js project| Next["@nextjs"]
    IsFrontend -->|React + Vite| React["@react-vite"]
    IsFrontend -->|No| IsData{Data/ML?}

    IsData -->|Notebook| Jupyter["@jupyter"]
    IsData -->|Pipeline/ETL| DE["@data-engineer"]
    IsData -->|EDA/Stats| DS["@data-science"]
    IsData -->|TensorFlow| TF_ML["@tensorflow"]
    IsData -->|PyTorch| PT["@pytorch"]
    IsData -->|Pandas/NumPy| PD["@pandas-numpy"]
    IsData -->|Scikit-learn| SK["@scikit"]
    IsData -->|No| IsService{Service?}

    IsService -->|Database| PG["@postgresql"]
    IsService -->|Cache| Redis["@redis"]
    IsService -->|Supabase| Supa["@supabase"]
    IsService -->|Deployment| Vercel["@vercel"]
    IsService -->|Payments| Stripe["@stripe"]
```

---

## Usage

```bash
# Full feature pipeline (both platforms)
/feature user management with registration, login, RBAC, and password reset

# Claude Code alias
/build-feature Add user auth with email/password and OAuth2 (Google, GitHub)

# Bug fix with TDD
/bugfix Users can submit empty forms on registration page

# Validation only
/validate

# Single TDD cycle
/tdd Order total should apply percentage discount before tax

# Manual delegation
@product-owner write user stories for email verification
@architect design architecture for: @docs/specs/user-auth.md
@dotnet make the test at tests/unit/UserRegistrationTests.cs pass

# Swarm mode (Claude Code only)
maple swarm full
```

---

## The CLI and Supporting Scripts

The template ships two executable scripts.

### `scripts/maple` — The Global CLI

Install the template globally once, then use `maple` from any directory:

```bash
git clone https://github.com/kinncj/AI-Development-Squad-Template.git ~/.maple
echo 'export PATH="$HOME/.maple/scripts:$PATH"' >> ~/.zshrc  # or ~/.bashrc
source ~/.zshrc
```

| Command | What it does |
|---|---|
| `maple init [project-name]` | Scaffold template into new or current directory, git init, npm install, optional labels |
| `maple labels [owner/repo]` | Create all GitHub issue labels for the MAPLE pipeline |
| `maple swarm full` | Launch all plan.md tasks in parallel Zellij tabs |
| `maple swarm tasks 1 3 5` | Launch specific task numbers |
| `maple swarm agent @dotnet "prompt"` | Run a single named agent |

```bash
#!/usr/bin/env bash
# maple — MAPLE CLI
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2025 Kinn Coelho Juliao <kinncj@protonmail.com>
set -euo pipefail

TEMPLATE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLAN_FILE="${PLAN_FILE:-docs/specs/current/plan.md}"
```bash
#!/usr/bin/env bash
# maple — MAPLE CLI
#
# Install globally:
#   git clone https://github.com/kinncj/AI-Development-Squad-Template.git ~/.maple
#   echo 'export PATH="$HOME/.maple/scripts:$PATH"' >> ~/.zshrc
#   source ~/.zshrc
#
# Usage:
#   maple init   [project-name]          Scaffold template into new or current directory
#   maple labels [owner/repo]            Create GitHub labels in current repo
#   maple swarm  full                    Launch all plan tasks in parallel Zellij tabs
#   maple swarm  tasks <n> [n...]        Launch specific task numbers
#   maple swarm  agent <@name> <prompt>  Run a single named agent
#   maple help                           Show this help
#
# Copyright (C) 2025 Kinn Coelho Juliao <kinncj@protonmail.com>
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

TEMPLATE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLAN_FILE="${PLAN_FILE:-docs/specs/current/plan.md}"

# ─── Colour palette ───────────────────────────────────────────────────────────
if [[ -n "${NO_COLOR:-}" || "${TERM:-}" == "dumb" || ! -t 1 ]]; then
  R=''; B=''; D=''; GRN=''; YLW=''; CYN=''
  BGRN=''; BRED=''; BYEL=''; BCYN=''; BMGT=''
else
  R='\033[0m';    B='\033[1m';    D='\033[2m'
  GRN='\033[0;32m';  YLW='\033[0;33m';  CYN='\033[0;36m'
  BGRN='\033[1;32m'; BRED='\033[1;31m'; BYEL='\033[1;33m'
  BCYN='\033[1;36m'; BMGT='\033[1;35m'
fi

HR="  ${D}$(printf '─%.0s' {1..60})${R}"

# ─── UI primitives ────────────────────────────────────────────────────────────
header() {
  printf "\n"
  printf "  ${B}${BMGT}MAPLE${R}  ${D}·${R}  ${B}%s${R}\n" "$1"
  printf "%b\n\n" "$HR"
}

step()    { printf "\n  ${BCYN}›${R}  ${B}%s${R}\n" "$1"; }
ok()      { printf "  ${BGRN}✓${R}  %-32s  ${D}%s${R}\n" "$1" "${2:-}"; }
info()    { printf "  ${CYN}›${R}  %-16s${B}%s${R}\n" "$1" "${2:-}"; }
warn()    { printf "  ${BYEL}!${R}  ${YLW}%s${R}\n" "$1"; }
missing() { printf "  ${BYEL}?${R}  %-32s  ${YLW}not found — %s${R}\n" "$1" "$2"; }
fail()    { printf "\n  ${BRED}✗${R}  ${B}%s${R}\n\n" "$*" >&2; exit 1; }

task_row() {
  local n="$1" agent="$2" desc="$3"
  printf "  ${D}[%2d]${R}  ${BCYN}%-22s${R}  %s\n" "$n" "$agent" "$desc"
}

session_info() {
  local session="$1"
  printf "\n%b\n" "$HR"
  printf "  ${CYN}›${R}  Session   ${B}%s${R}\n" "$session"
  printf "  ${D}›  Detach   Ctrl-o d${R}\n"
  printf "  ${D}›  Kill     zellij delete-session %s${R}\n" "$session"
  printf "%b\n\n" "$HR"
}

# ─── init ─────────────────────────────────────────────────────────────────────
cmd_init() {
  local TARGET_DIR
  if [[ $# -gt 0 ]]; then
    TARGET_DIR="$(pwd)/$1"
  else
    TARGET_DIR="$(pwd)"
  fi

  header "Initialize Project"
  info "Template" "$TEMPLATE_DIR"
  info "Target  " "$TARGET_DIR"

  if [[ -d "$TARGET_DIR" ]] && [[ -n "$(ls -A "$TARGET_DIR" 2>/dev/null)" ]]; then
    warn "Directory is not empty: $TARGET_DIR"
    printf "  ${BYEL}?${R}  Existing files may be overwritten. Continue? [y/N] "
    read -r _ans
    [[ "$_ans" =~ ^[Yy]$ ]] || fail "Aborted."
  fi

  mkdir -p "$TARGET_DIR"

  step "Checking dependencies"
  local cmd install
  while IFS=: read -r cmd install; do
    if command -v "$cmd" &>/dev/null; then
      ok "$cmd" "$(command -v "$cmd")"
    else
      missing "$cmd" "$install"
    fi
  done <<'DEPS'
git:https://git-scm.com
gh:brew install gh
docker:https://docker.com
node:https://nodejs.org
claude:npm install -g @anthropic-ai/claude-code
DEPS

  step "Copying template"
  rsync -a \
    --exclude='.git' \
    --exclude='node_modules' \
    --exclude='*.log' \
    "$TEMPLATE_DIR/" "$TARGET_DIR/"
  local FILE_COUNT
  FILE_COUNT=$(find "$TARGET_DIR" -type f | wc -l | tr -d ' ')
  ok "Files copied" "$FILE_COUNT files"

  step "Initializing git repository"
  cd "$TARGET_DIR"
  git init -q
  git add -A
  git commit -q -m "chore: initialize from MAPLE Template

Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)
Source:    $(basename "$TEMPLATE_DIR")"
  ok "Repository initialized" "commit $(git rev-parse --short HEAD)"

  if [[ -f "package.json" ]]; then
    step "Installing dependencies"
    npm install --silent
    ok "npm dependencies installed"
    if npx playwright install chromium --quiet 2>/dev/null; then
      ok "Playwright browsers installed"
    else
      warn "Playwright install failed — run: npx playwright install"
    fi
  fi

  printf "\n%b\n" "$HR"
  printf "  ${CYN}?${R}  Bootstrap GitHub labels? ${D}(requires gh auth + remote repo)${R} [y/N] "
  read -r _ans
  if [[ "$_ans" =~ ^[Yy]$ ]]; then
    cmd_labels
  fi

  local PROJECT_NAME
  PROJECT_NAME="$(basename "$TARGET_DIR")"

  printf "\n%b\n" "$HR"
  printf "  ${BGRN}✓${R}  ${B}Squad ready${R}  ${D}→${R}  %s\n\n" "$TARGET_DIR"
  printf "  ${D}Next steps:${R}\n"
  [[ "$TARGET_DIR" != "$(pwd)" ]] && \
    printf "  ${BCYN}›${R}  ${D}cd${R} ${B}%s${R}\n" "$PROJECT_NAME"
  printf "  ${BCYN}›${R}  ${D}gh repo create${R} ${B}%s${R} ${D}--public --push --source=.${R}\n" "$PROJECT_NAME"
  printf "  ${BCYN}›${R}  ${D}claude${R} ${B}/feature${R} ${D}\"your first feature\"${R}\n"
  printf "\n%b\n\n" "$HR"
}

# ─── labels ───────────────────────────────────────────────────────────────────
cmd_labels() {
  header "Bootstrap GitHub Labels"

  local REPO="${1:-}"
  [[ -z "$REPO" ]] && \
    REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)
  [[ -z "$REPO" ]] && \
    fail "Could not detect repository.  Pass owner/repo as argument:\n     maple labels owner/repo"

  info "Repository  " "$REPO"

  local CREATED=0 SKIPPED=0

  label() {
    local name="$1" color="$2" desc="$3"
    if gh label create "$name" --repo "$REPO" --color "$color" \
         --description "$desc" --force &>/dev/null; then
      ok "$name" "$desc"
      CREATED=$((CREATED + 1))
    else
      printf "  ${D}–  %-30s  already exists${R}\n" "$name"
      SKIPPED=$((SKIPPED + 1))
    fi
  }

  printf "  ${B}${BCYN}Story & Type${R}\n"
  label "story"        "0075ca" "User story"
  label "bug"          "d73a4a" "Bug report"
  label "must-have"    "e11d48" "MoSCoW: Must have"
  label "should-have"  "f97316" "MoSCoW: Should have"
  label "could-have"   "eab308" "MoSCoW: Could have"
  label "wont-have"    "6b7280" "MoSCoW: Won't have"

  printf "\n  ${B}${BCYN}Pipeline Phases${R}\n"
  label "phase:discover"   "dbeafe" "Phase 1: Discovery"
  label "phase:architect"  "c7d2fe" "Phase 2: Architecture"
  label "phase:plan"       "e9d5ff" "Phase 3: Planning"
  label "phase:infra"      "fce7f3" "Phase 4: Infrastructure"
  label "phase:implement"  "d1fae5" "Phase 5: Implementation"
  label "phase:validate"   "bfdbfe" "Phase 6: Validation"
  label "phase:document"   "fef3c7" "Phase 7: Documentation"
  label "phase:done"       "dcfce7" "Phase 8: Complete"

  printf "\n  ${B}${BCYN}Status${R}\n"
  label "in-progress"      "fde68a" "Work in progress"
  label "blocked"          "fca5a5" "Blocked — needs human intervention"
  label "validated"        "86efac" "All tests passing"
  label "ready-for-review" "a5f3fc" "PR ready for review"

  printf "\n  ${B}${BCYN}TDD State${R}\n"
  label "tdd:red"      "fca5a5" "Failing test written"
  label "tdd:green"    "86efac" "Tests passing"
  label "tdd:refactor" "fde68a" "Refactor phase"

  printf "\n%b\n" "$HR"
  printf "  ${BGRN}✓${R}  ${B}Done.${R}  "
  printf "${GRN}%d created${R}  ${D}·${R}  ${D}%d skipped${R}\n" "$CREATED" "$SKIPPED"
  printf "  ${D}›  https://github.com/%s/labels${R}\n\n" "$REPO"
}

# ─── swarm ────────────────────────────────────────────────────────────────────
_swarm_require_zellij() {
  command -v zellij &>/dev/null || \
    fail "zellij not installed.  Install with: brew install zellij"
}

_swarm_require_plan() {
  [[ -f "$PLAN_FILE" ]] || \
    fail "Plan file not found: ${PLAN_FILE}\n     Run /feature first to generate a plan."
}

_extract_tasks() {
  grep -E '^\- \[ \] Task [0-9]+:' "$PLAN_FILE" 2>/dev/null || true
}

_parse_agent() { echo "$1" | grep -oE '@[a-zA-Z0-9_-]+' | head -1; }
_parse_desc()  {
  local task="$1" agent="$2"
  echo "$task" | sed "s/.*${agent}//" | sed 's/^[[:space:]]*//'
}

_kdl_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

cmd_swarm() {
  local sub="${1:-help}"; shift || true

  case "$sub" in
    full)
      local SESSION="swarm-$(date +%s)"
      header "Swarm  —  Full Plan"
      _swarm_require_zellij
      _swarm_require_plan
      [[ -n "${ZELLIJ:-}" ]] && fail "Already inside Zellij.  Detach first (Ctrl-o d)"

      mapfile -t TASKS < <(_extract_tasks)
      [[ ${#TASKS[@]} -eq 0 ]] && \
        fail "No unchecked tasks found in ${PLAN_FILE}\n     All tasks may already be complete."

      info "Plan   " "$PLAN_FILE"
      info "Session" "$SESSION"
      info "Agents " "${#TASKS[@]}"
      printf "\n%b\n" "$HR"

      local i task agent desc
      for i in "${!TASKS[@]}"; do
        task="${TASKS[$i]}"
        agent=$(_parse_agent "$task")
        desc=$(_parse_desc  "$task" "$agent")
        task_row "$((i + 1))" "$agent" "$desc"
      done
      printf "%b\n\n" "$HR"

      local LAYOUT; LAYOUT="$(mktemp "${TMPDIR:-/tmp}/maple-layout.XXXXXX.kdl")"
      printf 'layout {\n' > "$LAYOUT"
      for i in "${!TASKS[@]}"; do
        task="${TASKS[$i]}"; agent=$(_parse_agent "$task"); desc=$(_parse_desc "$task" "$agent")
        local _n_esc; _n_esc="$(_kdl_escape "${agent#@}-$((i+1))")"
        local _a_esc; _a_esc="$(_kdl_escape "${agent#@}")"
        local _p_esc; _p_esc="$(_kdl_escape "$desc")"
        printf '    tab name="%s" {\n        pane command="claude" {\n            args "--agent" "%s" "%s"\n        }\n    }\n' \
          "$_n_esc" "$_a_esc" "$_p_esc" >> "$LAYOUT"
        ok "Launched  ${BCYN}${agent}${R}"
      done
      printf '}\n' >> "$LAYOUT"

      session_info "$SESSION"
      zellij --layout "$LAYOUT" --session "$SESSION"
      rm -f "$LAYOUT"
      ;;

    tasks)
      [[ $# -eq 0 ]] && fail "Specify task numbers.  Usage: maple swarm tasks 1 3 5"
      local TASK_NUMS=("$@")
      local SESSION="swarm-$(date +%s)"

      header "Swarm  —  Selected Tasks"
      _swarm_require_zellij
      _swarm_require_plan
      [[ -n "${ZELLIJ:-}" ]] && fail "Already inside Zellij.  Detach first (Ctrl-o d)"

      mapfile -t ALL_TASKS < <(_extract_tasks)
      [[ ${#ALL_TASKS[@]} -eq 0 ]] && fail "No tasks found in ${PLAN_FILE}"

      info "Session" "$SESSION"
      info "Tasks  " "${TASK_NUMS[*]}"
      printf "\n%b\n" "$HR"

      local LAYOUT; LAYOUT="$(mktemp "${TMPDIR:-/tmp}/maple-layout.XXXXXX.kdl")"
      printf 'layout {\n' > "$LAYOUT"
      local PANE=0 NUM task agent desc

      for NUM in "${TASK_NUMS[@]}"; do
        task=$(printf '%s\n' "${ALL_TASKS[@]}" | grep "Task ${NUM}:" || true)
        if [[ -z "$task" ]]; then
          warn "Task ${NUM} not found in plan — skipping"
          continue
        fi
        agent=$(_parse_agent "$task")
        desc=$(_parse_desc  "$task" "$agent")
        task_row "$NUM" "$agent" "$desc"
        local _n_esc; _n_esc="$(_kdl_escape "${agent#@}-${NUM}")"
        local _a_esc; _a_esc="$(_kdl_escape "${agent#@}")"
        local _p_esc; _p_esc="$(_kdl_escape "$desc")"
        printf '    tab name="%s" {\n        pane command="claude" {\n            args "--agent" "%s" "%s"\n        }\n    }\n' \
          "$_n_esc" "$_a_esc" "$_p_esc" >> "$LAYOUT"
        ok "Launched  ${BCYN}${agent}${R}"
        PANE=$((PANE + 1))
      done
      printf '}\n' >> "$LAYOUT"

      [[ "$PANE" -eq 0 ]] && { rm -f "$LAYOUT"; fail "No valid tasks launched."; }
      session_info "$SESSION"
      zellij --layout "$LAYOUT" --session "$SESSION"
      rm -f "$LAYOUT"
      ;;

    agent)
      [[ $# -lt 2 ]] && fail "Usage: maple swarm agent @agent-name \"prompt\""
      local AGENT="$1"; shift
      local PROMPT="$*"
      local SESSION="swarm-$(date +%s)"

      header "Swarm  —  Single Agent"
      _swarm_require_zellij
      [[ -n "${ZELLIJ:-}" ]] && fail "Already inside Zellij.  Detach first (Ctrl-o d)"

      info "Agent " "$AGENT"
      info "Prompt" "$PROMPT"

      local LAYOUT; LAYOUT="$(mktemp "${TMPDIR:-/tmp}/maple-layout.XXXXXX.kdl")"
      local _n_esc; _n_esc="$(_kdl_escape "${AGENT#@}")"
      local _p_esc; _p_esc="$(_kdl_escape "$PROMPT")"
      printf 'layout {\n    tab name="%s" focus=true {\n        pane command="claude" {\n            args "--agent" "%s" "%s"\n        }\n    }\n}\n' \
        "$_n_esc" "$_n_esc" "$_p_esc" > "$LAYOUT"
      ok "Launched ${BCYN}${AGENT}${R}"

      session_info "$SESSION"
      zellij --layout "$LAYOUT" --session "$SESSION"
      rm -f "$LAYOUT"
      ;;

    help | *)
      header "Swarm  —  Help"
      printf "  ${B}USAGE${R}\n\n"
      printf "    ${D}maple swarm${R} ${B}<subcommand>${R} ${D}[arguments]${R}\n\n"
      printf "%b\n" "$HR"
      printf "  ${BCYN}full${R}\n"
      printf "  ${D}    Launch all unchecked tasks from:${R}\n"
      printf "  ${D}    %s${R}\n\n" "$PLAN_FILE"
      printf "  ${BCYN}tasks${R}  ${D}<n> [n...]${R}\n"
      printf "  ${D}    Launch specific task numbers from the plan${R}\n\n"
      printf "  ${BCYN}agent${R}  ${D}<@name> <prompt>${R}\n"
      printf "  ${D}    Run a single named agent with a custom prompt${R}\n\n"
      printf "%b\n" "$HR"
      printf "  ${D}PLAN_FILE env var overrides the default plan path${R}\n\n"
      ;;
  esac
}

# ─── help ─────────────────────────────────────────────────────────────────────
cmd_help() {
  header "Help"

  printf "  ${B}USAGE${R}\n\n"
  printf "    ${D}maple${R} ${B}<command>${R} ${D}[arguments]${R}\n"

  printf "\n%b\n" "$HR"
  printf "  ${B}COMMANDS${R}\n\n"

  printf "  ${BCYN}init${R}    ${D}[project-name]${R}\n"
  printf "  ${D}    Scaffold the template into a new or existing directory${R}\n\n"

  printf "  ${BCYN}labels${R}  ${D}[owner/repo]${R}\n"
  printf "  ${D}    Create GitHub issue labels for the MAPLE pipeline${R}\n\n"

  printf "  ${BCYN}swarm${R}   ${D}<full | tasks <n...> | agent <@name> <prompt>>${R}\n"
  printf "  ${D}    Launch agents in parallel Zellij tabs (Claude Code only)${R}\n"
  printf "  ${D}    Run: maple swarm help  for details${R}\n\n"

  printf "%b\n" "$HR"
  printf "  ${B}EXAMPLES${R}\n\n"

  printf "  ${D}# New project${R}\n"
  printf "  ${BCYN}maple${R} ${B}init${R} my-project\n\n"

  printf "  ${D}# Create labels after connecting a remote repo${R}\n"
  printf "  ${BCYN}maple${R} ${B}labels${R}\n\n"

  printf "  ${D}# Run all plan tasks in parallel${R}\n"
  printf "  ${BCYN}maple${R} ${B}swarm full${R}\n\n"

  printf "  ${D}# Run specific tasks${R}\n"
  printf "  ${BCYN}maple${R} ${B}swarm tasks${R} 1 3 5\n\n"

  printf "  ${D}# Single agent${R}\n"
  printf "  ${BCYN}maple${R} ${B}swarm agent${R} @dotnet ${D}\"make OrderTests.cs pass\"${R}\n"

  printf "\n%b\n" "$HR"
  printf "  ${D}Template:  %s${R}\n\n" "$TEMPLATE_DIR"
}

# ─── Dispatch ─────────────────────────────────────────────────────────────────
case "${1:-help}" in
  init)               shift; cmd_init   "$@" ;;
  labels)             shift; cmd_labels "$@" ;;
  swarm)              shift; cmd_swarm  "$@" ;;
  help | -h | --help) cmd_help ;;
  *)
    printf "\n  ${BRED}✗${R}  Unknown command: ${B}%s${R}\n" "$1" >&2
    printf "  ${D}Run${R} ${B}maple help${R} ${D}for available commands.${R}\n\n" >&2
    exit 1
    ;;
esac
```

### `Makefile` — The 12-Target Contract

```makefile
# Makefile — MAPLE multi-agent contract
# All agents, CI/CD, and the Orchestrator use these targets.
# Stack-specific commands go in the recipe bodies.
# DO NOT remove targets — agents depend on them.

.PHONY: build test test-integration test-e2e test-contract test-all \
        lint security-scan fmt \
        containers-up containers-down migrate

# ============================================================
# Build
# ============================================================

build:  ## Build all artifacts
	@echo "🔨 Building..."
	# --- Stack-specific: uncomment your stack ---
	# dotnet build --configuration Release
	# mvn package -DskipTests
	# npm run build
	# bun run build
	@echo "✅ Build complete"

# ============================================================
# Test Pyramid
# ============================================================

test:  ## Unit tests
	@echo "🧪 Running unit tests..."
	# dotnet test --filter "Category=Unit"
	# mvn test -Dgroups=unit
	# npx vitest run tests/unit
	# pytest tests/unit -v
	@echo "✅ Unit tests complete"

test-integration: containers-up  ## Integration tests (requires containers)
	@echo "🧪 Running integration tests..."
	# dotnet test --filter "Category=Integration"
	# mvn test -Dgroups=integration
	# npx vitest run tests/integration
	# pytest tests/integration -v
	@echo "✅ Integration tests complete"

test-e2e: containers-up  ## E2E tests (browser + API)
	@echo "🧪 Running E2E tests..."
	npx playwright test tests/e2e/
	@echo "✅ E2E tests complete"

test-contract:  ## Contract tests (schema validation)
	@echo "🧪 Running contract tests..."
	# npx vitest run tests/contract
	# pytest tests/contract -v
	@echo "✅ Contract tests complete"

test-all: test test-integration test-e2e test-contract  ## Full test suite
	@echo "✅ All tests passed"

# ============================================================
# Quality
# ============================================================

lint:  ## Lint all code
	@echo "🔍 Linting..."
	# dotnet format --verify-no-changes
	# npx eslint . && npx tsc --noEmit
	# ruff check .
	@echo "✅ Lint complete"

security-scan:  ## Security scanning
	@echo "🔒 Security scan..."
	# dotnet list package --vulnerable
	# npm audit --production
	# pip-audit
	# trivy fs .
	@echo "✅ Security scan complete"

fmt:  ## Format all code
	@echo "✨ Formatting..."
	# dotnet format
	# npx prettier --write .
	# ruff format .
	# terraform fmt -recursive
	@echo "✅ Format complete"

# ============================================================
# Infrastructure
# ============================================================

containers-up:  ## Start test containers
	@echo "🐳 Starting containers..."
	docker compose -f docker-compose.test.yml up -d --wait
	@echo "✅ Containers healthy"

containers-down:  ## Stop and remove test containers
	@echo "🐳 Stopping containers..."
	docker compose -f docker-compose.test.yml down -v
	@echo "✅ Containers removed"

migrate:  ## Run database migrations
	@echo "📦 Running migrations..."
	# dotnet ef database update
	# npx prisma migrate deploy
	# npx drizzle-kit push
	# supabase db push
	# flyway migrate
	@echo "✅ Migrations complete"

# ============================================================
# Help
# ============================================================

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
```

### `docker-compose.test.yml` — Test Infrastructure

```yaml
# docker-compose.test.yml
# Hermetic test infrastructure — used by make containers-up
# All agents reference this file for integration/E2E tests.

services:
  test-db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: testpass
      POSTGRES_DB: testdb
    ports:
      - "5433:5432"
    volumes:
      - ./docs/specs/current/contracts:/seeds:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 2s
      timeout: 5s
      retries: 10

  test-redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 2s
      timeout: 5s
      retries: 10

  # Uncomment per stack:

  # test-app:
  #   build:
  #     context: .
  #     target: development
  #   environment:
  #     DATABASE_URL: postgres://postgres:testpass@test-db:5432/testdb
  #     REDIS_URL: redis://test-redis:6379
  #   ports:
  #     - "3000:3000"
  #   depends_on:
  #     test-db:
  #       condition: service_healthy
  #     test-redis:
  #       condition: service_healthy
  #   healthcheck:
  #     test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
  #     interval: 5s
  #     timeout: 5s
  #     retries: 10
```

### `opencode.json` — OpenCode Project Configuration

```json
{
  "$schema": "https://opencode.ai/config.json",
  "theme": "catppuccin",
  "model": "anthropic/claude-sonnet-4-6",
  "default_agent": "orchestrator",
  "instructions": [
    "AGENTS.md",
    "CLAUDE.md"
  ],
  "permission": {
    "edit": "ask",
    "bash": {
      "*": "ask",
      "ls*": "allow",
      "cat*": "allow",
      "find*": "allow",
      "git status*": "allow",
      "git diff*": "allow",
      "gh *": "allow",
      "make *": "allow"
    },
    "webfetch": "deny"
  },
  "mcp": {
    "context7": {
      "type": "local",
      "command": ["npx", "-y", "@upstash/context7-mcp@latest"],
      "enabled": true
    }
  },
  "watcher": {
    "ignore": [
      "node_modules", ".git", "dist", "build", "coverage",
      ".next", ".nuxt", "bin", "obj", "target",
      "__pycache__", ".pytest_cache", ".mypy_cache",
      ".terraform", "*.tfstate*"
    ]
  }
}
```

Model IDs follow `provider/model-id` format (e.g., `anthropic/claude-opus-4-6`). Per-agent overrides (model, temperature, tools) live in agent markdown frontmatter. The project-level `permission` block sets safe defaults; agents can tighten but not loosen.

### `CLAUDE.md` — Claude Code Project Rules

```markdown
# CLAUDE.md — Project Rules for Claude Code

## Agent System
This project uses an orchestrated multi-agent system.
Default agent: @orchestrator (never writes code, delegates everything).

## Commands
- `/feature {description}` — Full 8-phase pipeline
- `/build-feature {description}` — Alias for /feature
- `/bugfix {description}` — Reproduce → fix → validate → CHANGELOG
- `/validate` — Run full test suite
- `/tdd {requirement}` — Single RED → GREEN → REFACTOR cycle

## Rules
1. The Orchestrator NEVER writes code. It delegates to specialist agents.
2. Tests are written BEFORE implementation (TDD enforced).
3. QA writes failing tests. Implementation agents make them pass.
4. 3 consecutive failures on any task → escalate to human.
5. All phases produce artifacts in `docs/specs/{feature-slug}/`.
6. `make test-all` must pass before Phase 8 gate.
7. Every feature gets a GitHub issue. Agents update issues via `gh` CLI.
8. Conventional Commits: `feat:`, `fix:`, `test:`, `docs:`, `infra:`.

## MCP Servers
- context7: Library documentation lookup (`use context7` in prompts)

## Skills
Read skills from `.claude/skills/` before executing tasks.
Key skills: tdd-workflow, playwright-cli, github-cli, mermaid-diagrams.
```

### `AGENTS.md` — Human-Readable Agent Roster

```markdown
# AGENTS.md — Multi-Agent Squad Roster

## Quick Reference

| # | Agent | Role | Model |
|---|---|---|---|
| 1 | orchestrator | Pipeline control (never codes) | opus |
| 2 | product-owner | User stories, acceptance criteria | sonnet |
| 3 | architect | ADR, contracts, threat models | opus |
| 4 | qa | Write tests (RED) + validate (GREEN) | sonnet |
| 5 | dotnet | .NET backend implementation | sonnet |
| 6 | javascript | Node.js / vanilla JS | sonnet |
| 7 | typescript | TypeScript backend/libraries | sonnet |
| 8 | react-vite | React + Vite + TypeScript SPA | sonnet |
| 9 | nextjs | Next.js full-stack | sonnet |
| 10 | java | Java backend (non-Spring) | sonnet |
| 11 | springboot | Spring Boot applications | sonnet |
| 12 | kubernetes | K8s manifests, Kustomize, Helm | gpt-4.1 |
| 13 | terraform | Terraform IaC | gpt-4.1 |
| 14 | docker | Dockerfiles, Compose | gpt-4.1 |
| 15 | postgresql | Schema, migrations, RLS | sonnet |
| 16 | redis | Caching, pub/sub, streams | sonnet |
| 17 | supabase | Auth, RLS, Edge Functions | sonnet |
| 18 | vercel | Deployment, edge, config | sonnet |
| 19 | stripe | Payments, webhooks, billing | sonnet |
| 20 | data-science | EDA, stats, visualization | sonnet |
| 21 | data-engineer | Pipelines, ETL, orchestration | sonnet |
| 22 | tensorflow | TF/Keras models, training | sonnet |
| 23 | pytorch | PyTorch models, training | sonnet |
| 24 | pandas-numpy | Data manipulation, arrays | sonnet |
| 25 | scikit | Classical ML, pipelines | sonnet |
| 26 | jupyter | Notebooks, papermill | sonnet |
| 27 | docs | Feature docs, CHANGELOG, Mermaid | sonnet |

## Pipeline Phases
1. DISCOVER → 2. ARCHITECT → 3. PLAN → 4. INFRA → 5. IMPLEMENT → 6. VALIDATE → 7. DOCUMENT → 8. FINAL GATE

## Makefile Contract
All agents use: `make build`, `make test`, `make test-integration`, `make test-e2e`,
`make test-contract`, `make test-all`, `make lint`, `make security-scan`, `make fmt`,
`make containers-up`, `make containers-down`, `make migrate`.

## Git Conventions
- Conventional Commits: `feat:`, `fix:`, `test:`, `docs:`, `infra:`, `refactor:`
- Branch naming: `feat/{slug}`, `fix/{slug}`
- Squash merge to main
```

### `playwright.config.ts` — Playwright E2E Configuration

```typescript
// playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { open: 'never' }],
    ['json', { outputFile: 'test-results/e2e-results.json' }],
  ],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  webServer: process.env.CI ? undefined : {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
});
```

| Component | Count |
|---|---|
| **Agents** | 27 (1 orchestrator, 2 discovery, 1 QA, 5 backend, 2 frontend, 3 infrastructure, 5 services, 2 data, 5 ML, 1 docs) |
| **Phases** | 8 |
| **Commands** | 5 (`/feature`, `/build-feature`, `/bugfix`, `/validate`, `/tdd`) |
| **Skills** | 17 (3 process, 3 output, 11 tool/CLI wrappers including GitHub CLI) |
| **MCP Servers** | 1 (Context7 only — Playwright moved to CLI+Skill) |
| **CLI Tools** | 20+ (dotnet, mvn, npm, bun, python, docker, kubectl, terraform, playwright-cli, stripe, supabase, vercel, jupyter, etc.) |
| **Makefile Targets** | 13 |
| **Test Types** | 6 (unit, integration, E2E browser, E2E API, contract, smoke) |
| **Platforms** | 2 (Claude Code Max, OpenCode + GH Copilot Enterprise) |

One entrypoint. Twenty-seven agents. Seventeen skills. One MCP server. Zero failing tests.