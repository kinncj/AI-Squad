package state

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// SessionDetail renders a readable transcript for the detail overlay, chosen by source.
// Claude stores a JSONL path in ID; Copilot stores an id whose events live under
// ~/.copilot/session-state/<id>/events.jsonl; anything else falls back to a metadata
// summary so pressing Enter never errors with "cannot read <hash>".
func SessionDetail(se Session) []string {
	switch se.Source {
	case "claude":
		return SessionTranscript(se.ID)
	case "copilot":
		if lines := copilotTranscript(se.ID); len(lines) > 0 {
			return lines
		}
	}
	return sessionSummary(se)
}

func sessionSummary(se Session) []string {
	return []string{
		"── " + se.Title,
		"source:  " + se.Source,
		"updated: " + se.TS,
		"",
		"Press o to resume this session.",
	}
}

// copilotTranscript renders the user prompts, assistant replies, and tool calls from a
// Copilot session's events.jsonl. Returns nil when the file is absent/unreadable.
func copilotTranscript(id string) []string {
	home, err := os.UserHomeDir()
	if err != nil || id == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(home, ".copilot", "session-state", id, "events.jsonl"))
	if err != nil {
		return nil
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
		case "user.message":
			if c := eventContent(e); c != "" {
				out = append(out, "▶ "+truncate(c, 76))
			}
		case "assistant.message":
			if c := eventContent(e); c != "" {
				out = append(out, "  "+truncate(c, 76))
			}
		case "tool.execution_start":
			if n := eventToolName(e); n != "" {
				out = append(out, "🔧 "+n)
			}
		}
	}
	const maxLines = 400
	if len(out) > maxLines {
		out = out[len(out)-maxLines:]
	}
	return out
}

func eventContent(e map[string]any) string {
	data, _ := e["data"].(map[string]any)
	if data == nil {
		return ""
	}
	switch c := data["content"].(type) {
	case string:
		return strings.TrimSpace(strings.ReplaceAll(c, "\n", " "))
	case []any:
		var parts []string
		for _, blk := range c {
			if bm, ok := blk.(map[string]any); ok {
				if t, _ := bm["text"].(string); t != "" {
					parts = append(parts, t)
				}
			}
		}
		return strings.TrimSpace(strings.ReplaceAll(strings.Join(parts, " "), "\n", " "))
	}
	return ""
}

func eventToolName(e map[string]any) string {
	data, _ := e["data"].(map[string]any)
	if data == nil {
		return ""
	}
	n, _ := data["toolName"].(string)
	return n
}

// openCodeSessions reads recent OpenCode sessions for cwd from the OpenCode SQLite
// store via the sqlite3 CLI. Best-effort: returns nil when sqlite3 or the db is
// absent.
func openCodeSessions(home, cwd string) []Session {
	sqlite3, err := exec.LookPath("sqlite3")
	if err != nil {
		return nil
	}
	db := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if _, err := os.Stat(db); err != nil {
		return nil
	}
	esc := strings.ReplaceAll(cwd, "'", "''")
	query := fmt.Sprintf(
		"SELECT s.id, s.title, datetime(s.time_updated,'unixepoch'), "+
			"(SELECT COUNT(*) FROM part p WHERE p.session_id=s.id AND json_extract(p.data,'$.type')='tool') "+
			"FROM session s JOIN project pr ON s.project_id=pr.id "+
			"WHERE pr.worktree='%s' ORDER BY s.time_updated DESC LIMIT 10;", esc)
	out, err := exec.Command(sqlite3, db, query).Output()
	if err != nil {
		return nil
	}
	return parseOpenCodeRows(string(out))
}

// parseOpenCodeRows parses the pipe-delimited "id|title|ts|toolcount" rows.
func parseOpenCodeRows(out string) []Session {
	var rows []Session
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 2 {
			continue
		}
		id, title := parts[0], parts[1]
		if title == "" {
			title = shortenID(id)
		}
		ts := ""
		if len(parts) >= 3 {
			ts = parts[2]
			if len(ts) > 16 {
				ts = ts[:16]
			}
		}
		tools := 0
		if len(parts) >= 4 {
			tools, _ = strconv.Atoi(strings.TrimSpace(parts[3]))
		}
		rows = append(rows, Session{ID: id, Title: title, Source: "opencode", TS: ts, ToolCount: tools})
	}
	return rows
}

// copilotSessions reads Copilot workspace sessions for cwd from ~/.copilot/
// session-state. Best-effort: returns nil when the store is absent.
func copilotSessions(home, cwd string) []Session {
	root := filepath.Join(home, ".copilot", "session-state")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []Session
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta := parseCopilotWorkspace(filepath.Join(root, e.Name(), "workspace.yaml"))
		if meta["cwd"] != cwd {
			continue
		}
		id := meta["id"]
		if id == "" {
			id = e.Name()
		}
		title := firstNonEmpty(meta["summary"], meta["name"], shortenID(id))
		ts := meta["updated_at"]
		if ts == "" {
			ts = meta["created_at"]
		}
		if len(ts) > 16 {
			ts = ts[:16]
		}
		out = append(out, Session{ID: id, Title: title, Source: "copilot", TS: ts})
	}
	return out
}

// parseCopilotWorkspace reads the flat key/value workspace.yaml.
func parseCopilotWorkspace(path string) map[string]string {
	m := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(data), "\n") {
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
