package harness

import (
	"os"
	"strings"
	"testing"
)

func TestKey(t *testing.T) {
	cases := map[string]string{
		"claude":         "claude",
		"/usr/bin/claude": "claude",
		"copilot":        "copilot",
		"opencode":       "opencode",
		"cursor-agent":   "cursor",
		"cursor":         "cursor",
		"weird-tool":     "weird-tool",
	}
	for bin, want := range cases {
		if got := Key([]string{bin, "arg"}); got != want {
			t.Errorf("Key(%q) = %q, want %q", bin, got, want)
		}
	}
	if Key(nil) != "" {
		t.Error("Key(nil) should be empty")
	}
}

func TestLaunchInPaneNoMultiplexer(t *testing.T) {
	getenv := func(string) string { return "" } // neither TMUX nor ZELLIJ
	_, ok, err := LaunchInPane(getenv, "claude", []string{"claude"})
	if ok || err != nil {
		t.Errorf("outside a multiplexer, ok=%v err=%v; want ok=false err=nil", ok, err)
	}
}

func TestLaunchInPaneTmuxCapturesPaneID(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	var got []string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		got = append([]string{name}, args...)
		return []byte("%42\n"), nil
	})
	defer restore()
	swapLookPath(t, func(string) (string, error) { return "", os.ErrNotExist }) // no rtk

	getenv := func(k string) string {
		if k == "TMUX" {
			return "/tmp/tmux-1000/default,1234,0"
		}
		return ""
	}
	ref, ok, err := LaunchInPane(getenv, "claude", []string{"claude", "/spec-kit"})
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if ref.Kind != "tmux" || ref.Target != "%42" {
		t.Errorf("ref = %+v, want tmux/%%42", ref)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "tmux new-window -PF #{pane_id} -- claude /spec-kit") {
		t.Errorf("tmux invocation = %q", joined)
	}
	// The pane ref must be persisted so NotifyContinue can find it later.
	if _, statErr := os.Stat(panesFile); statErr != nil {
		t.Error("pane ref should be saved to panes.json")
	}
}

func TestNotifyContinueSendsToRecordedPanes(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	os.MkdirAll(".claude/state", 0o755)
	os.WriteFile(panesFile, []byte(`{"claude":{"kind":"tmux","target":"%7"},"opencode":{"kind":"zellij","target":"maple-opencode"}}`), 0o644)

	var calls []string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil, nil
	})
	defer restore()

	n := NotifyContinue()
	if n != 2 {
		t.Errorf("NotifyContinue nudged %d panes, want 2", n)
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "tmux send-keys -t %7 continue Enter") {
		t.Errorf("missing tmux send-keys: %q", joined)
	}
	if !strings.Contains(joined, "zellij action write-chars continue") {
		t.Errorf("missing zellij write-chars: %q", joined)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func swapRunner(fn func(string, ...string) ([]byte, error)) func() {
	old := runner
	runner = fn
	return func() { runner = old }
}

func swapLookPath(t *testing.T, fn func(string) (string, error)) {
	old := lookPath
	lookPath = fn
	t.Cleanup(func() { lookPath = old })
}

func chdir(t *testing.T, dir string) {
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}
