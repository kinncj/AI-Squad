package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kinncj/maple/app/internal/tui/brand"
)

func newModel(t *testing.T) Model {
	t.Helper()
	m, err := New("v-test")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

// sized dismisses the splash and gives the model a viewport, returning the ready model.
func sized(t *testing.T, m Model) Model {
	t.Helper()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}) // dismiss splash
	return nm.(Model)
}

func TestNewStartsOnSplashWithFourPanes(t *testing.T) {
	m := newModel(t)
	if !m.splash {
		t.Error("model should start on the splash")
	}
	if got := len(m.group.Panes()); got != 4 {
		t.Errorf("dashboard has %d panes, want 4", got)
	}
}

func TestWindowSizeSetsDimensions(t *testing.T) {
	m := newModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = nm.(Model)
	if m.width != 100 || m.height != 40 {
		t.Errorf("size = %dx%d, want 100x40", m.width, m.height)
	}
}

func TestFirstKeyDismissesSplash(t *testing.T) {
	m := newModel(t)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if nm.(Model).splash {
		t.Error("a key press should dismiss the splash")
	}
}

func TestQuitFromSplashAndDashboard(t *testing.T) {
	// From splash.
	m := newModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assertQuit(t, cmd, "splash q")
	// From dashboard.
	m = sized(t, newModel(t))
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assertQuit(t, cmd, "dashboard q")
}

func assertQuit(t *testing.T, cmd tea.Cmd, ctx string) {
	t.Helper()
	if cmd == nil {
		t.Fatalf("%s: expected a quit command, got nil", ctx)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("%s: command did not produce tea.QuitMsg", ctx)
	}
}

func TestTabCyclesFocus(t *testing.T) {
	m := sized(t, newModel(t))
	if m.group.FocusIndex() != 0 {
		t.Fatalf("focus should start at 0, got %d", m.group.FocusIndex())
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if nm.(Model).group.FocusIndex() != 1 {
		t.Errorf("tab should move focus to 1, got %d", nm.(Model).group.FocusIndex())
	}
}

func TestArrowMovesSelectionOnFocusedPane(t *testing.T) {
	m := sized(t, newModel(t))
	// Render once so the focused pane knows its visible height.
	_ = m.View()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	if got := m.group.Panes()[0].Selected(); got != 1 {
		t.Errorf("down should move focused pane selection to 1, got %d", got)
	}
}

func TestViewShowsSplashThenGrid(t *testing.T) {
	t.Setenv("MAPLE_ASCII", "1") // deterministic ASCII splash in the test env
	m := newModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = nm.(Model)
	sp := m.View()
	if !strings.Contains(sp, brand.Tagline) {
		t.Error("splash view should show the brand tagline")
	}
	// Dismiss and render the dashboard shell.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	grid := nm.(Model).View()
	if !strings.Contains(grid, "Stories") {
		t.Error("grid view should show the Stories pane title")
	}
	if !strings.Contains(grid, "quit") {
		t.Error("grid view should show the footer help")
	}
	if !strings.Contains(grid, brand.Leaf) {
		t.Error("dashboard header should show the maple leaf")
	}
}

func TestViewEmptyBeforeSize(t *testing.T) {
	if out := newModel(t).View(); out != "" {
		t.Errorf("view before WindowSizeMsg should be empty, got %q", out)
	}
}
