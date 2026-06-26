package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// resumeNote describes what happened after an approval, given how many agent
// panes accepted the "continue" keystroke. With 0 panes (a direct launch, no
// multiplexer, or an agent that maple did not spawn) the gate is still cleared
// by deleting approval-pending.txt — the agent resumes on its next poll — so the
// message must not imply a live nudge happened.
func resumeNote(n int) string {
	if n > 0 {
		return fmt.Sprintf("sent 'continue' to %d pane(s)", n)
	}
	return "no live agent pane to nudge — it resumes on its next poll, or paste 'continue' in its terminal"
}

type paneRef struct {
	Kind   string `json:"kind"`   // "tmux", "zellij", ""
	Target string `json:"target"` // pane id (tmux) or tab/session name
}

const panesFile = ".claude/state/panes.json"

func loadPanes() map[string]paneRef {
	data, err := os.ReadFile(panesFile)
	if err != nil {
		return map[string]paneRef{}
	}
	var m map[string]paneRef
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]paneRef{}
	}
	return m
}

func savePaneRef(harness string, p paneRef) {
	_ = os.MkdirAll(".claude/state", 0o755)
	m := loadPanes()
	m[harness] = p
	data, _ := json.Marshal(m)
	_ = os.WriteFile(panesFile, append(data, '\n'), 0o644)
}

// sendContinueToPane types "continue\n" into the target pane, as if the
// user typed it in the agent's terminal. Fails silently when the pane is
// gone or the multiplexer is unreachable.
func sendContinueToPane(p paneRef) bool {
	if p.Kind == "" || p.Target == "" {
		return false
	}
	switch p.Kind {
	case "tmux":
		if err := exec.Command("tmux", "send-keys", "-t", p.Target, "continue", "Enter").Run(); err == nil {
			return true
		}
	case "zellij":
		if err := exec.Command("zellij", "action", "go-to-tab-name", p.Target).Run(); err != nil {
			return false
		}
		if err := exec.Command("zellij", "action", "write-chars", "continue").Run(); err != nil {
			return false
		}
		if err := exec.Command("zellij", "action", "write", "13").Run(); err == nil {
			return true
		}
	}
	return false
}

// notifyAllPanesContinue sends "continue" to every recorded pane. Returns
// the number of panes that accepted the keys.
func notifyAllPanesContinue() int {
	n := 0
	for _, p := range loadPanes() {
		if sendContinueToPane(p) {
			n++
		}
	}
	return n
}
