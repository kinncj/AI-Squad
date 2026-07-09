package portal

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// activityLog is the shared newline-delimited event log that `maple emit` (from agents) and
// the portal's own actions append to. The portal tails it and pushes each new line over SSE,
// so the browser shows a live activity feed without polling.
func activityLog(root string) string {
	return filepath.Join(root, ".claude", "state", "portal-events.jsonl")
}

// AppendActivity appends one event to the activity log (used by `maple emit` and portal
// actions). Best-effort; a missing state dir is created.
func AppendActivity(root string, ev map[string]any) error {
	path := activityLog(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

// tailActivity reads any lines appended to the activity log since the last poll, adds them
// to the in-memory ring, and publishes each as an SSE "activity" event.
func (s *Server) tailActivity() {
	data, err := os.ReadFile(activityLog(s.root))
	if err != nil {
		return
	}
	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) < s.actSeen { // file was truncated/rotated
		s.actSeen = 0
		s.actMu.Lock()
		s.activity = nil
		s.actMu.Unlock()
	}
	for _, line := range lines[s.actSeen:] {
		var ev map[string]any
		if json.Unmarshal([]byte(line), &ev) != nil {
			continue
		}
		s.actMu.Lock()
		s.activity = append(s.activity, ev)
		if len(s.activity) > 300 {
			s.activity = s.activity[len(s.activity)-300:]
		}
		s.actMu.Unlock()
		s.Publish(map[string]any{"event": "activity", "payload": ev})
	}
	s.actSeen = len(lines)
}

// logActivity records a portal-originated event (approve/stop/etc.) into the same feed.
func (s *Server) logActivity(kind, message string) {
	_ = AppendActivity(s.root, map[string]any{"event": kind, "message": message, "source": "portal"})
}

func (s *Server) handleActivity(w http.ResponseWriter, _ *http.Request) {
	s.actMu.Lock()
	items := make([]map[string]any, len(s.activity))
	copy(items, s.activity)
	s.actMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
