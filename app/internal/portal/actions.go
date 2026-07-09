package portal

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	s.logActivity("approve", "Stage approved from the portal")
	s.Publish(map[string]any{"event": "change", "action": "approve"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nudged": n})
}

// reviseInstruction is what the harness is told on reject / request-changes. It must NOT
// be "continue" (that reads as approval); it tells the agent to revise per the feedback.
func reviseInstruction(verb, message string) string {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "(see .claude/state/design-feedback.json)"
	}
	return "The reviewer " + verb + " in the design review. Do NOT treat this as approval and " +
		"do NOT proceed to the next stage. Read .claude/state/design-feedback.json, revise the " +
		"current artifact to address this feedback, then re-request review. Feedback: " + msg
}

// handleReject records feedback and tells the harness to revise (NOT to continue).
func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	body := s.readBody(r)
	s.writeFeedback(body)
	if err := s.fs.RejectGate(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	msg, _ := body["message"].(string)
	n := harness.NotifyHarness(s.getenv, reviseInstruction("REJECTED the current stage", msg))
	s.logActivity("reject", "Stage rejected — revision feedback sent to the agent")
	s.Publish(map[string]any{"event": "change", "action": "reject"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nudged": n})
}

// handleRequestChanges records feedback and tells the harness to revise (NOT to continue).
func (s *Server) handleRequestChanges(w http.ResponseWriter, r *http.Request) {
	body := s.readBody(r)
	s.writeFeedback(body)
	_ = s.fs.RejectGate()
	msg, _ := body["message"].(string)
	n := harness.NotifyHarness(s.getenv, reviseInstruction("REQUESTED CHANGES", msg))
	s.logActivity("request-changes", "Changes requested — revision feedback sent to the agent")
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
	s.logActivity("stop", "Workflow stopped from the portal")
	s.Publish(map[string]any{"event": "change", "action": "stop"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "stopped": true, "nudged": n})
}
