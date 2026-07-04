package state

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// GitChanges returns a working-tree summary: `git status --porcelain` followed by
// `git diff HEAD`. Empty git output yields a "clean" note.
func (s *FS) GitChanges() []string {
	status := gitOutput(s.Root, "status", "--porcelain")
	diff := gitOutput(s.Root, "diff", "HEAD")
	return formatGitChanges(status, diff)
}

func gitOutput(dir string, args ...string) string {
	git, err := exec.LookPath("git")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, git, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// formatGitChanges composes the status + diff sections into display lines.
func formatGitChanges(status, diff string) []string {
	out := []string{"── status ──"}
	if strings.TrimSpace(status) == "" {
		out = append(out, "(clean working tree)")
	} else {
		out = append(out, strings.Split(strings.TrimRight(status, "\n"), "\n")...)
	}
	if strings.TrimSpace(diff) != "" {
		out = append(out, "", "── diff HEAD ──")
		out = append(out, strings.Split(strings.TrimRight(diff, "\n"), "\n")...)
	}
	return out
}
