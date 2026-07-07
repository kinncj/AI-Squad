package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PullRequest is an open GitHub pull request surfaced in the dashboard.
type PullRequest struct {
	Number int
	Title  string
	State  string
}

// PRDetail renders `gh pr view <n>` as lines for the detail overlay.
func (s *FS) PRDetail(number int) []string {
	gh, err := exec.LookPath("gh")
	if err != nil {
		return []string{"(gh not installed)"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, gh, "pr", "view", strconv.Itoa(number),
		"--comments").CombinedOutput()
	text := strings.TrimRight(string(out), "\n")
	if err != nil && text == "" {
		return []string{"(could not load PR #" + strconv.Itoa(number) + ": " + err.Error() + ")"}
	}
	return strings.Split(text, "\n")
}

// ApprovePR submits an approving review via the gh CLI.
func (s *FS) ApprovePR(number int) error {
	gh, err := exec.LookPath("gh")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, gh, "pr", "review", strconv.Itoa(number), "--approve").CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
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
