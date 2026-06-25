package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseArtifactStatus(t *testing.T) {
	if got := parseArtifactStatus("status: approved\n# x\n"); got != "approved" {
		t.Errorf("got %q", got)
	}
	if got := parseArtifactStatus("# no frontmatter"); got != "draft" {
		t.Errorf("default got %q", got)
	}
}

func TestA11ySummary(t *testing.T) {
	j := []byte(`{"violations":[{"impact":"critical"},{"impact":"serious"},{"impact":"minor"}]}`)
	crit, total := a11ySummary(j)
	if crit != 2 || total != 3 {
		t.Errorf("crit=%d total=%d", crit, total)
	}
}

func TestLoadDesignReviewAndApprove(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)
	os.MkdirAll("docs/design/wireframes", 0o755)
	os.MkdirAll("docs/design/mockups", 0o755)
	os.WriteFile("docs/design/wireframes/S1.wireframe.md", []byte("status: draft\n"), 0o644)
	os.WriteFile("docs/design/mockups/S1.mockup.md", []byte("status: approved\n"), 0o644)
	os.WriteFile("docs/design/mockups/S1.a11y.json", []byte(`{"violations":[{"impact":"serious"}]}`), 0o644)

	r := loadDesignReview("S1")
	if len(r.Artifacts) != 3 {
		t.Fatalf("want 3 artifacts, got %d", len(r.Artifacts))
	}
	if r.Artifacts[0].Kind != "wireframe" || r.Artifacts[0].Status != "draft" {
		t.Errorf("wireframe: %+v", r.Artifacts[0])
	}
	if r.Artifacts[2].Kind != "a11y" || r.Artifacts[2].Summary != "1 critical/serious" {
		t.Errorf("a11y: %+v", r.Artifacts[2])
	}

	if err := approveDesignArtifact(filepath.Join("docs/design/wireframes/S1.wireframe.md")); err != nil {
		t.Fatal(err)
	}
	if parseArtifactStatus(readFileT(t, "docs/design/wireframes/S1.wireframe.md")) != "approved" {
		t.Errorf("approve did not set status")
	}
}

func readFileT(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
