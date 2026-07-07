package state

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

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
