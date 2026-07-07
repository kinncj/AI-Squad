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
	// Sees past an env VAR=VAL wrapper (added for rtk).
	if got := Key([]string{"env", "RTK_HOOK_AUDIT=1", "copilot", "-i", "x"}); got != "copilot" {
		t.Errorf("Key past env wrapper = %q, want copilot", got)
	}
}

func TestRtkWrapDoesNotDoubleWrap(t *testing.T) {
	swapLookPath(t, func(string) (string, error) { return "/usr/bin/rtk", nil })
	// Already env-wrapped → returned unchanged (no second env prefix).
	in := []string{"env", "RTK_HOOK_AUDIT=1", "claude"}
	got := rtkWrap(in)
	if len(got) != len(in) {
		t.Errorf("rtkWrap double-wrapped: %v", got)
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
	var calls []string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
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
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "tmux split-window -h -PF #{pane_id} -- claude /spec-kit") {
		t.Errorf("split-window call missing: %q", joined)
	}
	if !strings.Contains(joined, "tmux select-pane -t %42 -T claude") {
		t.Errorf("pane should be titled with the harness name: %q", joined)
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
	os.WriteFile(panesFile, []byte(`{"claude":{"kind":"tmux","target":"%7"},"opencode":{"kind":"wezterm","target":"3"}}`), 0o644)

	var calls []string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil, nil
	})
	defer restore()

	// getenv returns no TMUX so only recorded panes are nudged (no broadcast).
	n := NotifyContinue(func(string) string { return "" })
	if n != 2 {
		t.Errorf("NotifyContinue nudged %d panes, want 2", n)
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "tmux send-keys -t %7 continue Enter") {
		t.Errorf("missing tmux send-keys: %q", joined)
	}
	if !strings.Contains(joined, "wezterm cli send-text --no-paste --pane-id 3 continue") {
		t.Errorf("missing wezterm send-text: %q", joined)
	}
}

func TestNotifyContinueBroadcastsToSiblingHarnessPanes(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir) // no panes.json — maple launched nothing

	var sent []string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		if name == "tmux" && len(args) >= 1 && args[0] == "list-panes" {
			// %1 = maple itself, %2 = a copilot (node) pane, %3 = a plain shell.
			return []byte("%1\tnode\n%2\tnode\n%3\tzsh\n"), nil
		}
		sent = append(sent, name+" "+strings.Join(args, " "))
		return nil, nil
	})
	defer restore()

	getenv := func(k string) string {
		switch k {
		case "TMUX":
			return "/tmp/tmux-1000/default,1,0"
		case "TMUX_PANE":
			return "%1" // maple's own pane — must be skipped
		}
		return ""
	}
	n := NotifyContinue(getenv)
	if n != 1 {
		t.Errorf("should nudge exactly the sibling harness pane (%%2), got %d", n)
	}
	joined := strings.Join(sent, "\n")
	if !strings.Contains(joined, "tmux send-keys -t %2 continue Enter") {
		t.Errorf("should send continue to the copilot pane, calls: %q", joined)
	}
	if strings.Contains(joined, "-t %1") {
		t.Error("must not nudge maple's own pane")
	}
	if strings.Contains(joined, "-t %3") {
		t.Error("must not nudge a plain shell pane")
	}
}

func TestInMultiplexerHerdr(t *testing.T) {
	herdr := func(k string) string {
		if k == "HERDR_PANE_ID" {
			return "w5:p1"
		}
		return ""
	}
	if !InMultiplexer(herdr) {
		t.Error("HERDR_PANE_ID should count as a multiplexer")
	}
	if InMultiplexer(func(string) string { return "" }) {
		t.Error("no multiplexer env → false")
	}
}

func TestLaunchInPaneHerdrSplitsRunsRenames(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	swapLookPath(t, func(string) (string, error) { return "", os.ErrNotExist }) // no rtk

	var calls []string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if len(args) >= 2 && args[0] == "pane" && args[1] == "split" {
			return []byte(`{"result":{"pane":{"pane_id":"w5:p2"},"type":"pane_info"}}`), nil
		}
		return nil, nil
	})
	defer restore()

	getenv := func(k string) string {
		if k == "HERDR_PANE_ID" {
			return "w5:p1"
		}
		return ""
	}
	ref, ok, err := LaunchInPane(getenv, "copilot", []string{"copilot", "-p", "do it"})
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if ref.Kind != "herdr" || ref.Target != "w5:p2" {
		t.Errorf("ref = %+v, want herdr/w5:p2", ref)
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "herdr pane split w5:p1 --direction right") {
		t.Errorf("missing split: %q", joined)
	}
	// the prompt arg with a space must be shell-quoted into the single run string
	if !strings.Contains(joined, "herdr pane run w5:p2 copilot -p 'do it'") {
		t.Errorf("run should shell-quote args: %q", joined)
	}
	if !strings.Contains(joined, "herdr pane rename w5:p2 copilot") {
		t.Errorf("missing rename: %q", joined)
	}
	if _, statErr := os.Stat(panesFile); statErr != nil {
		t.Error("herdr pane ref should be saved to panes.json")
	}
}

func TestLaunchInPaneHerdrPrefersOverTmux(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	swapLookPath(t, func(string) (string, error) { return "", os.ErrNotExist })
	var first string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		if first == "" {
			first = name
		}
		if name == "herdr" && len(args) >= 2 && args[1] == "split" {
			return []byte(`{"result":{"pane":{"pane_id":"w5:p2"}}}`), nil
		}
		return nil, nil
	})
	defer restore()
	// Both herdr and tmux present — herdr must win.
	getenv := func(k string) string {
		switch k {
		case "HERDR_PANE_ID":
			return "w5:p1"
		case "TMUX":
			return "/tmp/tmux/default,1,0"
		}
		return ""
	}
	ref, _, _ := LaunchInPane(getenv, "claude", []string{"claude"})
	if ref.Kind != "herdr" {
		t.Errorf("herdr should win over tmux, got %q", ref.Kind)
	}
	if first != "herdr" {
		t.Errorf("first call should be herdr, got %q", first)
	}
}

func TestNotifyContinueHerdrPane(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	os.MkdirAll(".claude/state", 0o755)
	os.WriteFile(panesFile, []byte(`{"copilot":{"kind":"herdr","target":"w5:p2"}}`), 0o644)

	var calls []string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil, nil
	})
	defer restore()

	n := NotifyContinue(func(string) string { return "" })
	if n != 1 {
		t.Errorf("want 1 nudge, got %d", n)
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "herdr pane send-text w5:p2 continue") {
		t.Errorf("missing send-text: %q", joined)
	}
	if !strings.Contains(joined, "herdr pane send-keys w5:p2 enter") {
		t.Errorf("missing send-keys enter: %q", joined)
	}
}

func TestNotifyContinueHerdrSessionScoped(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	os.MkdirAll(".claude/state", 0o755)
	os.WriteFile(panesFile, []byte(`{"copilot":{"kind":"herdr","target":"wB:p2","session":"maple-lolz"}}`), 0o644)

	var calls []string
	restore := swapRunner(func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil, nil
	})
	defer restore()

	NotifyContinue(func(string) string { return "" })
	joined := strings.Join(calls, "\n")
	// the recorded session must be passed explicitly so the nudge hits the right socket
	if !strings.Contains(joined, "herdr --session maple-lolz pane send-text wB:p2 continue") {
		t.Errorf("nudge should target the recorded session: %q", joined)
	}
	if !strings.Contains(joined, "herdr --session maple-lolz pane send-keys wB:p2 enter") {
		t.Errorf("send-keys should be session-scoped: %q", joined)
	}
}

func TestHerdrSessionFromSocketPath(t *testing.T) {
	cases := map[string]string{
		"/Users/x/.config/herdr/sessions/maple-lolz/herdr.sock": "maple-lolz",
		"/Users/x/.config/herdr/herdr.sock":                     "", // default session
		"": "",
	}
	for sock, want := range cases {
		got := herdrSession(func(k string) string {
			if k == "HERDR_SOCKET_PATH" {
				return sock
			}
			return ""
		})
		if got != want {
			t.Errorf("herdrSession(%q) = %q, want %q", sock, got, want)
		}
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"simple": "simple",
		"a b":    "'a b'",
		"":       "''",
		"it's":   `'it'\''s'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
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
