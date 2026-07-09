package brand

import (
	"strings"
	"testing"

	"github.com/kinncj/maple/app/internal/tui/theme"
)

func TestWordmarkIsMultilineAndTrimmed(t *testing.T) {
	w := Wordmark()
	if w == "" {
		t.Fatal("wordmark is empty")
	}
	if strings.HasSuffix(w, "\n") {
		t.Error("wordmark should have trailing newlines trimmed")
	}
	if lines := strings.Count(w, "\n") + 1; lines != 4 {
		t.Errorf("wordmark = %d lines, want 4", lines)
	}
}

func TestRenderStylesWordmark(t *testing.T) {
	th, _ := theme.Load()
	out := Render(th.ActiveMode())
	if out == "" {
		t.Fatal("render is empty")
	}
	// Styling wraps the content but must preserve the ASCII glyphs.
	if !strings.Contains(out, "|") {
		t.Errorf("styled wordmark should still contain the ASCII art")
	}
}
