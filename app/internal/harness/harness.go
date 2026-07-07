// Package harness launches AI harnesses next to the maple TUI and nudges them to
// continue past human-approval gates. When maple runs inside a multiplexer (tmux or
// zellij) it opens the harness in a split pane/tab so the TUI stays live and the pane
// stays addressable; on approval it types "continue" into that pane (as the user
// would). Outside a multiplexer the caller falls back to an in-terminal launch.
// Ported from tui/panes.go + tui/terminal_spawn.go.
package harness

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const panesFile = ".claude/state/panes.json"

// PaneRef locates a launched harness within a multiplexer so it can be signalled.
type PaneRef struct {
	Kind   string `json:"kind"`   // "tmux" | "zellij"
	Target string `json:"target"` // tmux pane id, or zellij tab name
}

// runner runs a command, returning its stdout and error. Swapped in tests. Package
// default shells out for real.
var runner = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// lookPath is exec.LookPath, swappable in tests.
var lookPath = exec.LookPath

// Key maps a launch argv to a stable harness key.
func Key(args []string) string {
	if len(args) == 0 {
		return ""
	}
	switch filepath.Base(args[0]) {
	case "claude":
		return "claude"
	case "copilot":
		return "copilot"
	case "opencode":
		return "opencode"
	case "cursor-agent", "cursor":
		return "cursor"
	default:
		return args[0]
	}
}

// rtkWrap prepends `env RTK_HOOK_AUDIT=1` when rtk is installed, matching how maple
// runs harnesses so their output is token-compressed.
func rtkWrap(args []string) []string {
	if p, _ := lookPath("rtk"); p != "" {
		return append([]string{"env", "RTK_HOOK_AUDIT=1"}, args...)
	}
	return args
}

// LaunchInPane opens args in a multiplexer pane/tab and records a ref keyed by
// harness. ok=false means maple isn't inside tmux/zellij — the caller should run the
// harness in the current terminal instead. err is set only when a multiplexer was
// detected but the launch failed.
func LaunchInPane(getenv func(string) string, harness string, args []string) (PaneRef, bool, error) {
	if len(args) == 0 {
		return PaneRef{}, false, nil
	}
	cmd := rtkWrap(args)

	if getenv("TMUX") != "" {
		out, err := runner("tmux", append([]string{"new-window", "-PF", "#{pane_id}", "--"}, cmd...)...)
		if err != nil {
			return PaneRef{}, true, err
		}
		ref := PaneRef{Kind: "tmux", Target: strings.TrimSpace(string(out))}
		saveRef(harness, ref)
		return ref, true, nil
	}

	if getenv("ZELLIJ") != "" {
		tab := "maple-" + harness
		if _, err := runner("zellij", append([]string{"action", "new-tab", "--name", tab, "--"}, cmd...)...); err != nil {
			return PaneRef{}, true, err
		}
		ref := PaneRef{Kind: "zellij", Target: tab}
		saveRef(harness, ref)
		return ref, true, nil
	}

	return PaneRef{}, false, nil
}

// nonHarnessCommands are foreground pane commands we never nudge — shells, editors,
// pagers, multiplexers, VCS. Everything else is treated as a possible AI harness, so
// the nudge works for ANY harness (claude/copilot/opencode/cursor/pi/hermes/…, known
// or not) without maintaining an allow-list per tool.
var nonHarnessCommands = map[string]bool{
	"zsh": true, "bash": true, "sh": true, "fish": true, "tcsh": true, "csh": true, "ksh": true, "dash": true,
	"vim": true, "nvim": true, "vi": true, "nano": true, "emacs": true, "helix": true, "hx": true, "micro": true,
	"less": true, "more": true, "man": true, "bat": true,
	"tmux": true, "zellij": true, "screen": true,
	"top": true, "htop": true, "btop": true, "watch": true,
	"git": true, "lazygit": true, "gitui": true, "ssh": true, "mosh": true,
	"maple": true, // belt-and-suspenders alongside the $TMUX_PANE skip
}

// NotifyContinue types "continue" into harness panes so an agent that yielded to wait
// for a reply resumes. It nudges (1) every pane maple recorded when it launched a
// harness, and (2) as a fallback for harnesses maple did NOT launch, any *other* pane
// in the current tmux server whose foreground command looks like a harness. Returns
// how many panes accepted the keys. getenv is injected for tests.
func NotifyContinue(getenv func(string) string) int {
	n := 0
	nudgedTmux := map[string]bool{}
	for _, p := range loadPanes() {
		if p.Kind == "tmux" {
			nudgedTmux[p.Target] = true
		}
		if sendContinue(p) {
			n++
		}
	}
	if getenv("TMUX") != "" {
		n += broadcastTmux(getenv("TMUX_PANE"), nudgedTmux)
	}
	return n
}

// broadcastTmux sends "continue" to every tmux pane running a harness-like command,
// skipping maple's own pane and any already nudged. Avoids typing into shells/editors.
func broadcastTmux(self string, skip map[string]bool) int {
	out, err := runner("tmux", "list-panes", "-a", "-F", "#{pane_id}\t#{pane_current_command}")
	if err != nil {
		return 0
	}
	n := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		id, cmd := parts[0], parts[1]
		if id == self || skip[id] || nonHarnessCommands[cmd] {
			continue
		}
		if _, err := runner("tmux", "send-keys", "-t", id, "continue", "Enter"); err == nil {
			n++
		}
	}
	return n
}

func sendContinue(p PaneRef) bool {
	switch p.Kind {
	case "tmux":
		_, err := runner("tmux", "send-keys", "-t", p.Target, "continue", "Enter")
		return err == nil
	case "zellij":
		if _, err := runner("zellij", "action", "go-to-tab-name", p.Target); err != nil {
			return false
		}
		if _, err := runner("zellij", "action", "write-chars", "continue"); err != nil {
			return false
		}
		_, err := runner("zellij", "action", "write", "13") // Enter
		return err == nil
	}
	return false
}

func loadPanes() map[string]PaneRef {
	data, err := os.ReadFile(panesFile)
	if err != nil {
		return map[string]PaneRef{}
	}
	var m map[string]PaneRef
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]PaneRef{}
	}
	return m
}

func saveRef(harness string, p PaneRef) {
	_ = os.MkdirAll(filepath.Dir(panesFile), 0o755)
	m := loadPanes()
	m[harness] = p
	if data, err := json.Marshal(m); err == nil {
		_ = os.WriteFile(panesFile, append(data, '\n'), 0o644)
	}
}
