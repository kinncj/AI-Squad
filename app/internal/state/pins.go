package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const sessionsFile = "sessions.json"

// PinnedSessions returns the pinned session id per harness source, from
// .claude/state/sessions.json. Empty map when absent.
func (s *FS) PinnedSessions() map[string]string {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", sessionsFile))
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if json.Unmarshal(data, &m) != nil {
		return map[string]string{}
	}
	return m
}

// SetPinnedSession pins id for source (or clears it when id==""), persisting the map.
func (s *FS) SetPinnedSession(source, id string) error {
	m := s.PinnedSessions()
	if id == "" {
		delete(m, source)
	} else {
		m[source] = id
	}
	dir := filepath.Join(s.Root, ".claude", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, sessionsFile), data, 0644)
}
