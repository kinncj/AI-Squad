package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func fakeTemplate() fstest.MapFS {
	return fstest.MapFS{
		"CLAUDE.md":                 {Data: []byte("# rules\n")},
		"docs/stories/.gitkeep":     {Data: []byte("")},
		".claude/skills/x/SKILL.md": {Data: []byte("skill\n")},
		"project.config.yaml":       {Data: []byte("stale: do not copy\n")},
	}
}

func TestRunCopiesTreeAndGeneratesConfig(t *testing.T) {
	cwd := t.TempDir()
	written, err := Run(fakeTemplate(), cwd, false, "2026-07-04T00:00:00Z")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Files copied.
	if _, err := os.Stat(filepath.Join(cwd, "CLAUDE.md")); err != nil {
		t.Error("CLAUDE.md should be copied")
	}
	if _, err := os.Stat(filepath.Join(cwd, ".claude/skills/x/SKILL.md")); err != nil {
		t.Error("nested skill file should be copied")
	}
	// project.config.yaml is generated, not the stale template copy.
	cfg, err := os.ReadFile(filepath.Join(cwd, "project.config.yaml"))
	if err != nil {
		t.Fatal("project.config.yaml should be generated")
	}
	if strings.Contains(string(cfg), "stale") {
		t.Error("config should be generated, not the stale template copy")
	}
	if !strings.Contains(string(cfg), "milestone_granularity") {
		t.Error("generated config should have the github block")
	}
	// The stale template config is not in the written list.
	for _, w := range written {
		if w == "project.config.yaml" && !strings.Contains(string(cfg), "created_at") {
			t.Error("config should carry created_at")
		}
	}
}

func TestRunSkipsExistingUnlessForce(t *testing.T) {
	cwd := t.TempDir()
	// Pre-place a customized CLAUDE.md.
	os.WriteFile(filepath.Join(cwd, "CLAUDE.md"), []byte("MY CUSTOM RULES\n"), 0644)
	if _, err := Run(fakeTemplate(), cwd, false, "t"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(cwd, "CLAUDE.md"))
	if string(got) != "MY CUSTOM RULES\n" {
		t.Error("existing file should be kept without force")
	}
	// Force overwrites.
	if _, err := Run(fakeTemplate(), cwd, true, "t"); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(cwd, "CLAUDE.md"))
	if string(got) != "# rules\n" {
		t.Error("force should overwrite with the template file")
	}
}

func TestRunKeepsExistingConfig(t *testing.T) {
	cwd := t.TempDir()
	os.WriteFile(filepath.Join(cwd, "project.config.yaml"), []byte("project:\n  name: mine\n"), 0644)
	if _, err := Run(fakeTemplate(), cwd, true, "t"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(cwd, "project.config.yaml"))
	if !strings.Contains(string(got), "name: mine") {
		t.Error("an existing project.config.yaml must never be overwritten")
	}
}
