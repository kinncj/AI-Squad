package state

import (
	"strings"
	"testing"
)

func TestFormatGitChangesClean(t *testing.T) {
	got := formatGitChanges("", "")
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "── status ──") {
		t.Error("should have a status header")
	}
	if !strings.Contains(joined, "clean working tree") {
		t.Error("empty status should note a clean tree")
	}
	if strings.Contains(joined, "diff HEAD") {
		t.Error("no diff section when diff is empty")
	}
}

func TestFormatGitChangesDirty(t *testing.T) {
	status := " M app/foo.go\n?? new.txt"
	diff := "diff --git a/app/foo.go b/app/foo.go\n+added line"
	got := formatGitChanges(status, diff)
	joined := strings.Join(got, "\n")
	for _, want := range []string{"M app/foo.go", "?? new.txt", "── diff HEAD ──", "+added line"} {
		if !strings.Contains(joined, want) {
			t.Errorf("git changes missing %q; got:\n%s", want, joined)
		}
	}
}
