package req

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	core "github.com/kinncj/maple/app/internal/req"
)

// taffyPathForHarness returns the implement-stories workflow path for a harness.
func taffyPathForHarness(kind string) string {
	switch kind {
	case "opencode":
		return ".opencode/taffy/implement-stories.yaml"
	case "cursor":
		return ".cursor/taffy/implement-stories.yaml"
	default:
		return ".claude/taffy/implement-stories.yaml"
	}
}

// harnessInstructionMarkdowns returns the governance markdown files for a harness.
func harnessInstructionMarkdowns(harness string) []string {
	base := []string{
		"AGENTS.md",
		".github/copilot-instructions.md",
		".github/instructions/stories.instructions.md",
	}
	switch harness {
	case "opencode":
		return append([]string{"OPENCODE.md"}, base...)
	case "cursor":
		return append([]string{"CURSOR.md"}, base...)
	case "copilot":
		return append([]string{"COPILOT.md"}, base...)
	default:
		return append([]string{"CLAUDE.md"}, base...)
	}
}

func governanceBootstrapBlock(harness string) string {
	files := harnessInstructionMarkdowns(harness)
	var sb strings.Builder
	sb.WriteString("\n<maple-governance-bootstrap>\n")
	sb.WriteString("Before running any TAFFY stage, read and enforce these markdown files in order:\n")
	for _, f := range files {
		sb.WriteString("- " + f + "\n")
	}
	sb.WriteString("Treat them as mandatory constraints for every delegated skill/agent call.\n")
	sb.WriteString("Runtime code and tests must never import from docs/, .github/, or .claude/ paths.\n")
	sb.WriteString("Copying or adapting artifact content from docs into app/test code is allowed; path-based imports/references are not.\n")
	sb.WriteString("</maple-governance-bootstrap>\n")
	return sb.String()
}

func activeDesignPortalURL() string {
	b, err := os.ReadFile(".claude/state/design-portal.url")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// writeQuickLaunchState writes an initial RUNNING pipeline state to maple.json,
// merging with existing content so skill-owned keys are preserved.
func writeQuickLaunchState(skill, stage, harness string) {
	_ = os.MkdirAll(".claude/state", 0o755)
	merged := map[string]interface{}{}
	if raw, err := os.ReadFile(".claude/state/maple.json"); err == nil {
		_ = json.Unmarshal(raw, &merged)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	merged["taffy"] = skill
	merged["stage"] = stage
	merged["status"] = "RUNNING"
	merged["harness"] = harness
	merged["started_at"] = now
	merged["updated_at"] = now
	if data, err := json.MarshalIndent(merged, "", "  "); err == nil {
		_ = os.WriteFile(".claude/state/maple.json", append(data, '\n'), 0o644)
	}
}

// writeImplementationHandoff records the generated stories for the taffy runner.
func writeImplementationHandoff(stories []core.Story) error {
	type handoffStory struct {
		Title string `json:"title"`
		Path  string `json:"path"`
		UI    bool   `json:"ui"`
	}
	var out []handoffStory
	for _, s := range stories {
		if strings.TrimSpace(s.SavedTo) == "" {
			continue
		}
		out = append(out, handoffStory{Title: s.Title, Path: s.SavedTo, UI: s.UI})
	}
	_ = os.MkdirAll(".claude/state", 0o755)
	data, err := json.MarshalIndent(map[string]interface{}{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"stories":      out,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(".claude/state/gherkin-handoff.json", append(data, '\n'), 0o644)
}

// loadPinnedSessions reads the pinned session ids per harness (best-effort).
func loadPinnedSessions() map[string]string {
	out := map[string]string{}
	if raw, err := os.ReadFile(".claude/state/sessions.json"); err == nil {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

// buildLaunchCmd builds the argv to launch a harness with an opening prompt.
func buildLaunchCmd(tool, cmd string, pinned map[string]string, dangerous bool) []string {
	pinnedID := pinned[tool]
	var args []string
	switch tool {
	case "claude":
		args = []string{"claude"}
		if dangerous {
			args = append(args, "--dangerously-skip-permissions")
		}
		if pinnedID != "" {
			args = append(args, "--resume", pinnedID)
		}
		if cmd != "" {
			args = append(args, cmd)
		}
	case "opencode":
		if cmd != "" {
			args = []string{"opencode", cmd}
		} else {
			args = []string{"opencode"}
		}
	case "copilot":
		args = []string{"copilot"}
		if dangerous {
			args = append(args, "--allow-all")
		}
		if pinnedID != "" {
			args = append(args, "--resume="+pinnedID)
		}
		if cmd != "" {
			args = append(args, "-i", cmd)
		}
	case "cursor":
		cursorBin := "cursor-agent"
		if p, _ := exec.LookPath("cursor-agent"); p == "" {
			cursorBin = "cursor"
		}
		args = []string{cursorBin}
		if cmd != "" {
			if dangerous {
				args = append(args, "--yolo", "--sandbox", "disabled", "--approve-mcps")
			}
			args = append(args, "-p", "--output-format", "text", "--trust", cmd)
		}
	default:
		args = []string{tool}
	}
	if rtkPath, err := exec.LookPath("rtk"); err == nil && rtkPath != "" {
		args = append([]string{"env", "RTK_HOOK_AUDIT=1"}, args...)
	}
	return args
}

// buildImplementationPrompt assembles the /pipeline-runner implement-stories launch
// prompt with governance + progress contracts for the given harness.
func buildImplementationPrompt(harness string, stories []core.Story) string {
	var paths []string
	for _, s := range stories {
		if strings.TrimSpace(s.SavedTo) != "" {
			paths = append(paths, s.SavedTo)
		}
	}
	var sb strings.Builder
	sb.WriteString("/pipeline-runner implement-stories")
	sb.WriteString(governanceBootstrapBlock(harness))
	sb.WriteString("\n\n<maple-gherkin-handoff>\n")
	sb.WriteString("These stories were generated and approved via `maple req`.\n")
	sb.WriteString("Implement and test only these story directories:\n")
	for _, p := range paths {
		sb.WriteString("- " + p + "\n")
	}
	sb.WriteString("Use `.claude/state/gherkin-handoff.json` as the source of truth for targets.\n")
	sb.WriteString("Do not regenerate or rewrite stories. Do not rerun spec-kit.\n")
	sb.WriteString("Run orchestrator + delegation to execute implementation and tests for these stories.\n")
	sb.WriteString("Enforce the full 8-phase pipeline and Karpathy gate.\n")
	sb.WriteString("</maple-gherkin-handoff>\n")
	sb.WriteString("\n<maple-governance>\n")
	sb.WriteString("Hard requirements:\n")
	sb.WriteString("- Strictly follow repository instruction files: harness root markdown (`CLAUDE.md`/`OPENCODE.md`/`CURSOR.md`/`COPILOT.md`), `.github/copilot-instructions.md`, and `.github/instructions/stories.instructions.md`.\n")
	sb.WriteString("- Preserve the BusinessRepo model and required repository layout defined by those files.\n")
	sb.WriteString("- Never place acceptance tests in `/app`; write them under `/tests` and `/tests/features`.\n")
	sb.WriteString("- Runtime code and tests must never import from `docs/`, `.github/`, or `.claude/` paths.\n")
	sb.WriteString("- Copying/adapting approved artifacts into app/test source is allowed; direct path imports/references to docs are not.\n")
	sb.WriteString("- Respect the existing Cucumber stack already present in the generated stories.\n")
	sb.WriteString("- If a story has `cucumber/*_steps.py`, continue with Python behave-style steps and do NOT switch to TypeScript `@cucumber/cucumber`.\n")
	sb.WriteString("- Do not invent alternative test frameworks when story artifacts already define one.\n")
	sb.WriteString("</maple-governance>\n")
	if u := activeDesignPortalURL(); u != "" {
		sb.WriteString("\n<maple-design-portal>\n")
		sb.WriteString("The MAPLE design review portal is running at: " + u + "\n")
		sb.WriteString("Browse docs/design/ artifacts there to reference approved wireframes, mockups, and identity tokens.\n")
		sb.WriteString("Wireframes are at: " + u + "/wireframes/\n")
		sb.WriteString("</maple-design-portal>\n")
	}
	return sb.String()
}
