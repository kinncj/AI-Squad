package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionDetailCopilotTranscript(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	id := "abc-123"
	dir := filepath.Join(home, ".copilot", "session-state", id)
	os.MkdirAll(dir, 0o755)
	events := strings.Join([]string{
		`{"type":"session.start","data":{}}`,
		`{"type":"user.message","data":{"content":"add a filter"}}`,
		`{"type":"tool.execution_start","data":{"toolName":"write_file"}}`,
		`{"type":"assistant.message","data":{"content":[{"text":"Done adding the filter."}]}}`,
	}, "\n")
	os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte(events), 0o644)

	got := SessionDetail(Session{Source: "copilot", ID: id})
	joined := strings.Join(got, "\n")
	for _, want := range []string{"▶ add a filter", "🔧 write_file", "Done adding the filter"} {
		if !strings.Contains(joined, want) {
			t.Errorf("copilot transcript missing %q, got:\n%s", want, joined)
		}
	}
}

func TestSessionDetailFallbackSummary(t *testing.T) {
	// opencode has no readable file here → metadata summary, never a read error
	got := SessionDetail(Session{Source: "opencode", ID: "ses_x", Title: "My Session", TS: "2026-07-07 15:00"})
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "My Session") || !strings.Contains(joined, "Press o to resume") {
		t.Errorf("summary missing expected fields:\n%s", joined)
	}
	if strings.Contains(joined, "cannot read") {
		t.Error("must not attempt to read a non-path id")
	}
}
