# Quickstart — Claude Code

Get from zero to a running feature pipeline using **Claude Code** and MAPLE.

---

## Prerequisites

| Tool | Install |
|---|---|
| [Claude Code](https://claude.ai/claude-code) | `npm install -g @anthropic-ai/claude-code` |
| [GitHub CLI](https://cli.github.com) | `brew install gh` |
| [Git](https://git-scm.com) | pre-installed on macOS/Linux |
| [Go 1.26+](https://go.dev) | `brew install go` *(only to build from source)* |
| [Node.js](https://nodejs.org) | `brew install node` *(Playwright E2E tests + npx skills)* |
| [Docker](https://docker.com) | [docker.com/get-started](https://docker.com/get-started) |

---

## 1. Install `maple`

**Pre-built binary (recommended):**

```bash
curl -fsSL https://raw.githubusercontent.com/kinncj/maple/main/scripts/install.sh | bash
```

Installs to `~/.tools/maple/bin/`. Add to your shell profile:

```bash
export PATH="$HOME/.tools/maple/bin:$PATH"
```

**From source:**

```bash
git clone https://github.com/kinncj/maple.git
cd maple
make build-tui           # produces ./maple
export PATH="$PWD:$PATH"
```

Verify: `maple --version`

---

## 2. Scaffold your project

```bash
cd your-project-directory
maple init
```

`maple init` copies agents, skills, hooks, taffy workflows, Makefile stubs, and docs structure into the current directory. It also wires RTK via hooks if available.

After init, `maple` launches the boot check and drops you into the dashboard.

---

## 3. Customize the Makefile

The Makefile ships with stubs. Open `Makefile` and replace the recipe bodies with your stack's commands:

```makefile
build:
    npm run build      # or: dotnet build, cargo build, etc.

test:
    npx vitest run tests/unit

test-integration:
    npx vitest run tests/integration

test-e2e:
    npx playwright test tests/e2e/
```

---

## 4. Bootstrap GitHub

Authenticate with the GitHub CLI, then from the `maple` dashboard:

```
:labels     # create MAPLE phase labels on the repo
:project    # create a GitHub Project v2 board
```

Or from the CLI: `maple labels` / `maple project`.

---

## 5. Write your first story

Press `n` in the dashboard (or run `maple req`) to open the Gherkin requirements wizard. Walk through the prompts — the wizard produces a story file at `docs/stories/{slug}/Story.md` with embedded Gherkin and links it to a GitHub Issue.

---

## 6. Run a feature

Open the project in **Claude Code**:

```bash
claude .
```

Then run:

```
/feature "short description of what you want to build"
```

The orchestrator takes over: DISCOVER → ARCHITECT → PLAN → INFRA → IMPLEMENT → VALIDATE → DOCUMENT → FINAL GATE. Each phase produces artifacts in `docs/specs/{feature-slug}/`. Human gates at Phase 1 and 2 pause for your approval.

---

## Commands reference

| Command | What it does |
|---|---|
| `/feature "description"` | Full 8-phase pipeline |
| `/bugfix "description"` | Reproduce → fix → validate → CHANGELOG |
| `/validate` | Run full test suite via `make test-all` |
| `/tdd "requirement"` | Single RED → GREEN → REFACTOR cycle |

---

## Next steps

- [The 8-Phase Pipeline](./pipeline.md) — what happens inside each phase
- [The Agents](./agents.md) — full roster and how to add custom agents
- [Customization Guide](./customization.md) — skills, themes, Makefile tweaks
