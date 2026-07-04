package state

import "testing"

func TestPinnedSessionsRoundTrip(t *testing.T) {
	fs := NewFS(t.TempDir())
	if len(fs.PinnedSessions()) != 0 {
		t.Error("fresh project should have no pins")
	}
	if err := fs.SetPinnedSession("claude", "sess-abc"); err != nil {
		t.Fatalf("SetPinnedSession: %v", err)
	}
	if got := fs.PinnedSessions()["claude"]; got != "sess-abc" {
		t.Errorf("pinned claude = %q, want sess-abc", got)
	}
	// Re-pin replaces.
	_ = fs.SetPinnedSession("claude", "sess-xyz")
	if got := fs.PinnedSessions()["claude"]; got != "sess-xyz" {
		t.Errorf("re-pin should replace, got %q", got)
	}
	// Empty id clears.
	_ = fs.SetPinnedSession("claude", "")
	if _, ok := fs.PinnedSessions()["claude"]; ok {
		t.Error("empty id should clear the pin")
	}
}
