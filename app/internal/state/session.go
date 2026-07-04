package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session is a harness session surfaced in the dashboard.
type Session struct {
	ID        string // JSONL file path
	Title     string
	Source    string // "claude", "opencode", "copilot"
	TS        string // last-activity timestamp, "2006-01-02 15:04"
	ToolCount int    // number of tool calls in the session
}

// maxSessions caps how many recent sessions each source contributes.
const maxSessions = 10

// Sessions returns recent Claude, OpenCode, and Copilot sessions for the project at
// Root, merged and sorted newest-first. Each source is best-effort: a missing store
// contributes nothing.
func (s *FS) Sessions() []Session {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	cwd, err := filepath.Abs(s.Root)
	if err != nil {
		return nil
	}
	encoded := strings.ReplaceAll(cwd, "/", "-")

	var all []Session
	all = append(all, claudeSessions(filepath.Join(home, ".claude", "projects", encoded))...)
	all = append(all, openCodeSessions(home, cwd)...)
	all = append(all, copilotSessions(home, cwd)...)
	sort.Slice(all, func(i, j int) bool { return all[i].TS > all[j].TS })
	return all
}

// claudeSessions reads every *.jsonl in dir as a Claude session, newest first,
// capped at maxSessions. Exposed to the package for testing with a fixture dir.
func claudeSessions(dir string) []Session {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type finfo struct {
		path string
		mod  time.Time
	}
	var files []finfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, finfo{filepath.Join(dir, e.Name()), info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })

	var out []Session
	for _, f := range files {
		if len(out) >= maxSessions {
			break
		}
		if sess, ok := parseClaudeSession(f.path, f.mod); ok {
			out = append(out, sess)
		}
	}
	return out
}

// parseClaudeSession reads a Claude JSONL transcript and extracts its AI title and
// tool-call count. The title falls back to a shortened file name.
func parseClaudeSession(path string, mod time.Time) (Session, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, false
	}
	title := ""
	tools := 0
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e map[string]any
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		switch e["type"] {
		case "ai-title":
			if t, _ := e["aiTitle"].(string); t != "" {
				title = t
			}
		case "assistant":
			tools += countToolUses(e)
		}
	}
	if title == "" {
		title = shortenID(strings.TrimSuffix(filepath.Base(path), ".jsonl"))
	}
	return Session{
		ID:        path,
		Title:     title,
		Source:    "claude",
		TS:        mod.Format("2006-01-02 15:04"),
		ToolCount: tools,
	}, true
}

func countToolUses(e map[string]any) int {
	msg, ok := e["message"].(map[string]any)
	if !ok {
		return 0
	}
	content, ok := msg["content"].([]any)
	if !ok {
		return 0
	}
	n := 0
	for _, c := range content {
		if cm, ok := c.(map[string]any); ok && cm["type"] == "tool_use" {
			n++
		}
	}
	return n
}

// shortenID collapses a long id to "head…tail" for display.
func shortenID(id string) string {
	if len(id) > 20 {
		return id[:8] + "…" + id[len(id)-8:]
	}
	return id
}

// SessionTranscript renders a Claude session JSONL as readable lines: the title, any
// prompts (▶), and each tool call (🔧). Used by the session detail overlay.
func SessionTranscript(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{"(cannot read " + path + ")"}
	}
	var out []string
	for _, raw := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		var e map[string]any
		if json.Unmarshal([]byte(raw), &e) != nil {
			continue
		}
		switch e["type"] {
		case "ai-title":
			if t, _ := e["aiTitle"].(string); t != "" {
				out = append(out, "── "+t)
			}
		case "last-prompt":
			if p, _ := e["lastPrompt"].(string); p != "" {
				out = append(out, "▶ "+truncate(p, 72))
			}
		case "assistant":
			for _, name := range assistantToolNames(e) {
				out = append(out, "🔧 "+name)
			}
		}
	}
	if len(out) == 0 {
		out = append(out, "(empty session)")
	}
	return out
}

// assistantToolNames returns the tool_use names in an assistant record.
func assistantToolNames(e map[string]any) []string {
	msg, ok := e["message"].(map[string]any)
	if !ok {
		return nil
	}
	content, ok := msg["content"].([]any)
	if !ok {
		return nil
	}
	var names []string
	for _, c := range content {
		if cm, ok := c.(map[string]any); ok && cm["type"] == "tool_use" {
			if n, _ := cm["name"].(string); n != "" {
				names = append(names, n)
			}
		}
	}
	return names
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
