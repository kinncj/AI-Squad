package resume

import (
	"strings"
	"testing"
)

func TestResolvePreferredOrder(t *testing.T) {
	sessions := map[string]string{"opencode": "oc1", "claude": "cc1"}
	h, id, args, err := resolve(sessions, "", "cursor-agent")
	if err != nil {
		t.Fatal(err)
	}
	if h != "claude" { // claude wins the preferred order over opencode
		t.Errorf("harness = %q, want claude", h)
	}
	if id != "cc1" {
		t.Errorf("id = %q, want cc1", id)
	}
	if strings.Join(args, " ") != "claude --resume cc1" {
		t.Errorf("args = %v", args)
	}
}

func TestResolveExplicitHarness(t *testing.T) {
	sessions := map[string]string{"opencode": "oc1", "claude": "cc1"}
	h, _, args, err := resolve(sessions, "opencode", "cursor-agent")
	if err != nil || h != "opencode" {
		t.Fatalf("h=%q err=%v", h, err)
	}
	if strings.Join(args, " ") != "opencode --session oc1" {
		t.Errorf("args = %v", args)
	}
}

func TestResolvePerHarnessArgv(t *testing.T) {
	cases := []struct {
		harness, id, cursorBin, want string
	}{
		{"claude", "x", "cursor-agent", "claude --resume x"},
		// claude stores the JSONL path — resume must use the bare UUID.
		{"claude", "/Users/x/.claude/projects/p/abc-123.jsonl", "cursor-agent", "claude --resume abc-123"},
		{"copilot", "y", "cursor-agent", "copilot --resume=y"},
		{"opencode", "z", "cursor-agent", "opencode --session z"},
		{"cursor", "w", "cursor", "cursor"},
	}
	for _, c := range cases {
		_, _, args, err := resolve(map[string]string{c.harness: c.id}, c.harness, c.cursorBin)
		if err != nil {
			t.Fatalf("%s: %v", c.harness, err)
		}
		if strings.Join(args, " ") != c.want {
			t.Errorf("%s argv = %v, want %q", c.harness, args, c.want)
		}
	}
}

func TestResolveEmpty(t *testing.T) {
	if _, _, _, err := resolve(map[string]string{}, "", "cursor-agent"); err == nil {
		t.Error("empty sessions should error")
	}
}

func TestResolveUnknownHarness(t *testing.T) {
	_, _, _, err := resolve(map[string]string{"foo": "bar"}, "foo", "cursor-agent")
	if err == nil || !strings.Contains(err.Error(), "unknown harness") {
		t.Errorf("want unknown-harness error, got %v", err)
	}
}

func TestResolveNoPinForRequested(t *testing.T) {
	// requested harness has an empty id → error listing available ones
	_, _, _, err := resolve(map[string]string{"claude": "", "opencode": "oc1"}, "claude", "cursor-agent")
	if err == nil || !strings.Contains(err.Error(), "no pinned session") {
		t.Errorf("want no-pin error, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "opencode") {
		t.Errorf("error should list available harnesses, got %v", err)
	}
}
