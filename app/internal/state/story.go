// Package state reads MAPLE project state from the working tree — stories, sessions,
// pipeline status, and the like — behind small port interfaces. It has no TUI
// dependency; the dashboard adapts these types into pane sources. This replaces the
// flat tui/loaders.go with tested, single-responsibility readers.
package state

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Story is a Gherkin story surfaced in the dashboard.
type Story struct {
	ID        string
	Title     string
	Slug      string
	Priority  string
	Phase     string
	UI        bool
	Issue     int
	Path      string
	RunStatus string // taffy-reported run status: "in_progress" | "done" | "failed" | ""
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
	statuses := s.storyStatuses()
	for i := range out {
		out[i].RunStatus = matchStoryStatus(statuses, s.Root, out[i])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// storyStatuses reads the per-story run status the taffy harness writes to
// .claude/state/story-status.json (keyed by story directory path, dir basename, or id).
func (s *FS) storyStatuses() map[string]string {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "story-status.json"))
	if err != nil {
		return nil
	}
	var m map[string]string
	if json.Unmarshal(data, &m) != nil {
		return nil
	}
	return m
}

// matchStoryStatus resolves a story's run status by trying the relative dir path, the dir
// basename, and the id — so the harness can key by whichever it has from the handoff.
func matchStoryStatus(m map[string]string, root string, st Story) string {
	if len(m) == 0 {
		return ""
	}
	dir := filepath.Dir(st.Path)
	if rel, err := filepath.Rel(root, dir); err == nil {
		if v := m[rel]; v != "" {
			return v
		}
	}
	if v := m[filepath.Base(dir)]; v != "" {
		return v
	}
	return m[st.ID]
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
		Title:    fm["title"],
		Slug:     firstNonEmpty(fm["story_slug"], fm["epic"]),
		Priority: firstNonEmpty(fm["priority"], phaseLabel(fm["labels"], "priority")),
		Phase:    firstNonEmpty(phaseLabel(fm["labels"], "phase"), "discover"),
		UI:       fm["ui"] == "true",
		Path:     path,
	}
	if n, err := strconv.Atoi(fm["issue_number"]); err == nil {
		st.Issue = n
	}
	return st, true
}

// frontmatter parses the leading `---` fenced YAML block. It handles scalar keys
// (`title: "x"`) and multi-line block lists (`labels:` followed by `  - "item"`
// lines), joining list items into a comma string. Enough for Story.md — not full YAML.
func frontmatter(f *os.File) map[string]string {
	m := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // stories can have long lines
	var lines []string
	in := false
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			if !in {
				in = true
				continue
			}
			break
		}
		if in {
			lines = append(lines, line)
		}
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		// Skip list items and indented lines — they belong to the key above.
		if line != trimmed || strings.HasPrefix(trimmed, "-") || trimmed == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.Trim(strings.TrimSpace(line[idx+1:]), `"'`)
		if v == "" { // maybe a block list follows
			var items []string
			for j := i + 1; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if strings.HasPrefix(t, "- ") {
					items = append(items, strings.Trim(strings.TrimPrefix(t, "- "), `"'`))
				} else if t == "" {
					continue
				} else {
					break
				}
			}
			v = strings.Join(items, ",")
		}
		m[k] = v
	}
	return m
}

// phaseLabel extracts the value of a `<prefix>:*` label (e.g. phase / priority) from a
// comma-joined label list, or "" when absent.
func phaseLabel(labels, prefix string) string {
	for _, part := range strings.Split(labels, ",") {
		part = strings.Trim(strings.TrimSpace(part), `[]"' `)
		if strings.HasPrefix(part, prefix+":") {
			return strings.TrimPrefix(part, prefix+":")
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
