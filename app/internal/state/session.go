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

// Sessions returns recent Claude Code sessions for the project at Root, newest
// first. OpenCode and Copilot sources are added in a later iteration.
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
	dir := filepath.Join(home, ".claude", "projects", encoded)
	return claudeSessions(dir)
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
