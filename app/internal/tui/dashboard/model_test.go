package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kinncj/maple/app/internal/state"
	"github.com/kinncj/maple/app/internal/tui/brand"
)

// fakeStore returns fixed project state for deterministic dashboard tests.
type fakeStore struct{ n int }

func (f fakeStore) Stories() []state.Story {
	out := make([]state.Story, f.n)
	for i := range out {
		out[i] = state.Story{ID: "story-" + itoa(i+1), Phase: "implement"}
	}
	return out
}
func (f fakeStore) Sessions() []state.Session {
	return []state.Session{
		{Title: "Standardize tracking", Source: "claude", ToolCount: 293},
		{Title: "deadbeef…cafebabe", Source: "copilot", ToolCount: 42},
	}
}
func (f fakeStore) PullRequests() []state.PullRequest {
	return []state.PullRequest{{Number: 22, Title: "rework", State: "OPEN"}}
}
func (f fakeStore) Tests() []state.Test {
	return []state.Test{{Path: "app/x_test.go", Framework: "go"}}
}
func (f fakeStore) DesignTree() []string    { return []string{"📁 wireframes", "  📄 home.md"} }
func (f fakeStore) LogLines(n int) []string { return []string{"ts=12:00  agent=qa"} }
func (f fakeStore) GitChanges() []string    { return []string{"── status ──", " M app/x.go"} }
func (f fakeStore) PipelineLines() []string { return []string{"status  RUNNING", "stage  IMPLEMENT"} }
func (f fakeStore) Skills() []string        { return []string{"gh-issues", "humanizer"} }
func (f fakeStore) ProjectName() string     { return "test-project" }
func (f fakeStore) TaffyCount() int         { return 5 }
func (f fakeStore) PipelineStatus() string  { return "DONE" }

func newModel(t *testing.T) Model {
	t.Helper()
	m, err := New("v-test", fakeStore{n: 12})
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

func TestSplashAutoDismisses(t *testing.T) {
	m := newModel(t)
	nm, cmd := m.Update(splashDoneMsg{})
	if nm.(Model).splash {
		t.Error("splashDoneMsg should dismiss the splash")
	}
	if cmd == nil {
		t.Error("dismiss should return a ClearScreen command to wipe any image")
	}
}

func TestHelpTogglesAndAnyKeyCloses(t *testing.T) {
	m := sized(t, newModel(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = nm.(Model)
	if !m.showHelp {
		t.Fatal("? should open help")
	}
	if !strings.Contains(m.View(), "Keybindings") {
		t.Error("help view should list keybindings")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if nm.(Model).showHelp {
		t.Error("any key should close help")
	}
}

func TestFocusShortcutKeys(t *testing.T) {
	m := sized(t, newModel(t))
	for key, want := range map[string]int{"a": paneSessions, "p": panePRs, "Q": paneQA, "s": paneStories} {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if got := nm.(Model).group.FocusIndex(); got != want {
			t.Errorf("key %q focused pane %d, want %d", key, got, want)
		}
	}
}

func TestFilterModeTypesAndCancels(t *testing.T) {
	m := sized(t, newModel(t))
	_ = m.View() // populate pane heights
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = nm.(Model)
	if !m.filtering {
		t.Fatal("/ should enter filter mode")
	}
	for _, r := range "story-1" {
		nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(Model)
	}
	if m.filterBuf != "story-1" {
		t.Errorf("filter buffer = %q, want story-1", m.filterBuf)
	}
	if !strings.Contains(m.View(), "story-1") {
		t.Error("footer should echo the filter buffer")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).filtering {
		t.Error("esc should exit filter mode")
	}
}

func TestPendingOverlayKeySetsStatus(t *testing.T) {
	m := sized(t, newModel(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	if !strings.Contains(nm.(Model).status, "Design Review") {
		t.Errorf("D should set a status about Design Review, got %q", nm.(Model).status)
	}
}

func TestFullscreenDesignToggle(t *testing.T) {
	m := sized(t, newModel(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = nm.(Model)
	if m.fullscreen != fsDesign {
		t.Fatalf("d should open Design fullscreen, got %d", m.fullscreen)
	}
	if !strings.Contains(m.View(), "wireframes") {
		t.Error("Design fullscreen should render the design tree")
	}
	// d again closes it.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if nm.(Model).fullscreen != fsNone {
		t.Error("d again should close Design")
	}
}

func TestFullscreenLogsAndEscClose(t *testing.T) {
	m := sized(t, newModel(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = nm.(Model)
	if m.fullscreen != fsLogs {
		t.Fatalf("l should open Logs fullscreen, got %d", m.fullscreen)
	}
	if !strings.Contains(m.View(), "agent=qa") {
		t.Error("Logs fullscreen should render log lines")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).fullscreen != fsNone {
		t.Error("esc should close the fullscreen pane")
	}
}

func TestEnterOpensStoryDetail(t *testing.T) {
	// Point a story at a real file so the detail overlay has content.
	m, err := New("v-test", detailStore{})
	if err != nil {
		t.Fatal(err)
	}
	m = sized(t, m)
	_ = m.View()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.detail == nil {
		t.Fatal("Enter on Stories should open a detail overlay")
	}
	if !strings.Contains(m.View(), "Story ·") {
		t.Error("detail overlay should show the story title")
	}
	// esc closes it.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).detail != nil {
		t.Error("esc should close the detail overlay")
	}
}

func TestGitChangesOverlay(t *testing.T) {
	m := sized(t, newModel(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	m = nm.(Model)
	if m.detail == nil {
		t.Fatal("C should open the git changes overlay")
	}
	if !strings.Contains(m.View(), "Git Changes") {
		t.Error("git changes overlay should show its title")
	}
	if !strings.Contains(m.View(), "app/x.go") {
		t.Error("git changes overlay should show the status output")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).detail != nil {
		t.Error("esc should close git changes")
	}
}

// detailStore backs the Enter test with a story whose Path points at this test file.
type detailStore struct{ fakeStore }

func (detailStore) Stories() []state.Story {
	return []state.Story{{ID: "auth-0001", Path: "model_test.go"}}
}

func TestFullscreenSwitchDToL(t *testing.T) {
	m := sized(t, newModel(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if nm.(Model).fullscreen != fsLogs {
		t.Error("l while Design is open should switch to Logs")
	}
}
