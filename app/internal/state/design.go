package state

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Artifact is a design artifact (wireframe or mockup) for a story.
type Artifact struct {
	Path   string
	Kind   string // "wireframes" or "mockups"
	Status string // from the artifact's `status:` line, or "—"
}

// DesignArtifacts lists the wireframe/mockup files for a story under docs/design,
// each with its current status, sorted by path.
func (s *FS) DesignArtifacts(storyID string) []Artifact {
	var out []Artifact
	for _, kind := range []string{"wireframes", "mockups"} {
		matches, _ := filepath.Glob(filepath.Join(s.Root, "docs", "design", kind, storyID+".*"))
		for _, p := range matches {
			out = append(out, Artifact{Path: p, Kind: kind, Status: artifactStatus(p)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// artifactStatus returns the value of the first `status:` line, or "—".
func artifactStatus(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "?"
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "status:") {
			return strings.TrimSpace(strings.TrimPrefix(t, "status:"))
		}
	}
	return "—"
}

// ApproveArtifact rewrites the artifact's first `status:` line to "approved",
// preserving indentation. Errors if the file has no status line.
func ApproveArtifact(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "status:") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + "status: approved"
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		}
	}
	return fmt.Errorf("no status line in %s", path)
}

// DesignTree returns an indented listing of docs/design (📁 dirs, 📄 files),
// skipping .gitkeep. Empty when the directory is absent.
func (s *FS) DesignTree() []string {
	base := filepath.Join(s.Root, "docs", "design")
	var out []string
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Name() == ".gitkeep" {
			return nil
		}
		rel, e := filepath.Rel(base, path)
		if e != nil || rel == "." {
			return nil
		}
		indent := strings.Repeat("  ", strings.Count(rel, string(os.PathSeparator)))
		icon := "📄 "
		if d.IsDir() {
			icon = "📁 "
		}
		out = append(out, indent+icon+d.Name())
		return nil
	})
	return out
}
