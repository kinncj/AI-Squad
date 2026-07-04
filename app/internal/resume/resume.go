// Package resume implements `maple resume-session`: it reads the pinned session ids
// from .claude/state/sessions.json and re-launches the harness for one of them.
// Ported from tui/main.go.
package resume

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Session re-launches the pinned session for harness. If harness is "", it prefers
// the detected harness order (claude → copilot → opencode → cursor).
func Session(harness string) error {
	data, err := os.ReadFile(".claude/state/sessions.json")
	if err != nil {
		return fmt.Errorf("no sessions file — use the TUI [o] key to pin a session first\n  (expected .claude/state/sessions.json)")
	}
	var sessions map[string]string
	if err := json.Unmarshal(data, &sessions); err != nil {
		return fmt.Errorf("corrupt sessions file: %w", err)
	}
	if len(sessions) == 0 {
		return fmt.Errorf("sessions.json is empty — navigate to the Agents pane and press [o] or [p]")
	}

	if harness == "" {
		for _, pref := range []string{"claude", "copilot", "opencode", "cursor"} {
			if sessions[pref] != "" {
				harness = pref
				break
			}
		}
	}
	if harness == "" {
		for k := range sessions {
			harness = k
			break
		}
	}

	id := sessions[harness]
	if id == "" {
		var available []string
		for k, v := range sessions {
			if v != "" {
				available = append(available, k)
			}
		}
		return fmt.Errorf("no pinned session for %q\n  available: %s", harness, strings.Join(available, ", "))
	}

	var args []string
	switch harness {
	case "claude":
		args = []string{"claude", "--resume", id}
	case "opencode":
		args = []string{"opencode", "--session", id}
	case "copilot":
		args = []string{"copilot", "--resume=" + id}
	case "cursor":
		cursorBin := "cursor-agent"
		if _, err := exec.LookPath(cursorBin); err != nil {
			cursorBin = "cursor"
		}
		args = []string{cursorBin}
	default:
		return fmt.Errorf("unknown harness %q — supported: claude, copilot, opencode, cursor", harness)
	}

	short := id
	if len(short) > 8 {
		short = short[:8] + "…"
	}
	fmt.Printf("resuming %s session %s\n", harness, short)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if rtkPath, err := exec.LookPath("rtk"); err == nil && rtkPath != "" {
		cmd.Env = append(cmd.Env, "RTK_HOOK_AUDIT=1")
	}
	return cmd.Run()
}
