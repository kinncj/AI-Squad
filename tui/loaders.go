package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ─── Story loaders ────────────────────────────────────────────────────────────

func loadStories() []storyRow {
	var rows []storyRow
	dirs, _ := filepath.Glob("docs/stories/*/Story.md")
	for _, p := range dirs {
		if r, ok := parseStoryFile(p); ok {
			rows = append(rows, r)
		}
	}
	files, _ := filepath.Glob("docs/stories/*.md")
	for _, p := range files {
		if strings.HasPrefix(filepath.Base(p), "_") {
			continue
		}
		if r, ok := parseStoryFile(p); ok {
			rows = append(rows, r)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })
	return rows
}

func parseStoryFile(path string) (storyRow, bool) {
	f, err := os.Open(path)
	if err != nil {
		return storyRow{}, false
	}
	defer f.Close()

	fm := extractFrontmatter(f)
	if len(fm) == 0 {
		return storyRow{}, false
	}

	r := storyRow{
		id:       fm["id"],
		slug:     fm["story_slug"],
		priority: fm["priority"],
		path:     path,
	}
	if r.id == "" {
		r.id = fm["story_id"]
	}
	if r.id == "" {
		r.id = filepath.Base(filepath.Dir(path))
	}
	r.phase = extractPhaseFromLabels(fm["labels"])
	if fm["ui"] == "true" {
		r.ui = true
	}
	if n, err := strconv.Atoi(fm["issue_number"]); err == nil {
		r.issue = n
	}
	return r, true
}

func extractFrontmatter(f *os.File) map[string]string {
	m := map[string]string{}
	scanner := bufio.NewScanner(f)
	inFM := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if !inFM {
				inFM = true
				continue
			}
			break
		}
		if !inFM {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		v = strings.Trim(v, `"'`)
		m[k] = v
	}
	return m
}

func extractPhaseFromLabels(labels string) string {
	for _, part := range strings.Split(labels, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `[]"' `)
		if strings.HasPrefix(part, "phase:") {
			return strings.TrimPrefix(part, "phase:")
		}
	}
	return "discover"
}

// ─── PR loaders ───────────────────────────────────────────────────────────────

func loadPRsCmd() tea.Cmd {
	return func() tea.Msg {
		ghPath, err := exec.LookPath("gh")
		if err != nil {
			return prsLoadedMsg{err: "gh not found"}
		}
		out, err := runWithTimeout(timeoutNetwork, nil, ghPath, "pr", "list",
			"--json", "number,title,state",
			"--limit", "20",
		)
		if err != nil {
			return prsLoadedMsg{err: "gh pr list: " + strings.TrimSpace(string(out))}
		}
		var raw []struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			State  string `json:"state"`
		}
		if err := json.Unmarshal(out, &raw); err != nil {
			return prsLoadedMsg{err: "parse error"}
		}
		rows := make([]prRow, len(raw))
		for i, r := range raw {
			rows[i] = prRow{r.Number, r.Title, r.State}
		}
		return prsLoadedMsg{items: rows}
	}
}

func approvePRCmd(number int) tea.Cmd {
	return func() tea.Msg {
		ghPath, err := exec.LookPath("gh")
		if err != nil {
			return prApproveResultMsg{number: number, err: "gh not found"}
		}
		out, err := runWithTimeout(timeoutNetwork, nil, ghPath, "pr", "review", fmt.Sprintf("%d", number), "--approve")
		if err != nil {
			msg := strings.TrimSpace(string(out))
			// Strip verbose GraphQL prefix for readability
			if idx := strings.Index(msg, "GraphQL:"); idx >= 0 {
				msg = strings.TrimSpace(msg[idx+8:])
			}
			return prApproveResultMsg{number: number, err: msg}
		}
		return prApproveResultMsg{number: number}
	}
}

func loadPRDetailCmd(number int, title string) tea.Cmd {
	return func() tea.Msg {
		ghPath, err := exec.LookPath("gh")
		if err != nil {
			return prDetailLoadedMsg{err: "gh not found", title: title}
		}
		out, err := runWithTimeout(timeoutNetwork, nil, ghPath, "pr", "view", fmt.Sprintf("%d", number))
		if err != nil {
			return prDetailLoadedMsg{err: strings.TrimSpace(string(out)), title: title}
		}
		lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
		return prDetailLoadedMsg{lines: lines, title: fmt.Sprintf("#%d %s", number, title)}
	}
}

// ─── Design / logs / project ──────────────────────────────────────────────────

func loadDesignTree() []string {
	var entries []string
	_ = filepath.WalkDir("docs/design", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if filepath.Base(path) == ".gitkeep" {
			return nil
		}
		rel := strings.TrimPrefix(path, "docs/design/")
		if rel == "" || rel == "docs/design" {
			return nil
		}
		prefix := strings.Repeat("  ", strings.Count(rel, string(os.PathSeparator)))
		icon := "📄 "
		if d.IsDir() {
			icon = "📁 "
		}
		entries = append(entries, prefix+icon+filepath.Base(path))
		return nil
	})
	return entries
}

func loadLogLines(n int) []string {
	data, err := os.ReadFile(".claude/logs/skills.jsonl")
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			out = append(out, l)
			continue
		}
		parts := []string{}
		for _, k := range []string{"ts", "agent", "skill", "op", "file", "duration", "error"} {
			if v, ok := m[k]; ok && v != nil && v != "" {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
		}
		out = append(out, strings.Join(parts, "  "))
	}
	return out
}

func loadProjectName() string {
	data, err := os.ReadFile("project.config.yaml")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			n := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			return strings.Trim(n, `"'`)
		}
	}
	return ""
}

// ─── Pinned sessions ──────────────────────────────────────────────────────────

const sessionsFile = ".claude/state/sessions.json"

// loadPinnedSessions reads the persisted session IDs keyed by tool name.
func loadPinnedSessions() map[string]string {
	data, err := os.ReadFile(sessionsFile)
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]string{}
	}
	return m
}

// savePinnedSession persists a session ID for the given tool (e.g. "claude", "opencode").
func savePinnedSession(tool, id string) {
	_ = os.MkdirAll(".claude/state", 0o755)
	m := loadPinnedSessions()
	m[tool] = id
	data, _ := json.Marshal(m)
	_ = os.WriteFile(sessionsFile, append(data, '\n'), 0o644)
}

// ─── RTK harness registry ─────────────────────────────────────────────────────

type rtkHarness struct {
	name  string   // display name
	key   string   // storage key
	flags []string // args passed to rtk init
}

// allRTKHarnesses lists every supported harness in display order.
var allRTKHarnesses = []rtkHarness{
	{"Claude Code / Copilot (default)", "claude", []string{"init", "-g"}},
	{"Gemini CLI", "gemini", []string{"init", "-g", "--gemini"}},
	{"Codex (OpenAI)", "codex", []string{"init", "-g", "--codex"}},
	{"Cursor", "cursor", []string{"init", "--agent", "cursor"}},
	{"Windsurf", "windsurf", []string{"init", "--agent", "windsurf"}},
	{"Cline / Roo Code", "cline", []string{"init", "--agent", "cline"}},
	{"Kilo Code", "kilocode", []string{"init", "--agent", "kilocode"}},
	{"Google Antigravity", "antigravity", []string{"init", "--agent", "antigravity"}},
}

const rtkHarnessFile = ".claude/state/rtk-harnesses.json"

func loadRTKHarnesses() map[string]bool {
	data, err := os.ReadFile(rtkHarnessFile)
	if err != nil {
		return map[string]bool{}
	}
	var m map[string]bool
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]bool{}
	}
	return m
}

func saveRTKHarness(key string) {
	_ = os.MkdirAll(".claude/state", 0o755)
	m := loadRTKHarnesses()
	m[key] = true
	data, _ := json.Marshal(m)
	_ = os.WriteFile(rtkHarnessFile, append(data, '\n'), 0o644)
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func str(v interface{}) string {
	s, _ := v.(string)
	return s
}
