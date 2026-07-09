# Quickstart — GitHub Copilot CLI

Get from zero to a running feature pipeline using **GitHub Copilot CLI** and MAPLE.

---

## Prerequisites

| Tool | Install |
|---|---|
| [GitHub CLI](https://cli.github.com) | `brew install gh` |
| [Copilot CLI extension](https://github.com/github/gh-copilot) | `gh extension install github/gh-copilot` |
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

`maple init` copies `.github/copilot-instructions.md` along with agents, skills, hooks, Makefile stubs, and docs structure into the current directory.

---

## 3. Customize the Makefile

The Makefile ships with stubs. Open `Makefile` and replace the recipe bodies with your stack's commands before running any feature pipeline.

---

## 4. Bootstrap GitHub

```bash
gh auth login
maple labels    # create MAPLE phase labels on the repo
maple project   # create a GitHub Project v2 board
```

---

## 5. Write your first story

Press `n` in the `maple` dashboard to open the Gherkin requirements wizard, or run `maple req` directly. The wizard produces a story file at `docs/stories/{slug}/Story.md` with embedded Gherkin and links it to a GitHub Issue.

---

## 6. Run a feature

```bash
gh copilot suggest "/feature short description of what you want to build"
```

Or use the `experimental` mode for cross-model review:

```bash
gh copilot suggest --experimental "/feature short description"
```

The orchestrator pipeline follows the same 8 phases as in Claude Code. Artifacts land in `docs/specs/{feature-slug}/`.

---

## Rubber Duck reviewer

MAPLE includes a `rubber-duck` prompt that invokes Copilot's built-in cross-model reviewer. Enable it at plan, code, and test checkpoints:

```bash
gh copilot suggest "/validate"   # triggers rubber-duck review before FINAL GATE
```

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
