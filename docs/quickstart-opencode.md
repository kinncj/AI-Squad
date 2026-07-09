# Quickstart — OpenCode

Get from zero to a running feature pipeline using **OpenCode** and MAPLE.

---

## Prerequisites

| Tool | Install |
|---|---|
| [OpenCode](https://opencode.ai) | `npm install -g opencode-ai` |
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

## 2. Configure OpenCode providers

OpenCode reads `opencode.json` in your project root. MAPLE ships a pre-configured one that uses Anthropic and GitHub Copilot providers. Edit it to match your available API keys:

```json
{
  "providers": {
    "anthropic": { "apiKey": "$ANTHROPIC_API_KEY" },
    "github-copilot": {}
  }
}
```

For GitHub Copilot, authenticate with:

```bash
gh auth login --scopes copilot
```

---

## 3. Scaffold your project

```bash
cd your-project-directory
maple init
```

`maple init` copies `.opencode/` agents along with skills, hooks, Makefile stubs, and docs structure into the current directory.

---

## 4. Customize the Makefile

The Makefile ships with stubs. Open `Makefile` and replace the recipe bodies with your stack's commands before running any feature pipeline.

---

## 5. Bootstrap GitHub

```bash
gh auth login
maple labels    # create MAPLE phase labels on the repo
maple project   # create a GitHub Project v2 board
```

---

## 6. Write your first story

Press `n` in the `maple` dashboard to open the Gherkin requirements wizard, or run `maple req` directly. The wizard produces a story file at `docs/stories/{slug}/Story.md` with embedded Gherkin and links it to a GitHub Issue.

---

## 7. Run a feature

Open the project in **OpenCode**:

```bash
opencode .
```

Then run:

```
/feature "short description of what you want to build"
```

The orchestrator follows the same 8-phase pipeline as in Claude Code. Agent routing in OpenCode uses the `permission.task` list in `.opencode/agents/orchestrator.md`.

---

## Agent model routing

OpenCode agents declare their own model in frontmatter. MAPLE's defaults:

| Agent | Model |
|---|---|
| `orchestrator`, `architect` | `anthropic/claude-opus-4-7` |
| Implementation agents | `github-copilot/claude-sonnet-4.5` |
| `kubernetes`, `terraform`, `docker` | `copilot/gpt-4.1` |

Change any agent's model by editing the `model:` field in its `.opencode/agents/{name}.md` file.

---

## Commands reference

| Command | What it does |
|---|---|
| `/feature "description"` | Full 8-phase pipeline |
| `/bugfix "description"` | Reproduce → fix → validate → CHANGELOG |
| `/validate` | Run full test suite |
| `/tdd "requirement"` | Single RED → GREEN → REFACTOR cycle |

---

## Next steps

- [The 8-Phase Pipeline](./pipeline.md)
- [The Agents](./agents.md)
- [Customization Guide](./customization.md)
