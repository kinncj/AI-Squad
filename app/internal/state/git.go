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

// GitFile is one changed file in the working tree.
type GitFile struct {
	Path   string
	Status string // porcelain XY code, e.g. " M", "??", "A "
}

// GitFiles lists the changed files from `git status --porcelain`.
func (s *FS) GitFiles() []GitFile {
	out := gitOutput(s.Root, "status", "--porcelain")
	var files []GitFile
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		files = append(files, GitFile{Status: line[:2], Path: strings.TrimSpace(line[3:])})
	}
	return files
}

// GitDiff returns the diff for one file (staged + unstaged) as lines.
func (s *FS) GitDiff(path string) []string {
	if path == "" {
		return []string{"(no file selected)"}
	}
	diff := gitOutput(s.Root, "diff", "HEAD", "--", path)
	if strings.TrimSpace(diff) == "" {
		diff = gitOutput(s.Root, "diff", "--", path) // untracked/unstaged fallback
	}
	if strings.TrimSpace(diff) == "" {
		return []string{"(no diff — new/untracked file or no textual change)"}
	}
	return strings.Split(strings.TrimRight(diff, "\n"), "\n")
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
