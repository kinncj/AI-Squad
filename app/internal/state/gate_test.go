package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApprovalPending(t *testing.T) {
	if got := NewFS("testdata").ApprovalPending(); got != "IMPLEMENT" {
		t.Errorf("ApprovalPending() = %q, want IMPLEMENT", got)
	}
	if got := NewFS("testdata/nope").ApprovalPending(); got != "" {
		t.Errorf("no gate should yield empty, got %q", got)
	}
}

func TestApproveGate(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".claude", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(dir, "approval-pending.txt")
	if err := os.WriteFile(pending, []byte("VALIDATE\n"), 0644); err != nil {
		t.Fatal(err)
	}
	fs := NewFS(root)
	if fs.ApprovalPending() != "VALIDATE" {
		t.Fatal("precondition: gate should be pending")
	}
	if err := fs.ApproveGate(); err != nil {
		t.Fatalf("ApproveGate: %v", err)
	}
	if fs.ApprovalPending() != "" {
		t.Error("gate should be cleared after approve")
	}
	// Approving again (no file) is not an error.
	if err := fs.ApproveGate(); err != nil {
		t.Errorf("approving an absent gate should be a no-op, got %v", err)
	}
}

func TestRejectGate(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".claude", "state")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "approval-pending.txt"), []byte("DESIGN\n"), 0644)
	fs := NewFS(root)
	if err := fs.RejectGate(); err != nil {
		t.Fatalf("RejectGate: %v", err)
	}
	if fs.ApprovalPending() != "" {
		t.Error("reject should clear the pending gate")
	}
	rej, _ := os.ReadFile(filepath.Join(dir, "approval-rejected.txt"))
	if string(rej) != "DESIGN\n" {
		t.Errorf("reject should record the stage, got %q", rej)
	}
}

func TestPortalURL(t *testing.T) {
	root := t.TempDir()
	if NewFS(root).PortalURL() != "" {
		t.Error("no portal file should yield empty URL")
	}
	dir := filepath.Join(root, ".claude", "state")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "design-portal.url"), []byte("http://127.0.0.1:7802\n"), 0644)
	if got := NewFS(root).PortalURL(); got != "http://127.0.0.1:7802" {
		t.Errorf("PortalURL = %q", got)
	}
}
