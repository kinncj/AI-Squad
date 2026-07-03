// Package state reads MAPLE project state from the working tree — stories, sessions,
// pipeline status, and the like — behind small port interfaces. It has no TUI
// dependency; the dashboard adapts these types into pane sources. This replaces the
// flat tui/loaders.go with tested, single-responsibility readers.
package state

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Story is a Gherkin story surfaced in the dashboard.
type Story struct {
	ID       string
	Slug     string
	Priority string
	Phase    string
	UI       bool
	Issue    int
	Path     string
}

// StoryStore reads the project's stories.
type StoryStore interface {
	Stories() []Story
}

// FS reads project state from a root directory (usually the working dir ".").
type FS struct{ Root string }

// NewFS returns a filesystem-backed state reader rooted at root.
func NewFS(root string) *FS { return &FS{Root: root} }

// Stories reads docs/stories/*/Story.md and docs/stories/*.md (skipping _partials),
// parses frontmatter, and returns them sorted by ID.
func (s *FS) Stories() []Story {
	var out []Story
	base := filepath.Join(s.Root, "docs", "stories")

	dirStories, _ := filepath.Glob(filepath.Join(base, "*", "Story.md"))
	for _, p := range dirStories {
		if st, ok := parseStory(p); ok {
			out = append(out, st)
		}
	}
	flat, _ := filepath.Glob(filepath.Join(base, "*.md"))
	for _, p := range flat {
		if strings.HasPrefix(filepath.Base(p), "_") {
			continue
		}
		if st, ok := parseStory(p); ok {
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func parseStory(path string) (Story, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Story{}, false
	}
	defer f.Close()

	fm := frontmatter(f)
	if len(fm) == 0 {
		return Story{}, false
	}
	st := Story{
		ID:       firstNonEmpty(fm["id"], fm["story_id"], filepath.Base(filepath.Dir(path))),
		Slug:     fm["story_slug"],
		Priority: fm["priority"],
		Phase:    phaseFromLabels(fm["labels"]),
		UI:       fm["ui"] == "true",
		Path:     path,
	}
	if n, err := strconv.Atoi(fm["issue_number"]); err == nil {
		st.Issue = n
	}
	return st, true
}

// frontmatter parses the leading `---` fenced YAML-ish key/value block.
func frontmatter(f *os.File) map[string]string {
	m := map[string]string{}
	sc := bufio.NewScanner(f)
	in := false
	for sc.Scan() {
		line := sc.Text()
		if line == "---" {
			if !in {
				in = true
				continue
			}
			break
		}
		if !in {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.Trim(strings.TrimSpace(line[idx+1:]), `"'`)
		m[k] = v
	}
	return m
}

// phaseFromLabels extracts the `phase:*` label, defaulting to "discover".
func phaseFromLabels(labels string) string {
	for _, part := range strings.Split(labels, ",") {
		part = strings.Trim(strings.TrimSpace(part), `[]"' `)
		if strings.HasPrefix(part, "phase:") {
			return strings.TrimPrefix(part, "phase:")
		}
	}
	return "discover"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
