package main

import (
	"os/exec"
	"strings"
)

type gitFile struct {
	Status string // collapsed porcelain code: "M", "A", "D", "R", "??", …
	Path   string
	Staged bool
}

type gitChanges struct {
	Files []gitFile
	Err   string // non-empty when git is unavailable or this is not a repo
}

// statusGlyph collapses a porcelain XY pair into a single display code.
func statusGlyph(x, y byte) string {
	if x == '?' && y == '?' {
		return "??"
	}
	c := x
	if c == ' ' {
		c = y
	}
	return string(c)
}

// parsePorcelain parses `git status --porcelain` output into gitFile rows.
func parsePorcelain(out string) []gitFile {
	var files []gitFile
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		path := strings.TrimSpace(line[3:])
		if idx := strings.Index(path, " -> "); idx >= 0 { // rename/copy → keep destination
			path = path[idx+4:]
		}
		files = append(files, gitFile{
			Status: statusGlyph(x, y),
			Path:   path,
			Staged: x != ' ' && x != '?',
		})
	}
	return files
}

// diffLineKind classifies a unified-diff line for colouring.
func diffLineKind(line string) string {
	switch {
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		return "meta"
	case strings.HasPrefix(line, "@@"):
		return "hunk"
	case strings.HasPrefix(line, "+"):
		return "add"
	case strings.HasPrefix(line, "-"):
		return "del"
	default:
		return "ctx"
	}
}

// loadSelectedGitDiff recomputes the diff for the currently selected file and
// resets the diff scroll. Called on open and when the file selection moves.
func (m *dashboardModel) loadSelectedGitDiff() {
	m.gitDiffScroll = 0
	if m.gitChangesCur < len(m.gitChanges.Files) {
		m.gitDiffLines = strings.Split(strings.TrimRight(gitDiffFor(m.gitChanges.Files[m.gitChangesCur]), "\n"), "\n")
	} else {
		m.gitDiffLines = nil
	}
}

func (c gitChanges) stagedCount() int {
	n := 0
	for _, f := range c.Files {
		if f.Staged {
			n++
		}
	}
	return n
}

// loadGitChanges runs `git status --porcelain` and returns the changed files,
// or an error message when git is unavailable / this is not a repo.
func loadGitChanges() gitChanges {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return gitChanges{Err: "git not available (not a git repo, or git not on PATH)"}
	}
	return gitChanges{Files: parsePorcelain(string(out))}
}

// gitDiffFor returns the unified diff of a single file (vs HEAD; whole-file for
// untracked). Read-only; never mutates the working tree.
func gitDiffFor(f gitFile) string {
	if f.Status == "??" {
		out, _ := exec.Command("git", "diff", "--no-index", "--", "/dev/null", f.Path).CombinedOutput()
		return string(out)
	}
	out, _ := exec.Command("git", "diff", "HEAD", "--", f.Path).CombinedOutput()
	s := string(out)
	if strings.TrimSpace(s) == "" {
		return "(no textual diff)"
	}
	return s
}
