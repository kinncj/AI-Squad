package state

import (
	"strings"
	"testing"
)

func TestPipelineLines(t *testing.T) {
	got := NewFS("testdata").PipelineLines()
	joined := strings.Join(got, "\n")
	// The fixture maple.json has status RUNNING and stage IMPLEMENT.
	if !strings.Contains(joined, "status") || !strings.Contains(joined, "RUNNING") {
		t.Errorf("pipeline lines missing status/RUNNING:\n%s", joined)
	}
	if !strings.Contains(joined, "stage") || !strings.Contains(joined, "IMPLEMENT") {
		t.Errorf("pipeline lines missing stage/IMPLEMENT:\n%s", joined)
	}
}

func TestPipelineLinesMissing(t *testing.T) {
	got := NewFS("testdata/nope").PipelineLines()
	if len(got) != 1 || !strings.Contains(got[0], "no active pipeline") {
		t.Errorf("missing state should note it, got %v", got)
	}
}

func TestSkills(t *testing.T) {
	got := NewFS("testdata").Skills()
	if len(got) != 2 || got[0] != "gh-issues" || got[1] != "humanizer" {
		t.Errorf("Skills() = %v, want [gh-issues humanizer]", got)
	}
}

func TestSkillsMissing(t *testing.T) {
	if got := NewFS("testdata/nope").Skills(); got != nil {
		t.Errorf("missing skills dir should yield nil, got %v", got)
	}
}

func TestAgents(t *testing.T) {
	got := NewFS("testdata").Agents()
	if len(got) != 2 || got[0] != "orchestrator" || got[1] != "qa" {
		t.Errorf("Agents() = %v, want [orchestrator qa]", got)
	}
}
