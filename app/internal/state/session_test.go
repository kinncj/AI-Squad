package state

import "testing"

func TestClaudeSessionsReadsTitleAndToolCount(t *testing.T) {
	sessions := claudeSessions("testdata/claude-project")
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2 (ignore.txt must be skipped): %+v", len(sessions), sessions)
	}
	// Find the titled session (order is by mtime, which is not deterministic here).
	var titled, fallback *Session
	for i := range sessions {
		switch sessions[i].Title {
		case "Standardize GitHub issue tracking":
			titled = &sessions[i]
		default:
			fallback = &sessions[i]
		}
	}
	if titled == nil {
		t.Fatal("titled session not found")
	}
	if titled.ToolCount != 2 {
		t.Errorf("tool count = %d, want 2", titled.ToolCount)
	}
	if titled.Source != "claude" {
		t.Errorf("source = %q, want claude", titled.Source)
	}
	if fallback == nil {
		t.Fatal("fallback session not found")
	}
	// Untitled session falls back to a shortened file id (head…tail).
	if fallback.Title != "6de285d3…405db374" {
		t.Errorf("fallback title = %q, want 6de285d3…405db374", fallback.Title)
	}
	if fallback.ToolCount != 1 {
		t.Errorf("fallback tool count = %d, want 1", fallback.ToolCount)
	}
}

func TestClaudeSessionsMissingDir(t *testing.T) {
	if got := claudeSessions("testdata/nope"); got != nil {
		t.Errorf("missing dir should yield nil, got %+v", got)
	}
}

func TestShortenID(t *testing.T) {
	if got := shortenID("short"); got != "short" {
		t.Errorf("short id unchanged, got %q", got)
	}
	long := "0123456789abcdef0123456789" // 26 chars
	if got := shortenID(long); got != "01234567…23456789" {
		t.Errorf("shorten(%q) = %q, want 01234567…23456789", long, got)
	}
}
