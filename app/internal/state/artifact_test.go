package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesignArtifactsListsWithStatus(t *testing.T) {
	got := NewFS("testdata").DesignArtifacts("auth-reset-0001")
	if len(got) != 2 {
		t.Fatalf("got %d artifacts, want 2: %+v", len(got), got)
	}
	byKind := map[string]Artifact{}
	for _, a := range got {
		byKind[a.Kind] = a
	}
	if byKind["wireframes"].Status != "pending" {
		t.Errorf("wireframe status = %q, want pending", byKind["wireframes"].Status)
	}
	if byKind["mockups"].Status != "approved" {
		t.Errorf("mockup status = %q, want approved", byKind["mockups"].Status)
	}
}

func TestDesignArtifactsNoneForUnknownStory(t *testing.T) {
	if got := NewFS("testdata").DesignArtifacts("no-such-story"); len(got) != 0 {
		t.Errorf("unknown story should have no artifacts, got %d", len(got))
	}
}

func TestApproveArtifact(t *testing.T) {
	// Copy the pending wireframe into a temp file and approve it.
	src := "testdata/docs/design/wireframes/auth-reset-0001.wireframe.md"
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "artifact.md")
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}
	if artifactStatus(dst) != "pending" {
		t.Fatalf("precondition: status should be pending")
	}
	if err := ApproveArtifact(dst); err != nil {
		t.Fatalf("ApproveArtifact: %v", err)
	}
	if got := artifactStatus(dst); got != "approved" {
		t.Errorf("status after approve = %q, want approved", got)
	}
	// Other frontmatter is preserved.
	out, _ := os.ReadFile(dst)
	if !strings.Contains(string(out), "story: auth-reset-0001") {
		t.Error("approve should preserve the rest of the frontmatter")
	}
}

func TestApproveArtifactNoStatusLine(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "nostatus.md")
	_ = os.WriteFile(dst, []byte("# just a heading\n"), 0644)
	if err := ApproveArtifact(dst); err == nil {
		t.Error("approving a file with no status line should error")
	}
}
