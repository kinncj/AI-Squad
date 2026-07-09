package req

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	core "github.com/kinncj/maple/app/internal/req"
)

// ImplementArgs prepares an implement-stories run for storyDirs and returns the argv to
// launch `harness` for it: it writes the gherkin handoff + quick-launch state (side
// effects) and returns the per-harness launch command. The caller runs the argv (in a
// split pane or the current terminal). Used by the dashboard's `i` action on a story.
// ResumeArgs builds the launch argv to resume a rate-limited/paused taffy run from its
// current stage, without restarting from the beginning.
func ResumeArgs(harness, taffy, stage string) []string {
	if taffy == "" {
		taffy = "pipeline-runner implement-stories"
	}
	prompt := "/pipeline-runner " + strings.TrimPrefix(taffy, "pipeline-runner ") + "\n\n" +
		"<maple-resume>\nContinue taffy " + taffy + " from stage \"" + stage +
		"\" — do not restart from the beginning. Keep .claude/state/maple.json updated (status RUNNING → DONE/FAILED).\n</maple-resume>"
	writeQuickLaunchState(taffy, "resuming", harness)
	return buildLaunchCmd(harness, prompt, loadPinnedSessions(), true)
}

func ImplementArgs(harness string, storyDirs []string) []string {
	stories := make([]core.Story, 0, len(storyDirs))
	for _, d := range storyDirs {
		stories = append(stories, core.Story{SavedTo: d})
	}
	_ = writeImplementationHandoff(stories)
	writeQuickLaunchState("pipeline-runner implement-stories", "starting", harness)
	prompt := buildImplementationPrompt(harness, stories)
	return buildLaunchCmd(harness, prompt, loadPinnedSessions(), true)
}

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
	sb.WriteString(`
<maple-pipeline>
You were launched from MAPLE.
Keep .claude/state/maple.json updated as you work by writing (merge, never overwrite other keys):
  {"taffy":"pipeline-runner implement-stories","stage":"<current step>","phase":"<PHASE>","status":"RUNNING","updated_at":"<ISO-8601 timestamp>"}
"phase" MUST be one of the 8 canonical pipeline phases (uppercase), updated at each phase transition so the maple TUI and design portal show an accurate stepper:
  DISCOVER, ARCHITECT, PLAN, INFRA, IMPLEMENT, VALIDATE, DOCUMENT, FINAL
"stage" is the finer-grained step within a phase (e.g. "wireframe", "mockup"). Set status to "DONE" when finished, "FAILED" if you cannot complete.

Also track EACH story's progress in .claude/state/story-status.json (merge, never overwrite other keys).
Use the rich per-story form so the TUI + portal show an accurate per-story stepper:
  {"<story id>": {"status":"in_progress","phase":"IMPLEMENT"}}
  status = "in_progress" when you begin, "done" once its tests pass, "failed" if you abandon it.
  phase  = the current canonical phase for THAT story (DISCOVER…FINAL), updated as it advances.
A bare {"<story id>":"in_progress"} is also accepted (no per-story phase). Key by the story id
(e.g. "add-a-todo-item-0002") or its directory path from the handoff below.
</maple-pipeline>

<maple-progress>
Never go silent during this TAFFY implementation run:
- Post an immediate kickoff update before the first long-running tool/agent call.
- Post a concise progress heartbeat every 60-120 seconds while actively running stages.
- On each heartbeat, refresh .claude/state/maple.json updated_at and current stage.
- Every heartbeat must include concrete progress evidence:
  - changed files/artifacts since last update (explicit paths), or
  - a specific blocker that prevented changes.
- Use this heartbeat format:
  Progress: <stage name / phase>
  Done since last update: <brief>
  Current action: <brief>
  Blockers: <none or brief blocker>
  Next update: <ETA>
- Do not send heartbeat-only timestamp churn with no artifact/blocker details.
- If the stage requires writing artifacts and write access/tools are unavailable, set status FAILED with a clear error and stop.
- If blocked/waiting, state exactly what is pending and keep posting heartbeats.
- For design-review stages, continuously produce previewable artifacts (.excalidraw/.html/.svg/.png/.jpg/.md) and keep .claude/state/design-artifacts.json updated so the review portal reflects progress live.
- Before reporting DONE, verify and report concrete artifact paths for: app/domain changes, tests changes, and tests/features + step files.
- If required test or gherkin artifacts are missing, set status FAILED and report missing paths.
- If generated runtime code imports docs/, .github/, or .claude/ files by path, set status FAILED and report offending import paths.
</maple-progress>`)
	if u := activeDesignPortalURL(); u != "" {
		sb.WriteString("\n<maple-design-portal>\n")
		sb.WriteString("The MAPLE design review portal is running at: " + u + "\n")
		sb.WriteString("Open that URL (the portal root — it is a single-page app; there are no sub-paths like /wireframes/) to review approved wireframes, mockups, and identity tokens.\n")
		sb.WriteString("Design artifact FILES live under docs/design/ (wireframes/, mockups/, identity/); reference those paths, but always send users the portal root URL to review them.\n")
		sb.WriteString("</maple-design-portal>\n")
	}
	return sb.String()
}
