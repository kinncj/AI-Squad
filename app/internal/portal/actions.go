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

// writeFeedback records reviewer feedback (message + attachments + action) for the agent,
// which watches .claude/state/design-feedback.json while paused.
func (s *Server) writeFeedback(m map[string]any, action string) {
	msg, _ := m["message"].(string)
	fb := map[string]any{
		"message":     msg,
		"attachments": m["attachments"],
		"action":      action, // "rejected" | "requested_changes"
		"status":      action,
		"ts":          time.Now().UTC().Format(time.RFC3339),
	}
	if b, err := json.Marshal(fb); err == nil {
		_ = os.WriteFile(s.feedbackFile(), b, 0o644)
	}
}

func (s *Server) clearFeedback() { _ = os.Remove(s.feedbackFile()) }

// appendReviewHistory records a change-request / rejection to the per-story review log so the
// portal can list "what was asked for" per story across rounds. Story/stage come from the
// current pending review.
func (s *Server) appendReviewHistory(action, message string) {
	rv, _ := s.fs.Review()
	entry := map[string]any{
		"story": rv.Story, "title": rv.Title, "stage": rv.Stage,
		"action": action, "message": strings.TrimSpace(message),
		"ts": time.Now().UTC().Format(time.RFC3339),
	}
	path := filepath.Join(s.root, ".claude", "state", "review-history.jsonl")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	if b, err := json.Marshal(entry); err == nil {
		_, _ = f.Write(append(b, '\n'))
	}
}

// handleReviewHistory returns the recorded change requests, newest first, optionally scoped
// to one story (?story=<id>).
func (s *Server) handleReviewHistory(w http.ResponseWriter, r *http.Request) {
	story := r.URL.Query().Get("story")
	data, err := os.ReadFile(filepath.Join(s.root, ".claude", "state", "review-history.jsonl"))
	items := []map[string]any{}
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var e map[string]any
			if json.Unmarshal([]byte(line), &e) != nil {
				continue
			}
			if story != "" && mapStrV(e, "story") != story {
				continue
			}
			items = append(items, e)
		}
	}
	// newest first
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func mapStrV(m map[string]any, k string) string { v, _ := m[k].(string); return v }

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

// handleReject records feedback and tells the harness to revise. It does NOT clear the gate
// or flip the pipeline to RUNNING — the gate stays open (pending) until the human approves
// the revised artifact, so the state never contradicts the "revise" instruction.
func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	body := s.readBody(r)
	s.writeFeedback(body, "rejected")
	msg, _ := body["message"].(string)
	n := harness.NotifyHarness(s.getenv, reviseInstruction("REJECTED the current stage", msg))
	s.appendReviewHistory("rejected", msg)
	stage := s.fs.ApprovalPending()
	s.logActivity("reject", "Stage rejected — revision feedback sent to the agent")
	s.Publish(map[string]any{"event": "change", "action": "reject"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nudged": n, "stage": stage})
}

// handleRequestChanges records feedback and tells the harness to revise, keeping the gate
// open (see handleReject).
func (s *Server) handleRequestChanges(w http.ResponseWriter, r *http.Request) {
	body := s.readBody(r)
	s.writeFeedback(body, "requested_changes")
	msg, _ := body["message"].(string)
	n := harness.NotifyHarness(s.getenv, reviseInstruction("REQUESTED CHANGES", msg))
	s.appendReviewHistory("requested_changes", msg)
	stage := s.fs.ApprovalPending()
	s.logActivity("request-changes", "Changes requested — revision feedback sent to the agent")
	s.Publish(map[string]any{"event": "change", "action": "request-changes"})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nudged": n, "stage": stage})
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
