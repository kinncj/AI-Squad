package dashboard

import (
	"strings"
	"testing"

	"github.com/kinncj/maple/app/internal/tui/theme"
)

func TestColorizeStoryPreservesText(t *testing.T) {
	th, err := theme.Load()
	if err != nil {
		t.Fatal(err)
	}
	m := th.ActiveMode()
	in := []string{
		"# Password reset",
		"  Scenario: happy path",
		"    Given a user exists",
		"    And a session token",
		"- [x] done item",
		"plain narrative line",
	}
	out := colorizeStory(in, m)
	if len(out) != len(in) {
		t.Fatalf("colorize changed line count: %d → %d", len(in), len(out))
	}
	// Color styling is additive: every visible token must survive (ANSI presence is
	// terminal-profile dependent, so we assert on the text, not the escapes).
	for i := range in {
		for _, tok := range strings.Fields(strings.TrimSpace(in[i])) {
			if tok == "-" || tok == "[x]" { // checkbox glyph is re-rendered, skip
				continue
			}
			if !strings.Contains(out[i], tok) {
				t.Errorf("line %d dropped %q: %q", i, tok, out[i])
			}
		}
	}
}
