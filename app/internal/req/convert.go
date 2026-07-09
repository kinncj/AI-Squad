package req

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const aiTimeout = 120 * time.Second

// aiStreamTimeout is longer than aiTimeout because streaming shows live progress, so a
// big requirement that takes minutes is watchable (and cancellable) rather than a blind
// wait — it should not be killed at 2 minutes mid-reasoning.
const aiStreamTimeout = 6 * time.Minute

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

// invokeArgs returns the argv and optional stdin for a single-command harness. copilot
// MUST use -p (non-interactive, exits after completion) with --allow-all-tools — the old
// -i starts an interactive chat TUI that never exits on its own and corrupts the screen
// under maple's alt-screen. cursor is handled separately (it needs a candidate fallback).
func invokeArgs(ai Tool, prompt string) (args []string, stdin string, ok bool) {
	switch ai.Kind {
	case "claude":
		return []string{"-p", "--output-format", "text", "--no-session-persistence", prompt}, "", true
	case "copilot":
		// -s/--silent drops copilot's stats footer (AI Credits / Tokens / Resume …) so
		// stdout is just the gherkin; --allow-all-tools is required for non-interactive.
		return []string{"-p", prompt, "--allow-all-tools", "-s", "--no-color"}, "", true
	case "opencode":
		return []string{"run"}, prompt, true
	}
	return nil, "", false
}

// StreamToGherkin runs the AI tool for requirements, invoking onLine for each line of
// output (stdout and stderr) as it arrives, and returns the full stdout when the tool
// exits. This gives the caller live progress instead of a blind wait. onLine may be
// called concurrently from two reader goroutines; StreamToGherkin serialises those calls.
func StreamToGherkin(ctx context.Context, requirements string, ai Tool, onLine func(string)) (string, error) {
	prompt := fmt.Sprintf(gherkinPrompt, requirements)

	// cursor needs the sequential candidate fallback, which can't be streamed; run it
	// blocking and replay its output line by line so the view still fills in.
	if ai.Kind == "cursor" {
		out, err := invokeCursor(ctx, ai.Path, prompt)
		text := strings.TrimSpace(string(out))
		for _, l := range strings.Split(text, "\n") {
			onLine(l)
		}
		if err != nil {
			return text, fmt.Errorf("%s: %w", ai.Label, err)
		}
		return text, nil
	}

	args, stdin, ok := invokeArgs(ai, prompt)
	if !ok {
		return "", fmt.Errorf("unsupported AI tool: %s", ai.Kind)
	}

	cctx, cancel := context.WithTimeout(ctx, aiStreamTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, ai.Path, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s: %w", ai.Label, err)
	}

	var mu sync.Mutex
	emit := func(s string) { mu.Lock(); onLine(s); mu.Unlock() }

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { // stderr: shown live, not part of the parsed content
		defer wg.Done()
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			emit(sc.Text())
		}
	}()

	var buf strings.Builder
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		buf.WriteString(line)
		buf.WriteByte('\n')
		emit(line)
	}
	wg.Wait()
	werr := cmd.Wait()

	if cctx.Err() == context.DeadlineExceeded {
		return buf.String(), fmt.Errorf("%s timed out after %s — check auth / network", ai.Label, aiStreamTimeout)
	}
	if werr != nil {
		msg := strings.TrimSpace(buf.String())
		if msg == "" {
			msg = werr.Error()
		}
		return buf.String(), fmt.Errorf("%s: %s", ai.Label, lastLines(msg, 3))
	}
	return strings.TrimSpace(buf.String()), nil
}

// lastLines returns the final n non-empty lines of s, joined — enough error context
// without dumping a whole failed transcript into the error box.
func lastLines(s string, n int) string {
	var kept []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			kept = append(kept, strings.TrimSpace(l))
		}
	}
	if len(kept) > n {
		kept = kept[len(kept)-n:]
	}
	return strings.Join(kept, " · ")
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
