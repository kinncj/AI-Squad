package state

import "testing"

func TestProjectName(t *testing.T) {
	if got := NewFS("testdata").ProjectName(); got != "test-project" {
		t.Errorf("ProjectName() = %q, want test-project", got)
	}
	if got := NewFS("testdata/nonexistent").ProjectName(); got != "" {
		t.Errorf("missing config should yield empty name, got %q", got)
	}
}

func TestTaffyCountExcludesSchema(t *testing.T) {
	if got := NewFS("testdata").TaffyCount(); got != 2 {
		t.Errorf("TaffyCount() = %d, want 2 (a, b; schema excluded)", got)
	}
}

func TestPipelineStatus(t *testing.T) {
	if got := NewFS("testdata").PipelineStatus(); got != "RUNNING" {
		t.Errorf("PipelineStatus() = %q, want RUNNING", got)
	}
	if got := NewFS("testdata/nonexistent").PipelineStatus(); got != "" {
		t.Errorf("missing state should yield empty status, got %q", got)
	}
}

func TestInFlight(t *testing.T) {
	for _, s := range []string{"RUNNING", "paused", "Rate_Limited"} {
		if !InFlight(s) {
			t.Errorf("InFlight(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"DONE", "", "idle"} {
		if InFlight(s) {
			t.Errorf("InFlight(%q) = true, want false", s)
		}
	}
}
