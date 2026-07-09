package state

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestMergeMapleJSONPreservesAndDeletes(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, ".claude", "state")
	os.MkdirAll(sd, 0o755)
	os.WriteFile(filepath.Join(sd, "maple.json"),
		[]byte(`{"taffy":"impl","stage":"IMPLEMENT","status":"RATE_LIMITED","resume_at":"soon","harness":"copilot"}`), 0o644)
	fs := NewFS(dir)

	if err := fs.MergeMapleJSON(map[string]any{"status": "RUNNING", "resume_at": nil, "auto_resume": true}); err != nil {
		t.Fatal(err)
	}
	p := fs.Pipeline()
	if p.Status != "RUNNING" || p.Taffy != "impl" || p.Harness != "copilot" {
		t.Errorf("merge lost keys: %+v", p)
	}
	if p.ResumeAt != "" {
		t.Errorf("resume_at should be deleted, got %q", p.ResumeAt)
	}
	if !p.AutoResume {
		t.Error("auto_resume should be true")
	}
	if err := fs.ClearPipeline(); err != nil {
		t.Fatal(err)
	}
	if fs.Pipeline().Status != "" {
		t.Error("ClearPipeline should remove maple.json")
	}
}
