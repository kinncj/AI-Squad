package splash

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/kinncj/maple/app/internal/tui/brand"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

func TestAsciiFrameShowsBrandTaglineVersion(t *testing.T) {
	t.Setenv("MAPLE_ASCII", "1") // force the ASCII path regardless of terminal
	th, _ := theme.Load()
	out := Render(th.ActiveMode(), 120, 40, "maple v9.9.9")
	if !strings.Contains(out, "maple v9.9.9") {
		t.Error("splash should show the version")
	}
	if !strings.Contains(out, brand.Tagline) {
		t.Error("splash should show the tagline")
	}
	// brand.Render emits the ASCII wordmark, which contains pipe characters.
	if !strings.Contains(out, "|") {
		t.Error("splash should show the ASCII wordmark")
	}
}

func TestAsciiFrameFillsViewport(t *testing.T) {
	t.Setenv("MAPLE_ASCII", "1")
	th, _ := theme.Load()
	out := Render(th.ActiveMode(), 100, 30, "v1")
	if h := lipgloss.Height(out); h != 30 {
		t.Errorf("splash height = %d, want 30", h)
	}
	if w := lipgloss.Width(out); w != 100 {
		t.Errorf("splash width = %d, want 100", w)
	}
}

func TestAsciiLeafFitsNarrowWidthAndHeight(t *testing.T) {
	// The hand-drawn art must subsample to fit both bounds.
	leaf := asciiLeaf(40, 12)
	lines := strings.Split(leaf, "\n")
	if len(lines) > 12 {
		t.Errorf("subsampled art has %d rows, want <= 12", len(lines))
	}
	for _, line := range lines {
		if w := len([]rune(line)); w > 40 {
			t.Errorf("subsampled line width %d exceeds 40: %q", w, line)
		}
	}
}

func TestAsciiLeafUnchangedWhenItFits(t *testing.T) {
	// A very large budget returns the art unchanged.
	full := asciiLeaf(10000, 10000)
	if strings.TrimRight(full, "\n") != strings.TrimRight(asciiArt, "\n") {
		t.Error("asciiLeaf should return the art unchanged when it already fits")
	}
}

func TestForceASCIIEnvSelectsAsciiPath(t *testing.T) {
	t.Setenv("MAPLE_ASCII", "1")
	if !forceASCII() {
		t.Error("MAPLE_ASCII=1 should force the ASCII path")
	}
	// inlineImage must decline when ASCII is forced.
	if _, ok := inlineImage(120, 40); ok {
		t.Error("inlineImage should return ok=false when MAPLE_ASCII is set")
	}
}

func TestRenderDegradesWithoutSize(t *testing.T) {
	t.Setenv("MAPLE_ASCII", "1")
	th, _ := theme.Load()
	if out := Render(th.ActiveMode(), 0, 0, "v1"); out == "" {
		t.Error("splash should still render body when size is unknown")
	}
}
