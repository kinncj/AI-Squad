# Quickstart — Cursor

Get from zero to a running feature pipeline using **Cursor** and MAPLE.

---

## Prerequisites

| Tool | Install |
|---|---|
| [Cursor](https://cursor.com) | Download from cursor.com |
| [GitHub CLI](https://cli.github.com) | `brew install gh` |
| [Git](https://git-scm.com) | pre-installed on macOS/Linux |
| [Go 1.26+](https://go.dev) | `brew install go` *(only to build from source)* |
| [Node.js](https://nodejs.org) | `brew install node` *(Playwright E2E tests + npx skills)* |
| [Docker](https://docker.com) | [docker.com/get-started](https://docker.com/get-started) |

---

## 1. Install `maple`

**Pre-built binary (recommended):**

```bash
curl -fsSL https://raw.githubusercontent.com/kinncj/MAPLE/main/scripts/install.sh | bash
```

Installs to `~/.tools/maple/bin/`. Add to your shell profile:

```bash
export PATH="$HOME/.tools/maple/bin:$PATH"
```

**From source:**

```bash
git clone https://github.com/kinncj/MAPLE.git
cd MAPLE
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

`maple init` copies `.cursor/` agents along with skills, hooks, Makefile stubs, and docs structure into the current directory.

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

## 6. Configure Cursor for MAPLE

Open your project in **Cursor**:

```bash
cursor .
```

### Enable Skills Discovery

Cursor automatically discovers skills from `.cursor/skills/`. Verify they appear in the Agent panel (look for the lightning icon or "@" mention in chat):

- Skills marketplace appears when you type `@` in the chat
- Select any MAPLE-provided skill to invoke it
- Cursor Skills follow the [Agent Skills standard](https://github.com/hutchic/.cursor/blob/main/docs/cursor-skills.md)

### Optional: Configure API Provider

If using Cursor's default provider, no configuration needed. To customize:

1. Open Cursor Settings → Model / API
2. Add your LLM provider (Claude, OpenAI, Anthropic, etc.)
3. Cursor agents will use your configured provider

---

## 7. Run a feature

Open Cursor's chat or composer and invoke an agent:

```
@orchestrator /feature "user can reset password via email link"
```

Or run any Cursor command with MAPLE agents:

```
@architect /help          # ask for architecture guidance
@qa /validate             # run full test suite
@spec-kit /spec           # MAPLE internal spec-kit (Specification Knowledge & Integration Toolkit): generate Gherkin from requirements
```

The orchestrator follows the same 8-phase pipeline as Claude Code and OpenCode.

---

## Skills Reference

All MAPLE skills are available in `.cursor/skills/`:

| Skill | Use when |
|-------|----------|
| `rubber-duck` | Need a second opinion on design or code |
| `gherkin-authoring` | Writing BDD test scenarios |
| `component-scaffold` | Creating new UI components |
| `mermaid-diagrams` | Designing systems with flowcharts |
| `gh-issues` | Managing GitHub issues programmatically |
| `docker-patterns` | Building containerized services |

Browse all skills in `.cursor/skills/` — each has a `SKILL.md` file with detailed instructions.

---

## Agent Commands

Available in any Cursor chat:

| Command | What it does |
|---|---|
| `/feature "description"` | Full 8-phase pipeline |
| `/bugfix "description"` | Reproduce → fix → validate → CHANGELOG |
| `/validate` | Run full test suite |
| `/tdd "requirement"` | Single RED → GREEN → REFACTOR cycle |
| `/help` | Show available commands |

---

## RTK Token Optimizer (Optional)

To compress tool output and reduce token usage by 60-90%:

```bash
maple init
rtk init -g --cursor  # wire RTK into Cursor globally
```

This is optional but recommended for sustained feature work. RTK compresses `bash`, `find`, `grep`, and test output automatically.

---

## Next steps

- [The 8-Phase Pipeline](./pipeline.md)
- [The Agents](./agents.md)
- [Customization Guide](./customization.md)
- [Cursor Skills Documentation](https://github.com/hutchic/.cursor/blob/main/docs/cursor-skills.md)
