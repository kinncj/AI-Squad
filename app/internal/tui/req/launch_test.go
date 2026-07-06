package req

import (
	"os"
	"strings"
	"testing"

	core "github.com/kinncj/maple/app/internal/req"
)

func TestTaffyPathForHarness(t *testing.T) {
	cases := map[string]string{
		"claude":   ".claude/taffy/implement-stories.yaml",
		"copilot":  ".claude/taffy/implement-stories.yaml",
		"opencode": ".opencode/taffy/implement-stories.yaml",
		"cursor":   ".cursor/taffy/implement-stories.yaml",
	}
	for kind, want := range cases {
		if got := taffyPathForHarness(kind); got != want {
			t.Errorf("taffyPathForHarness(%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestHarnessInstructionMarkdowns(t *testing.T) {
	cases := map[string]string{
		"claude":   "CLAUDE.md",
		"opencode": "OPENCODE.md",
		"cursor":   "CURSOR.md",
		"copilot":  "COPILOT.md",
	}
	for harness, wantHead := range cases {
		files := harnessInstructionMarkdowns(harness)
		if files[0] != wantHead {
			t.Errorf("%s: head = %q, want %q", harness, files[0], wantHead)
		}
		// Shared governance files always follow.
		joined := strings.Join(files, ",")
		for _, want := range []string{"AGENTS.md", ".github/copilot-instructions.md", ".github/instructions/stories.instructions.md"} {
			if !strings.Contains(joined, want) {
				t.Errorf("%s: missing %q in %v", harness, want, files)
			}
		}
	}
}

func TestGovernanceBootstrapBlock(t *testing.T) {
	block := governanceBootstrapBlock("opencode")
	for _, want := range []string{
		"<maple-governance-bootstrap>",
		"OPENCODE.md",
		"never import from docs/, .github/, or .claude/ paths",
		"</maple-governance-bootstrap>",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("governance block missing %q", want)
		}
	}
}

func TestBuildLaunchCmd(t *testing.T) {
	// rtk may or may not be on PATH; strip the leading `env RTK_HOOK_AUDIT=1` wrapper
	// so the assertions are stable across machines.
	strip := func(a []string) []string {
		if len(a) >= 2 && a[0] == "env" && strings.HasPrefix(a[1], "RTK_HOOK_AUDIT=") {
			return a[2:]
		}
		return a
	}

	t.Run("claude with pinned + prompt", func(t *testing.T) {
		got := strip(buildLaunchCmd("claude", "/pipeline-runner", map[string]string{"claude": "sess1"}, true))
		want := []string{"claude", "--dangerously-skip-permissions", "--resume", "sess1", "/pipeline-runner"}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("copilot no pin, not dangerous", func(t *testing.T) {
		got := strip(buildLaunchCmd("copilot", "hi", nil, false))
		want := []string{"copilot", "-i", "hi"}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("opencode passes prompt positionally", func(t *testing.T) {
		got := strip(buildLaunchCmd("opencode", "do it", nil, true))
		want := []string{"opencode", "do it"}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("unknown tool falls back to bare name", func(t *testing.T) {
		got := strip(buildLaunchCmd("mystery", "x", nil, false))
		if len(got) != 1 || got[0] != "mystery" {
			t.Errorf("got %v, want [mystery]", got)
		}
	})
}

func TestBuildImplementationPrompt(t *testing.T) {
	stories := []core.Story{
		{Title: "A", SavedTo: "docs/stories/a-1"},
		{Title: "B", SavedTo: ""}, // unsaved → excluded from the path list
		{Title: "C", SavedTo: "docs/stories/c-3"},
	}
	prompt := buildImplementationPrompt("claude", stories)
	for _, want := range []string{
		"/pipeline-runner implement-stories",
		"<maple-governance-bootstrap>",
		"<maple-gherkin-handoff>",
		"- docs/stories/a-1",
		"- docs/stories/c-3",
		"gherkin-handoff.json",
		"<maple-governance>",
		"8-phase pipeline",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if strings.Contains(prompt, "docs/stories/b") {
		t.Error("unsaved story B should not appear in the path list")
	}
}

func TestBuildImplementationPromptDesignPortal(t *testing.T) {
	cwd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	os.MkdirAll(".claude/state", 0o755)
	os.WriteFile(".claude/state/design-portal.url", []byte("http://localhost:4321\n"), 0o644)

	prompt := buildImplementationPrompt("claude", []core.Story{{Title: "A", SavedTo: "docs/stories/a-1"}})
	if !strings.Contains(prompt, "<maple-design-portal>") || !strings.Contains(prompt, "http://localhost:4321") {
		t.Error("design portal block should be present when the url file exists")
	}
	if !strings.Contains(prompt, "http://localhost:4321/wireframes/") {
		t.Error("wireframes url should be present")
	}
}
