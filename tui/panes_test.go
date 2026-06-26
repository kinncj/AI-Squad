package main

import (
	"strings"
	"testing"
)

func TestResumeNote(t *testing.T) {
	got := resumeNote(2)
	if !strings.Contains(got, "2 pane(s)") {
		t.Errorf("resumeNote(2) = %q, want it to mention 2 pane(s)", got)
	}
	// With 0 panes the keystroke reached nothing — the message must NOT imply a
	// live nudge happened; it must point at the poll / manual continue instead.
	zero := resumeNote(0)
	if strings.Contains(zero, "pane(s)") {
		t.Errorf("resumeNote(0) = %q, should not claim panes were nudged", zero)
	}
	if !strings.Contains(zero, "poll") && !strings.Contains(zero, "continue") {
		t.Errorf("resumeNote(0) = %q, should mention the poll / manual continue fallback", zero)
	}
}
