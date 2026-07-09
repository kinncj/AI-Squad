package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectConfigAndClaudeDirExist(t *testing.T) {
	root := t.TempDir()
	fs := NewFS(root)
	if fs.ProjectConfigExists() {
		t.Error("empty dir should have no project.config.yaml")
	}
	if fs.ClaudeDirExists() {
		t.Error("empty dir should have no .claude")
	}
	os.WriteFile(filepath.Join(root, "project.config.yaml"), []byte("project:\n  name: x\n"), 0644)
	os.MkdirAll(filepath.Join(root, ".claude"), 0o755)
	if !fs.ProjectConfigExists() {
		t.Error("project.config.yaml should be detected")
	}
	if !fs.ClaudeDirExists() {
		t.Error(".claude dir should be detected")
	}
}

func TestClaudeDirRejectsFile(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, ".claude"), []byte("not a dir"), 0644)
	if NewFS(root).ClaudeDirExists() {
		t.Error("a .claude file (not dir) should not count")
	}
}
