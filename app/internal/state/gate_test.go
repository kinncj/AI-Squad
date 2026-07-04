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
