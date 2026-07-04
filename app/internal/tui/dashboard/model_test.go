package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kinncj/maple/app/internal/spawn"
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
func (f fakeStore) DesignArtifacts(id string) []state.Artifact {
	return []state.Artifact{{Path: "docs/design/wireframes/" + id + ".wireframe.md", Kind: "wireframes", Status: "pending"}}
}
func (f fakeStore) ApprovalPending() string { return "" }
func (f fakeStore) ApproveGate() error      { return nil }
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
	if !strings.Contains(grid, "[Tab] cycle") {
		t.Error("grid view should show the footer keybindings")
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
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	if !strings.Contains(nm.(Model).status, "ship-safe") {
		t.Errorf("S should set a status about ship-safe, got %q", nm.(Model).status)
	}
}

func TestDesignReviewOverlay(t *testing.T) {
	m := sized(t, newModel(t))
	_ = m.View()
	// D on the Stories pane (default focus) opens the review overlay.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	m = nm.(Model)
	if m.detail == nil || m.reviewStory == "" {
		t.Fatal("D on Stories should open the design review overlay")
	}
	v := m.View()
	if !strings.Contains(v, "Design Review") || !strings.Contains(v, "pending") {
		t.Error("review overlay should show its title and artifact status")
	}
	// esc closes and clears review mode.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).detail != nil || nm.(Model).reviewStory != "" {
		t.Error("esc should close review and clear reviewStory")
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

// typeCommand enters command mode, types the given text, and presses Enter.
func typeCommand(t *testing.T, m Model, text string) (Model, tea.Cmd) {
	t.Helper()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = nm.(Model)
	if !m.commanding {
		t.Fatal(": should enter command mode")
	}
	for _, r := range text {
		nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(Model)
	}
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return nm.(Model), cmd
}

func TestCommandQuit(t *testing.T) {
	m := sized(t, newModel(t))
	_, cmd := typeCommand(t, m, "q")
	assertQuit(t, cmd, ":q")
}

func TestCommandReload(t *testing.T) {
	m := sized(t, newModel(t))
	m2, _ := typeCommand(t, m, "reload")
	if m2.commanding {
		t.Error("command mode should end after Enter")
	}
	if m2.status != "reloaded" {
		t.Errorf("status = %q, want reloaded", m2.status)
	}
}

func TestCommandHelpAndUnknown(t *testing.T) {
	m := sized(t, newModel(t))
	m2, _ := typeCommand(t, m, "help")
	if !m2.showHelp {
		t.Error(":help should open help")
	}
	m3, _ := typeCommand(t, sized(t, newModel(t)), "bogus")
	if !strings.Contains(m3.status, "unknown command") {
		t.Errorf(":bogus should report unknown, got %q", m3.status)
	}
}

func TestCommandThemeSwitch(t *testing.T) {
	m := sized(t, newModel(t))
	if m.theme.Name != "tokyo-night" {
		t.Fatalf("default theme = %q, want tokyo-night", m.theme.Name)
	}
	m2, _ := typeCommand(t, m, "theme gruvbox")
	if m2.theme.Name != "gruvbox" {
		t.Errorf(":theme gruvbox should switch, got %q", m2.theme.Name)
	}
	if !strings.Contains(m2.View(), "gruvbox") {
		t.Error("header should show the active theme name")
	}
	// unknown theme reports and does not switch.
	m3, _ := typeCommand(t, m2, "theme bogus")
	if m3.theme.Name != "gruvbox" || !strings.Contains(m3.status, "unknown theme") {
		t.Errorf("bad theme should keep current + warn, got %q status=%q", m3.theme.Name, m3.status)
	}
}

func TestCommandEscCancels(t *testing.T) {
	m := sized(t, newModel(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).commanding {
		t.Error("esc should cancel command mode")
	}
}

func TestCommandBufferEchoedInFooter(t *testing.T) {
	m := sized(t, newModel(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = nm.(Model)
	for _, r := range "theme" {
		nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(Model)
	}
	if !strings.Contains(m.View(), ":theme") {
		t.Error("footer should echo the command buffer")
	}
}

func TestSpawnSuccessSetsStatus(t *testing.T) {
	m := sized(t, newModel(t))
	var got []string
	m.spawnFn = func(args []string) error { got = args; return nil }
	// focus Sessions, then open (o) resumes the focused claude session.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}) // focus Sessions
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	m = nm.(Model)
	if len(got) == 0 || got[0] != "claude" {
		t.Errorf("o on a claude session should spawn claude, got %v", got)
	}
	if !strings.Contains(m.status, "launched") {
		t.Errorf("status = %q, want a launched note", m.status)
	}
}

func TestSpawnNoTerminalShowsManualModal(t *testing.T) {
	m := sized(t, newModel(t))
	m.spawnFn = func([]string) error { return spawn.ErrNoTerminal }
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	m = nm.(Model)
	if !m.showManual {
		t.Fatal("ErrNoTerminal should open the manual-launch modal")
	}
	if m.manualCmd != "claude" {
		t.Errorf("manualCmd = %q, want claude", m.manualCmd)
	}
	if !strings.Contains(m.View(), "new terminal") {
		t.Error("manual modal should tell the user to run it in a new terminal")
	}
	// any key closes it — and does NOT quit.
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if nm.(Model).showManual {
		t.Error("a key should close the manual modal")
	}
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("closing the manual modal must not quit the TUI")
		}
	}
}

func TestResumeCommand(t *testing.T) {
	cases := map[string][]string{
		"claude":   {"claude", "--resume"},
		"opencode": {"opencode"},
		"copilot":  {"copilot"},
	}
	for src, want := range cases {
		got := resumeCommand(src)
		if len(got) != len(want) || got[0] != want[0] {
			t.Errorf("resumeCommand(%q) = %v, want %v", src, got, want)
		}
	}
}

// gateStore has a pending pipeline gate that ApproveGate clears.
type gateStore struct {
	fakeStore
	pending *string
}

func (g gateStore) ApprovalPending() string { return *g.pending }
func (g gateStore) ApproveGate() error      { *g.pending = ""; return nil }

func TestPipelineApproveGate(t *testing.T) {
	pending := "IMPLEMENT"
	m, err := New("v-test", gateStore{fakeStore{n: 3}, &pending})
	if err != nil {
		t.Fatal(err)
	}
	m = sized(t, m)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	m = nm.(Model)
	if m.detailKind != "pipeline" {
		t.Fatal("P should open the pipeline overlay")
	}
	if !strings.Contains(m.View(), "awaiting approval") {
		t.Error("pipeline overlay should show the pending-gate banner")
	}
	// `a` approves the gate.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = nm.(Model)
	if pending != "" {
		t.Errorf("a should clear the gate, pending still %q", pending)
	}
	if !strings.Contains(m.status, "approved pipeline gate") {
		t.Errorf("status = %q, want approved note", m.status)
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
