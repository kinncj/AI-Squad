package dashboard

import (
	"os/exec"
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
func (f fakeStore) DesignTree() []string                  { return []string{"📁 wireframes", "  📄 home.md"} }
func (f fakeStore) LogLines(n int) []string               { return []string{"ts=12:00  agent=qa"} }
func (f fakeStore) GitChanges() []string                  { return []string{"── status ──", " M app/x.go"} }
func (f fakeStore) PipelineLines() []string               { return []string{"status  RUNNING", "stage  IMPLEMENT"} }
func (f fakeStore) Skills() []string                      { return []string{"gh-issues", "humanizer"} }
func (f fakeStore) Agents() []string                      { return []string{"orchestrator"} }
func (f fakeStore) RTKHarnesses() map[string]bool         { return map[string]bool{} }
func (f fakeStore) SetRTKHarness(string, bool) error      { return nil }
func (f fakeStore) PinnedSessions() map[string]string     { return map[string]string{} }
func (f fakeStore) SetPinnedSession(string, string) error { return nil }
func (f fakeStore) DesignArtifacts(id string) []state.Artifact {
	return []state.Artifact{{Path: "docs/design/wireframes/" + id + ".wireframe.md", Kind: "wireframes", Status: "pending"}}
}
func (f fakeStore) ApprovalPending() string   { return "" }
func (f fakeStore) ApproveGate() error        { return nil }
func (f fakeStore) RejectGate() error         { return nil }
func (f fakeStore) PortalURL() string         { return "" }
func (f fakeStore) ProjectConfigExists() bool { return true }
func (f fakeStore) ClaudeDirExists() bool     { return true }
func (f fakeStore) ProjectName() string       { return "test-project" }
func (f fakeStore) TaffyCount() int           { return 5 }
func (f fakeStore) PipelineStatus() string    { return "DONE" }

func newModel(t *testing.T) Model {
	t.Helper()
	m, err := New("v-test", fakeStore{n: 12}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

// sized dismisses the splash + boot screen and gives the model a viewport.
func sized(t *testing.T, m Model) Model {
	t.Helper()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}) // dismiss splash
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})            // dismiss boot check
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

func TestExitActionsQuitWithFollowUp(t *testing.T) {
	// key → expected follow-up action after the dashboard quits.
	cases := map[string]ExitAction{
		"n": ExitReq,    // new story → req flow
		"u": ExitUpdate, // update → init --force
	}
	for key, want := range cases {
		m := sized(t, newModel(t))
		nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		assertQuit(t, cmd, "exit-action "+key)
		if got := nm.(Model).Action(); got != want {
			t.Errorf("key %q: Action() = %d, want %d", key, got, want)
		}
	}
}

func TestCommandModeCoversActions(t *testing.T) {
	// Overlay-opening commands (no cmd returned, mutate the model).
	overlay := map[string]func(Model) bool{
		"pipeline": func(m Model) bool { return m.detail != nil && m.detailKind == "pipeline" },
		"changes":  func(m Model) bool { return m.detail != nil },
		"skills":   func(m Model) bool { return m.detail != nil },
		"review":   func(m Model) bool { return m.reviewStory != "" },
		"help":     func(m Model) bool { return m.showHelp },
		"rtk":      func(m Model) bool { return m.picker != nil },
	}
	for cmd, ok := range overlay {
		m := sized(t, newModel(t))
		nm, _ := m.runCommand(cmd)
		if !ok(nm.(Model)) {
			t.Errorf(":%s did not open its overlay", cmd)
		}
	}
}

func TestCommandTabCompletes(t *testing.T) {
	m := sized(t, newModel(t))
	m.commanding, m.cmdBuf = true, "pi"
	nm, _ := m.handleCommandKey(tea.KeyMsg{Type: tea.KeyTab})
	if got := nm.(Model).cmdBuf; got != "pipeline" {
		t.Errorf("Tab on :pi should complete to :pipeline, got :%s", got)
	}
	// Ambiguous prefix completes to the common prefix ("re" → review/reload share "re").
	m.cmdBuf = "re"
	nm, _ = m.handleCommandKey(tea.KeyMsg{Type: tea.KeyTab})
	if got := nm.(Model).cmdBuf; !strings.HasPrefix("reload", got) && !strings.HasPrefix("review", got) {
		t.Errorf("Tab on :re should complete to a shared prefix, got :%s", got)
	}
}

func TestCommandModeExitActions(t *testing.T) {
	cases := map[string]ExitAction{
		"req":     ExitReq,
		"update":  ExitUpdate,
		"labels":  ExitLabels,
		"project": ExitProject,
	}
	for cmdName, want := range cases {
		m := sized(t, newModel(t))
		nm, cmd := m.runCommand(cmdName)
		assertQuit(t, cmd, ":"+cmdName)
		if got := nm.(Model).Action(); got != want {
			t.Errorf(":%s → Action() = %d, want %d", cmdName, got, want)
		}
	}
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

// liveStore lets a test mutate the pipeline status + story count between ticks.
type liveStore struct {
	fakeStore
	status *string
	nStory *int
}

func (l liveStore) PipelineStatus() string { return *l.status }
func (l liveStore) Stories() []state.Story {
	out := make([]state.Story, *l.nStory)
	for i := range out {
		out[i] = state.Story{ID: "s" + itoa(i+1)}
	}
	return out
}

func TestTickRefreshesLiveStateAndKeepsSelection(t *testing.T) {
	status, n := "RUNNING", 5
	m := sized(t, mustNew(t, liveStore{fakeStore{n: 5}, &status, &n}))
	_ = m.View()
	// Move selection down twice on the Stories pane.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	if m.group.Panes()[paneStories].Selected() != 2 {
		t.Fatalf("precondition: selection should be 2, got %d", m.group.Panes()[paneStories].Selected())
	}
	// The pipeline status changes underneath us (a harness advanced).
	status = "DONE"
	n = 8
	nm, cmd := m.Update(tickMsg{})
	m = nm.(Model)
	// Tick reschedules itself.
	if cmd == nil {
		t.Error("tick should reschedule itself")
	}
	// Live status is reflected in the header without a keypress.
	if !strings.Contains(m.View(), "Gherkin: 8") {
		t.Error("tick should refresh the story count shown in the header")
	}
	// Selection is preserved across the refresh.
	if got := m.group.Panes()[paneStories].Selected(); got != 2 {
		t.Errorf("tick must preserve selection, got %d want 2", got)
	}
}

func TestEscClosesPipelineAndTickDoesNotReopen(t *testing.T) {
	m := sized(t, newModel(t))
	// Shift+P opens the pipeline (taffy) overlay.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	m = nm.(Model)
	if m.detail == nil || m.detailKind != "pipeline" {
		t.Fatal("P should open the pipeline overlay")
	}
	// Esc closes it.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.detail != nil {
		t.Fatal("esc should close the pipeline overlay")
	}
	// A refresh tick must NOT re-open it (the bug: it kept going back to taffy).
	nm, _ = m.Update(tickMsg{})
	m = nm.(Model)
	if m.detail != nil {
		t.Error("a tick after esc must not re-open the pipeline overlay")
	}
	if m.detailKind != "" {
		t.Errorf("detailKind should be cleared after esc, got %q", m.detailKind)
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
	// Dismiss splash → boot check → dashboard shell.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	boot := nm.(Model).View()
	if !strings.Contains(boot, "boot check") {
		t.Error("after the splash, the boot-check screen should show")
	}
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
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

func TestBootScreenBetweenSplashAndDashboard(t *testing.T) {
	m := newModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = nm.(Model)
	// Dismiss splash → boot check.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = nm.(Model)
	if m.booted {
		t.Fatal("after splash the model should be on the boot check, not booted")
	}
	if !strings.Contains(m.View(), "boot check") || !strings.Contains(m.View(), "✓") {
		t.Error("boot view should show the readiness checklist")
	}
	// Enter continues.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !nm.(Model).booted {
		t.Error("Enter should pass the boot check")
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
	// Fresh model per key so focus starts on Stories (p focuses PRs there; on the
	// Sessions pane p pins instead — tested separately).
	for key, want := range map[string]int{"a": paneSessions, "p": panePRs, "Q": paneQA, "s": paneStories} {
		m := sized(t, newModel(t))
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if got := nm.(Model).group.FocusIndex(); got != want {
			t.Errorf("key %q focused pane %d, want %d", key, got, want)
		}
	}
}

// pinStore holds mutable session pins for the pin test.
type pinStore struct {
	fakeStore
	pins map[string]string
}

func (p pinStore) Sessions() []state.Session {
	return []state.Session{{ID: "sess-1", Title: "work", Source: "claude", ToolCount: 5}}
}
func (p pinStore) PinnedSessions() map[string]string { return p.pins }
func (p pinStore) SetPinnedSession(source, id string) error {
	if id == "" {
		delete(p.pins, source)
	} else {
		p.pins[source] = id
	}
	return nil
}

func TestSessionPinToggle(t *testing.T) {
	m, err := New("v-test", pinStore{fakeStore{n: 3}, map[string]string{}}, "")
	if err != nil {
		t.Fatal(err)
	}
	m = sized(t, m)
	// Focus Sessions, then p pins the selected session.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = nm.(Model)
	if !strings.Contains(m.status, "pinned claude") {
		t.Errorf("p on a session should pin it, status=%q", m.status)
	}
	if !strings.Contains(m.View(), "●") {
		t.Error("pinned session should show the ● marker")
	}
	// p again unpins.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	if !strings.Contains(nm.(Model).status, "unpinned") {
		t.Errorf("p again should unpin, status=%q", nm.(Model).status)
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
	m, err := New("v-test", detailStore{}, "")
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

func TestOpenSessionLaunchesInTerminal(t *testing.T) {
	m := sized(t, newModel(t))
	var got []string
	m.execFn = func(args []string) tea.Cmd { got = args; return nil }
	// focus Sessions, then open (o) resumes the focused claude session in-terminal.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}) // focus Sessions
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	if len(got) == 0 || got[0] != "claude" {
		t.Errorf("o on a claude session should launch claude, got %v", got)
	}
}

func TestShipSafeLaunchesInTerminal(t *testing.T) {
	m := sized(t, newModel(t))
	var got []string
	m.execFn = func(args []string) tea.Cmd { got = args; return nil }
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	if nm.(Model).showHelp {
		t.Fatal("S should not open help")
	}
	if len(got) < 2 || got[0] != "npx" || got[1] != "ship-safe" {
		t.Errorf("S should run npx ship-safe, got %v", got)
	}
}

func TestExecDoneResumesWithoutQuitting(t *testing.T) {
	m := sized(t, newModel(t))
	// A finished harness resume must clear the screen and NOT quit.
	nm, cmd := m.Update(execDoneMsg{args: []string{"claude"}})
	if cmd == nil {
		t.Fatal("execDoneMsg should return a ClearScreen command")
	}
	if _, ok := cmd().(tea.QuitMsg); ok {
		t.Error("resuming from a harness must never quit the dashboard")
	}
	if !strings.Contains(nm.(Model).status, "back in maple") {
		t.Errorf("status = %q, want a resume note", nm.(Model).status)
	}
}

func TestLauncherPickerAndSpawn(t *testing.T) {
	m := sized(t, newModel(t))
	// Only claude + copilot are "installed".
	m.lookPath = func(bin string) (string, error) {
		if bin == "claude" || bin == "copilot" {
			return "/usr/bin/" + bin, nil
		}
		return "", exec.ErrNotFound
	}
	var launched []string
	m.execFn = func(args []string) tea.Cmd { launched = args; return nil }

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	m = nm.(Model)
	if m.picker == nil {
		t.Fatal("L should open the launcher picker")
	}
	if len(m.pickItems) != 2 {
		t.Errorf("picker should list 2 installed harnesses, got %d", len(m.pickItems))
	}
	if !strings.Contains(m.View(), "claude") {
		t.Error("picker should show harness options")
	}
	// Move to the second (copilot) and launch it.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.picker != nil {
		t.Error("Enter should close the picker")
	}
	if len(launched) != 1 || launched[0] != "copilot" {
		t.Errorf("should have launched copilot, got %v", launched)
	}
}

func TestQuickPromptPicksSkillOrAgent(t *testing.T) {
	m := sized(t, newModel(t))
	m.lookPath = func(bin string) (string, error) {
		if bin == "claude" {
			return "/usr/bin/claude", nil
		}
		return "", exec.ErrNotFound
	}
	var launched []string
	m.execFn = func(args []string) tea.Cmd { launched = args; return nil }

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = nm.(Model)
	if m.picker == nil {
		t.Fatal("x should open the quick-prompt picker")
	}
	// fakeStore: 2 skills + 1 agent = 3 items.
	if len(m.pickItems) != 3 {
		t.Errorf("picker should list skills + agents (3), got %d", len(m.pickItems))
	}
	if !strings.Contains(m.View(), "/gh-issues") || !strings.Contains(m.View(), "@orchestrator") {
		t.Error("picker should show /skill and @agent entries")
	}
	// Launch the first skill.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if len(launched) != 2 || launched[0] != "claude" || launched[1] != "/gh-issues" {
		t.Errorf("should launch claude with /gh-issues, got %v", launched)
	}
}

func TestLauncherNoHarness(t *testing.T) {
	m := sized(t, newModel(t))
	m.lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	m = nm.(Model)
	if m.picker != nil {
		t.Error("no harness should not open a picker")
	}
	if !strings.Contains(m.status, "no harness") {
		t.Errorf("status = %q, want no-harness note", m.status)
	}
}

// rtkStore holds a mutable rtk-harness map for the toggle test.
type rtkStore struct {
	fakeStore
	on map[string]bool
}

func (r rtkStore) RTKHarnesses() map[string]bool { return r.on }
func (r rtkStore) SetRTKHarness(name string, v bool) error {
	r.on[name] = v
	return nil
}

func TestRTKToggleOverlay(t *testing.T) {
	on := map[string]bool{}
	m, err := New("v-test", rtkStore{fakeStore{n: 3}, on}, "")
	if err != nil {
		t.Fatal(err)
	}
	m = sized(t, m)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	m = nm.(Model)
	if m.picker == nil || m.pickerMode != "rtk" {
		t.Fatal("R should open the rtk toggle overlay")
	}
	if !strings.Contains(m.View(), "○ claude") {
		t.Errorf("rtk overlay should show claude as off:\n%s", m.View())
	}
	// Enter toggles the focused (first = claude) harness on, keeping the picker open.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if !on["claude"] {
		t.Error("Enter should wire claude")
	}
	if m.picker == nil {
		t.Error("toggle should keep the picker open")
	}
	if !strings.Contains(m.View(), "✓ claude") {
		t.Errorf("rtk overlay should now show claude wired:\n%s", m.View())
	}
	// esc closes.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).picker != nil {
		t.Error("esc should close the rtk overlay")
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
type portalStore struct {
	fakeStore
	url string
}

func (p portalStore) PortalURL() string { return p.url }

func TestHeaderShowsPortalURL(t *testing.T) {
	url := "http://127.0.0.1:7811"
	m := mustNew(t, portalStore{fakeStore{n: 2}, url})
	// Wide viewport so the header isn't truncated before the portal suffix.
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}) // splash
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})                     // boot
	view := nm.(Model).View()
	if !strings.Contains(view, "7811") {
		t.Error("header should show the design portal URL/port")
	}
	// Plain text (terminal auto-detects the URL) — NOT an OSC-8 hyperlink, which caused
	// accidental plain-click opens.
	if strings.Contains(view, "\x1b]8;;") {
		t.Error("portal URL must be plain text, not an OSC-8 hyperlink")
	}
}

func TestPortalKeyOpensBrowser(t *testing.T) {
	// Inject the opener so the test NEVER launches a real browser (it used to open
	// http://127.0.0.1:78xx on every `go test` run).
	var opened []string
	m := mustNew(t, portalStore{fakeStore{n: 2}, "http://127.0.0.1:7811"})
	m.openFn = func(args []string) error { opened = args; return nil }
	m = sized(t, m)
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if cmd == nil {
		t.Fatal("v with a portal URL should return an open command")
	}
	if _, ok := cmd().(tea.QuitMsg); ok {
		t.Error("opening the portal must not quit the dashboard")
	}
	if len(opened) == 0 || !strings.Contains(strings.Join(opened, " "), "7811") {
		t.Errorf("v should open the portal URL, got %v", opened)
	}
	// Without a URL, v just notes it (and returns no open command).
	m2 := sized(t, newModel(t)) // fakeStore.PortalURL == ""
	nm2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if !strings.Contains(nm2.(Model).status, "no design portal") {
		t.Errorf("v without a portal should note it, got %q", nm2.(Model).status)
	}
	_ = nm
}

func mustNew(t *testing.T, store Store) Model {
	t.Helper()
	m, err := New("v-test", store, "")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type gateStore struct {
	fakeStore
	pending *string
}

func (g gateStore) ApprovalPending() string { return *g.pending }
func (g gateStore) ApproveGate() error      { *g.pending = ""; return nil }
func (g gateStore) RejectGate() error       { *g.pending = ""; return nil }

func TestPipelineApproveGate(t *testing.T) {
	pending := "IMPLEMENT"
	m, err := New("v-test", gateStore{fakeStore{n: 3}, &pending}, "")
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
	if !strings.HasPrefix(m.status, "approved") {
		t.Errorf("status = %q, want approved note", m.status)
	}
}

func TestPipelineRejectGate(t *testing.T) {
	pending := "DESIGN"
	m, err := New("v-test", gateStore{fakeStore{n: 3}, &pending}, "")
	if err != nil {
		t.Fatal(err)
	}
	m = sized(t, m)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	m = nm.(Model)
	if !strings.Contains(m.View(), "reject") {
		t.Error("pipeline banner should offer [r] reject")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = nm.(Model)
	if pending != "" {
		t.Errorf("r should reject/clear the gate, pending=%q", pending)
	}
	if !strings.HasPrefix(m.status, "rejected") {
		t.Errorf("status = %q, want rejected note", m.status)
	}
}

func TestRateLimitBadge(t *testing.T) {
	m := sized(t, New2(t, rlStore{}))
	if !strings.Contains(m.View(), "RATE-LIMITED") {
		t.Error("header should show a RATE-LIMITED badge when the pipeline is rate-limited")
	}
}

// rlStore reports a rate-limited pipeline.
type rlStore struct{ fakeStore }

func (rlStore) PipelineStatus() string { return "RATE_LIMITED" }

// New2 builds a sized model from a store.
func New2(t *testing.T, s Store) Model {
	t.Helper()
	m, err := New("v-test", s, "")
	if err != nil {
		t.Fatal(err)
	}
	return sized(t, m)
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
