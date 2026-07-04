package dashboard

import (
	"strings"
	"testing"

	"github.com/kinncj/maple/app/internal/state"
)

func sampleStories() []state.Story {
	return []state.Story{
		{ID: "auth-0001", Phase: "implement", UI: true},
		{ID: "checkout-0002", Phase: "architect"},
		{ID: "auth-0003", Phase: "validate"},
	}
}

func TestStorySourceRendersPhaseIDAndUIMark(t *testing.T) {
	s := newStorySource(sampleStories())
	rows := s.Rows()
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	if !strings.Contains(rows[0], "implement") || !strings.Contains(rows[0], "auth-0001") {
		t.Errorf("row 0 = %q, want phase + id", rows[0])
	}
	if !strings.Contains(rows[0], "◈") {
		t.Errorf("UI story should carry the ◈ mark, got %q", rows[0])
	}
	if strings.Contains(rows[1], "◈") {
		t.Errorf("non-UI story should not carry the mark, got %q", rows[1])
	}
}

func TestStorySourceFilterMatchesIDAndPhase(t *testing.T) {
	s := newStorySource(sampleStories())
	s.SetFilter("auth")
	if got := s.RowCount(); got != 2 {
		t.Errorf("filter 'auth' matched %d, want 2", got)
	}
	s.SetFilter("architect")
	if got := s.RowCount(); got != 1 {
		t.Errorf("filter 'architect' (phase) matched %d, want 1", got)
	}
	s.SetFilter("")
	if got := s.RowCount(); got != 3 {
		t.Errorf("empty filter should show all, got %d", got)
	}
}

func TestStorySourceEmpty(t *testing.T) {
	s := newStorySource(nil)
	if len(s.Rows()) != 0 || s.RowCount() != 0 {
		t.Errorf("empty source should render no rows")
	}
}

func TestSessionSourceRendersTagAndToolCount(t *testing.T) {
	s := newSessionSource([]state.Session{
		{Title: "Standardize tracking", Source: "claude", ToolCount: 293},
		{Title: "abc…def", Source: "copilot", ToolCount: 5},
		{Title: "oc one", Source: "opencode", ToolCount: 0},
	})
	rows := s.Rows()
	if rows[0] != "[cc] Standardize tracking  293t" {
		t.Errorf("row 0 = %q", rows[0])
	}
	if rows[1] != "[gh] abc…def  5t" {
		t.Errorf("copilot row = %q, want [gh] tag", rows[1])
	}
	if rows[2] != "[oc] oc one  0t" {
		t.Errorf("opencode row = %q, want [oc] tag", rows[2])
	}
}

func TestSessionSourceFilter(t *testing.T) {
	s := newSessionSource([]state.Session{
		{Title: "alpha", Source: "claude"},
		{Title: "beta", Source: "copilot"},
	})
	s.SetFilter("alpha")
	if s.RowCount() != 1 {
		t.Errorf("filter alpha matched %d, want 1", s.RowCount())
	}
	s.SetFilter("copilot") // matches source
	if s.RowCount() != 1 {
		t.Errorf("filter by source matched %d, want 1", s.RowCount())
	}
}
