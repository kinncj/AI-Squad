package portal

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kinncj/maple/app/internal/harness"
)

// writeFeedback records reviewer feedback (message + attachments) for the agent.
func (s *Server) writeFeedback(m map[string]any) {
	msg, _ := m["message"].(string)
	fb := map[string]any{
		"message":     msg,
		"attachments": m["attachments"],
		"ts":          time.Now().UTC().Format(time.RFC3339),
	}
	if b, err := json.Marshal(fb); err == nil {
		_ = os.WriteFile(s.feedbackFile(), b, 0o644)
	}
}

func (s *Server) clearFeedback() { _ = os.Remove(s.feedbackFile()) }

// handleApprove clears the gate (approval-pending.txt + maple.json) and nudges the harness.
func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	_ = r
	if s.fs.ApprovalPending() == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "note": "no gate pending"})
		return
	}
	if err := s.fs.ApproveGate(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.clearFeedback()
	n := harness.NotifyContinue(s.getenv)
	s.Publish(map[string]any{"event": "change", "action": "approve"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nudged": n})
}

// handleReject records feedback, clears the gate so the agent re-reads it, and nudges.
func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	s.writeFeedback(s.readBody(r))
	if err := s.fs.RejectGate(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	n := harness.NotifyContinue(s.getenv)
	s.Publish(map[string]any{"event": "change", "action": "reject"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nudged": n})
}

// handleRequestChanges records feedback and clears the gate (agent picks up the changes).
func (s *Server) handleRequestChanges(w http.ResponseWriter, r *http.Request) {
	s.writeFeedback(s.readBody(r))
	_ = s.fs.RejectGate()
	n := harness.NotifyContinue(s.getenv)
	s.Publish(map[string]any{"event": "change", "action": "request-changes"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nudged": n})
}

// handleStop halts the workflow: marks maple.json STOPPED, clears any gate, and tells the
// harness to stop the pipeline (but stay interactive so the user can talk to it directly).
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	_ = r
	_ = s.fs.MergeMapleJSON(map[string]any{
		"status":           "STOPPED",
		"awaiting_approval": nil,
		"updated_at":       time.Now().UTC().Format(time.RFC3339),
	})
	_ = os.Remove(filepath.Join(s.root, ".claude", "state", "approval-pending.txt"))
	n := harness.NotifyHarness(s.getenv,
		"Stop the current workflow now. Do not continue the pipeline. Wait for my next instruction.")
	s.Publish(map[string]any{"event": "change", "action": "stop"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "stopped": true, "nudged": n})
}
