package req

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const aiTimeout = 120 * time.Second

// Tool is one AI harness capable of requirements → Gherkin.
type Tool struct {
	Label string // display name, e.g. "Claude Code"
	Kind  string // "claude" | "copilot" | "opencode" | "cursor"
	Path  string // resolved binary path
}

// AvailableTools returns the req-capable harnesses found on PATH, in canonical order.
func AvailableTools() []Tool {
	var out []Tool
	add := func(label, kind string, names ...string) {
		for _, n := range names {
			if p, err := exec.LookPath(n); err == nil {
				out = append(out, Tool{Label: label, Kind: kind, Path: p})
				return
			}
		}
	}
	add("Claude Code", "claude", "claude")
	add("GitHub Copilot", "copilot", "copilot")
	add("OpenCode", "opencode", "opencode")
	add("Cursor Agent", "cursor", "cursor-agent", "cursor")
	return out
}

// runCtx runs name+args under a deadline, returning combined output.
func runCtx(parent context.Context, d time.Duration, stdin, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, d)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("%s timed out after %s — check auth / network", name, d)
	}
	return out, err
}

const gherkinPrompt = `You are a Gherkin authoring expert. Analyze the requirements below.

Determine if this describes:
  A) A single user story → output ONE story block
  B) An epic (multiple related stories) → split into separate stories, one block each

For EACH story use this exact format:
=== STORY: <Concise Story Title> ===
Feature: <feature name>

  @story @priority:medium
  Scenario: <happy path name>
    Given ...
    When ...
    Then ...

  Scenario: <edge case name>
    Given ...
    When ...
    Then ...

Rules:
- Use Feature, Background (if needed), Scenario, Given/When/Then/And/But
- Plain language that non-technical stakeholders can understand
- Separate scenarios for happy path and key edge cases
- Preserve concrete technical requirements from the input (frameworks, storage, APIs, design systems, visual style) as explicit, testable behavior in Background or Scenario steps
- If requirements describe a UI, SPA, frontend, design/look-and-feel, set assumptions and scenarios that clearly indicate a visual feature
- Output ONLY the === STORY: === blocks — no explanation, no prose
- Do not output markdown fences, token usage, change summaries, request stats, or any telemetry

Requirements:
%s`

// ConvertToGherkin sends requirements to the AI tool and returns the raw Gherkin.
func ConvertToGherkin(ctx context.Context, requirements string, ai Tool) (string, error) {
	prompt := fmt.Sprintf(gherkinPrompt, requirements)
	out, err := invokeAI(ctx, ai, prompt)
	if err != nil {
		return "", fmt.Errorf("%s: %w", ai.Label, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// invokeAI runs the selected AI tool and returns its stdout. The prompt is passed as
// a positional arg (not stdin) where possible to avoid inheriting a raw-mode terminal.
func invokeAI(ctx context.Context, ai Tool, prompt string) ([]byte, error) {
	var (
		out []byte
		err error
	)
	switch ai.Kind {
	case "claude":
		out, err = runCtx(ctx, aiTimeout, "", ai.Path, "-p", "--output-format", "text", "--no-session-persistence", prompt)
	case "copilot":
		out, err = runCtx(ctx, aiTimeout, "", ai.Path, "-i", prompt)
	case "opencode":
		out, err = runCtx(ctx, aiTimeout, prompt, ai.Path, "run")
	case "cursor":
		return invokeCursor(ctx, ai.Path, prompt)
	default:
		return nil, fmt.Errorf("unsupported AI tool: %s", ai.Kind)
	}

	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return out, nil
}

func invokeCursor(ctx context.Context, binPath, prompt string) ([]byte, error) {
	type attempt struct {
		args  []string
		stdin string
	}
	candidates := []attempt{
		{args: []string{"-p", "--output-format", "text", "--trust", prompt}},
		{args: []string{"-p", prompt}},
		{args: []string{"--prompt", prompt}},
		{args: []string{"run", prompt}},
		{args: []string{"run"}, stdin: prompt},
	}

	var lastErr string
	for _, c := range candidates {
		out, err := runCtx(ctx, aiTimeout, c.stdin, binPath, c.args...)
		if err == nil {
			return out, nil
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		lastErr = msg
	}
	if lastErr == "" {
		lastErr = "cursor-agent invocation failed"
	}
	return nil, fmt.Errorf("%s", lastErr)
}
