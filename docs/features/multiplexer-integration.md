# Multiplexer Integration (herdr / tmux / wezterm / kitty / zellij)

maple runs as a **mini agentic IDE**: when you launch a harness (claude / copilot / opencode /
cursor) it opens in a **side pane** next to the dashboard instead of taking over your terminal,
and when you approve a gate maple **nudges** that pane to continue. Both require a terminal
multiplexer. This page covers how maple picks one, wraps into one, launches panes, and nudges
them — and how to control all of it.

For the design rationale and trade-offs (including the herdr AGPL analysis), see
[ADR 0001 — herdr as the primary split-pane backend](../architecture/0001-herdr-primary-multiplexer.md).

---

## Backend preference

maple detects the multiplexer it is running inside from environment variables, in this order —
**first match wins**:

| Priority | Backend | Detected via | Pane addressing | Continue-nudge |
|---|---|---|---|---|
| 1 | **[herdr](https://herdr.dev)** | `HERDR_PANE_ID` | `w<ws>:p<n>` (socket API) | `pane send-text` + `send-keys enter` |
| 2 | tmux | `TMUX` | `%<id>` | `send-keys … Enter` |
| 3 | wezterm | `WEZTERM_PANE` | pane id | `cli send-text` |
| 4 | kitty | `KITTY_WINDOW_ID` | window id | `@ send-text` |
| 5 | zellij | `ZELLIJ` | *(none)* | falls back to the skill's file poll |

herdr is preferred because it is **agent-native**: it gives every pane a stable id over a local
socket API, so maple can address and nudge the exact harness pane it launched. herdr is always
**optional** — maple never requires it and falls back to tmux.

---

## Startup auto-wrap

If you run `maple` and you are **not** already inside a multiplexer, maple relaunches itself
inside one before showing the dashboard, so side-pane launches work in a plain terminal:

1. `wrapInHerdr` — used when the `herdr` binary is on `PATH`.
2. `wrapInTmux` — fallback: a styled tmux session (`wrapInTmux`).
3. Neither available (or both opted out) → harnesses run in the current terminal (suspend/resume).

### How `wrapInHerdr` works

herdr has no `tmux new-session <cmd>` equivalent — a fresh headless server starts with **zero
panes** — so maple bootstraps an isolated, persistent session named **`maple`** (it never
touches your `default` herdr session):

```
herdr session stop maple                        # clear any stale maple session
herdr --session maple server &                  # start a headless server for it
# wait until the server reports running
herdr --session maple workspace create --focus --cwd <project>   # → result.root_pane.pane_id
herdr --session maple pane run <id> "echo <sentinel>"            # prove the pane's shell is live…
herdr --session maple wait output <id> --match <sentinel>        # …block until it appears
herdr --session maple pane run <id> "exec maple"                 # replace the shell with maple
herdr --session maple                            # attach (blocks, like tmux attach)
```

The exec'd maple inherits the pane's `HERDR_PANE_ID`, so it detects it is already inside herdr
and does **not** re-wrap. If any step fails, maple stops the `maple` session and falls back to
tmux.

Because the `maple` session is **persistent**, detaching (not quitting) leaves it running
headless; the next `maple` launch stops the stale one and starts fresh, so you always get a
clean dashboard.

---

## Launching a harness into a side pane

The dashboard keys `o` / `L` / `x` / `S` / `i`, and the `maple req` "Implement via TAFFY" action,
all route through `harness.LaunchInPane`. Inside herdr that maps to:

```
herdr pane split $HERDR_PANE_ID --direction right --ratio 0.5 --focus   # new pane id from result.pane.pane_id
herdr pane run <new-id> "<shell-quoted harness argv>"                    # run the harness
herdr pane rename <new-id> <harness>                                     # title the pane
```

maple records the new pane as `{kind:"herdr", target:"<id>"}` in
`.claude/state/panes.json` so the continue-nudge can find it later.

---

## The continue-nudge

When an approval gate clears — in the TUI **or** the design-review portal — maple types
`continue` into every recorded harness pane so an agent that paused for confirmation resumes.
For herdr that is:

```
herdr pane send-text <id> continue
herdr pane send-keys <id> enter
```

(tmux/wezterm/kitty use their own send-text equivalents; zellij has no pane id, so its harnesses
resume via the skill's file poll instead.)

---

## Controlling it

### Install-time (installer flags / env)

herdr and rtk are installed by `scripts/install.sh` (macOS/Linux) and `scripts/install.ps1`
(Windows), the same way. Skip either:

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/kinncj/MAPLE/main/scripts/install.sh | bash -s -- --skip-herdr
SKIP_HERDR=1 SKIP_RTK=1 ./scripts/install.sh
```

```powershell
# Windows
./scripts/install.ps1 -SkipHerdr
```

Install herdr yourself at any time:

```bash
# macOS / Linux
curl -fsSL https://herdr.dev/install.sh | sh
```
```powershell
# Windows
powershell -ExecutionPolicy Bypass -c "irm https://herdr.dev/install.ps1 | iex"
```

### Runtime (env vars)

| Variable | Effect |
|---|---|
| `MAPLE_NO_HERDR=1` | Never auto-wrap into herdr (use tmux instead). |
| `MAPLE_NO_TMUX=1` | Never auto-wrap into tmux. |
| `MAPLE_NO_HERDR=1 MAPLE_NO_TMUX=1` | No auto-wrap — harnesses run in the current terminal (suspend/resume). |

To choose the backend explicitly, start the multiplexer yourself and then run `maple` inside it —
maple detects it and skips the wrap:

```bash
herdr        # then run: maple   (preferred)
tmux         # then run: maple
```

---

## Troubleshooting

- **"maple is still using tmux"** — you're likely on a build without herdr support, or `herdr`
  isn't on `PATH` when maple starts. Check `which herdr` and `maple --version`.
- **maple opens in tmux even though herdr is installed** — you launched maple from *inside* an
  existing tmux session, so `TMUX` was already set and maple used it. Start from a plain terminal
  (or from inside herdr) to get herdr.
- **Harness pane opens but the approval nudge doesn't reach it** — confirm the pane is recorded in
  `.claude/state/panes.json`; zellij panes are never recorded (no pane-id API) and rely on the
  file poll.
- **A stray `maple` herdr session lingers** — `herdr session stop maple` (safe; maple recreates it
  on next launch).
