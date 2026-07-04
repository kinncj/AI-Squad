package req

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseStoriesMultiBlock(t *testing.T) {
	out := "=== STORY: Export CSV ===\nFeature: Export\n\n  Scenario: happy\n    Given a\n    When b\n    Then c\n\n=== STORY: Filter Rows ===\nFeature: Filter\n\n  Scenario: edge\n    Given x\n    Then y\n"
	stories := ParseStories(out)
	if len(stories) != 2 {
		t.Fatalf("want 2 stories, got %d", len(stories))
	}
	if stories[0].Title != "Export CSV" || stories[1].Title != "Filter Rows" {
		t.Errorf("titles wrong: %q, %q", stories[0].Title, stories[1].Title)
	}
	if !strings.HasPrefix(stories[0].Gherkin, "Feature: Export") {
		t.Errorf("story 0 gherkin should start with Feature: Export, got %q", stories[0].Gherkin)
	}
}

func TestParseStoriesSingleNoDelimiter(t *testing.T) {
	out := "```gherkin\nFeature: Lonely\n\n  Scenario: s\n    Given g\n```\nTokens: 42\n"
	stories := ParseStories(out)
	if len(stories) != 1 {
		t.Fatalf("want 1 story, got %d", len(stories))
	}
	if stories[0].Title != "Lonely" {
		t.Errorf("title should come from Feature:, got %q", stories[0].Title)
	}
	if strings.Contains(stories[0].Gherkin, "```") || strings.Contains(stories[0].Gherkin, "Tokens:") {
		t.Errorf("fences/telemetry not stripped: %q", stories[0].Gherkin)
	}
}

func TestExtractFeatureTitleFallback(t *testing.T) {
	if got := extractFeatureTitle("no feature here"); got != "Untitled" {
		t.Errorf("want Untitled, got %q", got)
	}
}

func TestInferUIStory(t *testing.T) {
	if !InferUIStory("build a React frontend", "Login screen", "Given a button") {
		t.Error("should detect UI story")
	}
	if InferUIStory("nightly batch job", "Aggregate totals", "Given a cron") {
		t.Error("should not flag a backend story as UI")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Export CSV Data!":    "export-csv-data",
		"  Trailing  spaces ": "trailing-spaces",
		"UPPER_snake":         "upper-snake",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractStepsDedupAndNormalise(t *testing.T) {
	gherkin := "  Scenario: s\n    Given a user\n    And a session\n    When they click\n    Then it works\n    And it works\n"
	steps := extractSteps(gherkin)
	// "And it works" dups "Then it works" (both then|it works) → deduped.
	if len(steps) != 4 {
		t.Fatalf("want 4 unique steps, got %d: %+v", len(steps), steps)
	}
	if steps[1].keyword != "given" { // "And a session" normalises to given
		t.Errorf("And should inherit given, got %q", steps[1].keyword)
	}
}

func TestGenerateStepDefs(t *testing.T) {
	defs := generateStepDefs("My Feature", []stepLine{{keyword: "given", text: "a user exists"}})
	if !strings.Contains(defs, "from behave import given, when, then") {
		t.Error("missing behave import")
	}
	if !strings.Contains(defs, `@given(u"a user exists")`) {
		t.Errorf("missing decorator, got:\n%s", defs)
	}
	if !strings.Contains(defs, "def step_a_user_exists(context):") {
		t.Errorf("missing func def, got:\n%s", defs)
	}
}

func TestSaveStoryWritesTree(t *testing.T) {
	cwd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	gherkin := "Feature: Export\n\n  Scenario: s\n    Given a user\n    When they export\n    Then a file"
	dir, err := SaveStory("Export CSV", gherkin, true, 1, now)
	if err != nil {
		t.Fatalf("SaveStory: %v", err)
	}
	wantDir := "docs/stories/export-csv-20260704120000-0001"
	if dir != wantDir {
		t.Errorf("dir = %q, want %q", dir, wantDir)
	}
	story, err := os.ReadFile(filepath.Join(dir, "Story.md"))
	if err != nil {
		t.Fatalf("Story.md not written: %v", err)
	}
	for _, want := range []string{`ui: true`, `title: "Export CSV"`, "```gherkin", "type:feature"} {
		if !strings.Contains(string(story), want) {
			t.Errorf("Story.md missing %q", want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "cucumber", "export-csv.feature")); err != nil {
		t.Error("feature file not written")
	}
	steps, err := os.ReadFile(filepath.Join(dir, "cucumber", "export-csv_steps.py"))
	if err != nil {
		t.Fatalf("steps file not written: %v", err)
	}
	if !strings.Contains(string(steps), "NotImplementedError") {
		t.Error("steps stub missing NotImplementedError")
	}
}
