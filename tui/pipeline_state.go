package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

const stalePipelineThreshold = 10 * time.Minute

const rateLimitBreadcrumbPath = ".claude/state/rate-limit.txt"

// pipelineState mirrors the state written by the pipeline-runner skill
// to .claude/state/maple.json
type pipelineState struct {
	Taffy            string `json:"taffy"`
	Stage            string `json:"stage"`
	Status           string `json:"status"` // RUNNING | PAUSED | RATE_LIMITED | DONE | FAILED
	AwaitingApproval string `json:"awaiting_approval"`
	StartedAt        string `json:"started_at"`
	UpdatedAt        string `json:"updated_at"`
	ResumeAt         string `json:"resume_at"`
	RateLimitReason  string `json:"rate_limit_reason"`
	Harness          string `json:"harness"`
	AutoResume       bool   `json:"auto_resume"`
	// recovery marker fields written by the TUI itself
	State string `json:"state"`
	TS    string `json:"ts"`
}

func (p pipelineState) isTaffy() bool {
	return p.Taffy != ""
}

// isStale returns true if the pipeline claims RUNNING but hasn't been updated
// in stalePipelineThreshold — meaning the agent likely died without writing DONE/FAILED.
func (p pipelineState) isStale() bool {
	if strings.ToUpper(p.Status) != "RUNNING" {
		return false
	}
	if p.UpdatedAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, p.UpdatedAt)
	if err != nil {
		return false
	}
	return time.Since(t) > stalePipelineThreshold
}

func (p pipelineState) isRateLimited() bool {
	return strings.ToUpper(p.Status) == "RATE_LIMITED"
}

// windowCleared reports whether the rate-limit window has passed. Returns false
// when resume_at is empty or unparseable — the run stays paused for manual resume.
func (p pipelineState) windowCleared(now time.Time) bool {
	if p.ResumeAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, p.ResumeAt)
	if err != nil {
		return false
	}
	return !now.Before(t)
}

func (p pipelineState) statusIcon() string {
	switch strings.ToUpper(p.Status) {
	case "RUNNING":
		return "▶"
	case "PAUSED":
		return "⏸"
	case "RATE_LIMITED":
		return "⚑"
	case "DONE":
		return "✓"
	case "FAILED":
		return "✗"
	default:
		return "·"
	}
}

// approvalPending returns the stage name from .claude/state/approval-pending.txt,
// or "" if no approval is waiting. The pipeline-runner skill writes this file at
// human-approval gates; the TUI deletes it when the user presses [a].
func approvalPending() string {
	data, err := os.ReadFile(".claude/state/approval-pending.txt")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func loadPipelineState() (pipelineState, error) {
	data, err := os.ReadFile(".claude/state/maple.json")
	if err != nil {
		return pipelineState{}, err
	}
	var ps pipelineState
	if err := json.Unmarshal(data, &ps); err != nil {
		return pipelineState{}, err
	}
	return ps, nil
}

// mergeMapleJSON applies updates to .claude/state/maple.json, preserving every
// other key. A value of nil deletes its key. Follows the merge-not-overwrite protocol.
func mergeMapleJSON(updates map[string]interface{}) error {
	_ = os.MkdirAll(".claude/state", 0o755)
	merged := map[string]interface{}{}
	if raw, err := os.ReadFile(".claude/state/maple.json"); err == nil {
		_ = json.Unmarshal(raw, &merged)
	}
	for k, v := range updates {
		if v == nil {
			delete(merged, k)
			continue
		}
		merged[k] = v
	}
	data, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	return os.WriteFile(".claude/state/maple.json", append(data, '\n'), 0o644)
}

// parseRateLimitBreadcrumb splits a "resume_at|reason" line. A line with no pipe
// is treated as resume_at only. Both sides are trimmed.
func parseRateLimitBreadcrumb(content string) (resumeAt, reason string) {
	line := strings.TrimSpace(content)
	if line == "" {
		return "", ""
	}
	if i := strings.Index(line, "|"); i >= 0 {
		return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
	}
	return line, ""
}

// reclassifyRateLimit promotes a stale RUNNING pipeline to RATE_LIMITED when the
// agent left a breadcrumb it could not turn into a full state write. It consumes
// (deletes) the breadcrumb and returns the reloaded state plus whether it promoted.
func reclassifyRateLimit(ps pipelineState) (pipelineState, bool) {
	if strings.ToUpper(ps.Status) != "RUNNING" || !ps.isStale() {
		return ps, false
	}
	raw, err := os.ReadFile(rateLimitBreadcrumbPath)
	if err != nil {
		return ps, false
	}
	resumeAt, reason := parseRateLimitBreadcrumb(string(raw))
	if resumeAt == "" {
		resumeAt = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	}
	if reason == "" {
		reason = "rate limit (reclassified from stale run)"
	}
	_ = mergeMapleJSON(map[string]interface{}{
		"status":            "RATE_LIMITED",
		"resume_at":         resumeAt,
		"rate_limit_reason": reason,
	})
	_ = os.Remove(rateLimitBreadcrumbPath)
	if ps2, err := loadPipelineState(); err == nil {
		return ps2, true
	}
	return ps, true
}
