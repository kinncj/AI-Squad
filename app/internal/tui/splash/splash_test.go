package splash

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/kinncj/maple/app/internal/tui/brand"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

func TestRenderContainsBrandAndVersion(t *testing.T) {
	th, _ := theme.Load()
	out := Render(80, 24, "v9.9.9", th.ActiveMode())
	if !strings.Contains(out, "v9.9.9") {
		t.Error("splash should show the version")
	}
	if !strings.Contains(out, brand.Leaf) {
		t.Error("splash should show the maple leaf")
	}
	if !strings.Contains(out, brand.Tagline) {
		t.Error("splash should show the tagline")
	}
}

func TestRenderFillsViewport(t *testing.T) {
	th, _ := theme.Load()
	out := Render(80, 24, "v1", th.ActiveMode())
	if h := lipgloss.Height(out); h != 24 {
		t.Errorf("splash height = %d, want 24", h)
	}
	if w := lipgloss.Width(out); w != 80 {
		t.Errorf("splash width = %d, want 80", w)
	}
}

func TestRenderDegradesWithoutSize(t *testing.T) {
	th, _ := theme.Load()
	if out := Render(0, 0, "v1", th.ActiveMode()); out == "" {
		t.Error("splash should still render body when size is unknown")
	}
}
