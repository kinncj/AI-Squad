package req

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// SaveStory creates the full story directory structure under docs/stories/:
//
//	docs/stories/<slug>-<ts>-<idx:04d>/
//	  Story.md                  ← metadata + narrative + embedded Gherkin
//	  cucumber/
//	    <slug>.feature          ← pure Gherkin (extracted, for test runners)
//	    <slug>_steps.py         ← behave step stubs (generated once, never overwritten)
//
// now is threaded in so callers (and tests) control the timestamp.
func SaveStory(title, gherkin string, ui bool, idx int, now time.Time) (string, error) {
	slug := slugify(title)
	ts := now.Format("20060102150405")
	storyDir := filepath.Clean(fmt.Sprintf("docs/stories/%s-%s-%04d", slug, ts, idx))
	cucumberDir := filepath.Join(storyDir, "cucumber")

	if err := os.MkdirAll(cucumberDir, 0755); err != nil {
		return "", err
	}

	displayTitle := title
	if len(displayTitle) > 80 {
		displayTitle = displayTitle[:80]
	}

	storyContent := fmt.Sprintf(`---
id: "%s-%04d"
title: "%s"
epic: "%s"
priority: "medium"
ui: %t
adr_required: false
milestone: null
labels:
  - "type:feature"
  - "priority:medium"
  - "phase:discover"
issue_number: null
issue_url: null
created_at: "%s"
---

# %s

## Scenarios

`+"```gherkin\n%s\n```"+`

## Definition of Done

- [ ] Unit tests green
- [ ] Integration tests green
- [ ] Cucumber scenarios green
- [ ] ADRs linked where required

## ADR Links

(populated by adr-author agent)
`,
		slug, idx, displayTitle, slug, ui, now.Format(time.RFC3339),
		displayTitle, gherkin,
	)

	storyPath := filepath.Join(storyDir, "Story.md")
	if err := os.WriteFile(storyPath, []byte(storyContent), 0644); err != nil {
		return storyDir, err
	}

	featurePath := filepath.Join(cucumberDir, slug+".feature")
	if err := os.WriteFile(featurePath, []byte(gherkin+"\n"), 0644); err != nil {
		return storyDir, err
	}

	stepsPath := filepath.Join(cucumberDir, slug+"_steps.py")
	if _, err := os.Stat(stepsPath); os.IsNotExist(err) {
		steps := extractSteps(gherkin)
		if len(steps) > 0 {
			_ = os.WriteFile(stepsPath, []byte(generateStepDefs(displayTitle, steps)), 0644)
		}
	}

	return storyDir, nil
}

type stepLine struct {
	keyword string // given | when | then
	text    string
}

var stepLineRe = regexp.MustCompile(`^\s+(Given|When|Then|And|But)\s+(.+)$`)

// extractSteps parses unique Given/When/Then steps from a Gherkin block. And/But
// steps are normalised to the last main keyword.
func extractSteps(gherkin string) []stepLine {
	var steps []stepLine
	seen := map[string]bool{}
	last := "given"
	for _, line := range strings.Split(gherkin, "\n") {
		m := stepLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		kw, text := m[1], strings.TrimSpace(m[2])
		var decorator string
		switch kw {
		case "And", "But":
			decorator = last
		default:
			decorator = strings.ToLower(kw)
			last = decorator
		}
		key := decorator + "|" + text
		if !seen[key] {
			seen[key] = true
			steps = append(steps, stepLine{keyword: decorator, text: text})
		}
	}
	return steps
}

// generateStepDefs produces a behave step definitions skeleton.
func generateStepDefs(featureTitle string, steps []stepLine) string {
	var sb strings.Builder
	sb.WriteString("# Step definitions for: " + featureTitle + "\n")
	sb.WriteString("# Framework: behave  https://behave.readthedocs.io\n")
	sb.WriteString("# Each stub raises NotImplementedError until implemented.\n")
	sb.WriteString("from behave import given, when, then  # noqa: F401\n\n\n")
	for _, s := range steps {
		fn := stepFuncName(s.text)
		sb.WriteString(fmt.Sprintf("@%s(u%q)\n", s.keyword, s.text))
		sb.WriteString(fmt.Sprintf("def %s(context):\n", fn))
		sb.WriteString(fmt.Sprintf("    raise NotImplementedError(u\"STEP: %s %s\")\n\n\n", s.keyword, s.text))
	}
	return sb.String()
}

func stepFuncName(text string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s := strings.ToLower(text)
	s = re.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	runes := []rune(s)
	if len(runes) > 60 {
		runes = runes[:60]
		for len(runes) > 0 && runes[len(runes)-1] == '_' {
			runes = runes[:len(runes)-1]
		}
	}
	return "step_" + string(runes)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	runes := []rune(s)
	if len(runes) > 40 {
		runes = runes[:40]
		for len(runes) > 0 && !unicode.IsLetter(runes[len(runes)-1]) && !unicode.IsDigit(runes[len(runes)-1]) {
			runes = runes[:len(runes)-1]
		}
	}
	return string(runes)
}
