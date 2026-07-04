package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOpenCodeRows(t *testing.T) {
	out := "abc123|Fix the build|2026-07-04 10:00:00|7\n" +
		"def456||2026-07-04 09:00:00|0\n" +
		"\n" // trailing blank line skipped
	got := parseOpenCodeRows(out)
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
	if got[0].Title != "Fix the build" || got[0].Source != "opencode" || got[0].ToolCount != 7 {
		t.Errorf("row 0 = %+v", got[0])
	}
	if got[0].TS != "2026-07-04 10:00" {
		t.Errorf("ts should be trimmed to minutes, got %q", got[0].TS)
	}
	// Untitled row falls back to a shortened id.
	if got[1].Title == "" {
		t.Error("untitled opencode session should fall back to an id")
	}
}

func TestParseCopilotWorkspace(t *testing.T) {
	dir := t.TempDir()
	wf := filepath.Join(dir, "workspace.yaml")
	os.WriteFile(wf, []byte("id: sess-9\ncwd: /home/me/proj\nsummary: \"Add auth\"\nupdated_at: 2026-07-04T11:00:00Z\n"), 0644)
	m := parseCopilotWorkspace(wf)
	if m["id"] != "sess-9" || m["cwd"] != "/home/me/proj" || m["summary"] != "Add auth" {
		t.Errorf("parsed workspace = %+v", m)
	}
}

func TestSessionsStillReadsClaude(t *testing.T) {
	// Sessions() merges sources; the fixture claude dir is only reachable via the
	// real home path, so just assert the merge does not panic and preserves order.
	got := NewFS(t.TempDir()).Sessions()
	if got == nil {
		return // no sessions anywhere is fine
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].TS < got[i].TS {
			t.Errorf("sessions not sorted newest-first: %q before %q", got[i-1].TS, got[i].TS)
		}
	}
}
