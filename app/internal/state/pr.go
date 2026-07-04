package state

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"
)

// PullRequest is an open GitHub pull request surfaced in the dashboard.
type PullRequest struct {
	Number int
	Title  string
	State  string
}

// PullRequests lists open pull requests via the gh CLI. It returns nil when gh is
// absent or the call fails — the dashboard renders an empty pane in that case.
func (s *FS) PullRequests() []PullRequest {
	gh, err := exec.LookPath("gh")
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, gh, "pr", "list", "--json", "number,title,state", "--limit", "20")
	cmd.Dir = s.Root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parsePRs(out)
}

// parsePRs decodes `gh pr list --json number,title,state` output. Invalid JSON
// yields nil.
func parsePRs(data []byte) []PullRequest {
	var raw []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	out := make([]PullRequest, len(raw))
	for i, r := range raw {
		out[i] = PullRequest{Number: r.Number, Title: r.Title, State: r.State}
	}
	return out
}
