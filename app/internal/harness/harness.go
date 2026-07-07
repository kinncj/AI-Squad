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

// NotifyContinue types "continue" into every recorded harness pane, as if the user
// approved in that terminal. Returns how many panes accepted it.
func NotifyContinue() int {
	n := 0
	for _, p := range loadPanes() {
		if sendContinue(p) {
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
