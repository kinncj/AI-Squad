package dashboard

import (
	"os"
	"os/exec"
	"path/filepath"
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
func (f fakeStore) PRDetail(n int) []string { return []string{"PR #22", "rework", "OPEN"} }
func (f fakeStore) ApprovePR(int) error     { return nil }
func (f fakeStore) Tests() []state.Test {
	return []state.Test{{Path: "app/x_test.go", Framework: "go"}}
}
func (f fakeStore) RunTest(state.Test) []string           { return []string{"$ go test", "ok"} }
func (f fakeStore) SkillsSearch(string) []string          { return []string{"skill-a", "skill-b"} }
func (f fakeStore) SkillInstall(p string) []string        { return []string{"installed " + p} }
func (f fakeStore) SkillRemove(n string) []string         { return []string{"removed " + n} }
func (f fakeStore) ShipSafeAudit() []string               { return []string{"$ npx ship-safe audit .", "✓ passed"} }
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
func (f fakeStore) ApprovalPending() string             { return "" }
func (f fakeStore) ApproveGate() error                  { return nil }
func (f fakeStore) RejectGate() error                   { return nil }
func (f fakeStore) PortalURL() string                   { return "" }
func (f fakeStore) ProjectConfigExists() bool           { return true }
func (f fakeStore) ClaudeDirExists() bool               { return true }
func (f fakeStore) ProjectName() string                 { return "test-project" }
func (f fakeStore) TaffyCount() int                     { return 5 }
func (f fakeStore) PipelineStatus() string              { return "DONE" }
func (f fakeStore) Pipeline() state.Pipeline            { return state.Pipeline{Status: "DONE"} }
func (f fakeStore) MergeMapleJSON(map[string]any) error { return nil }
func (f fakeStore) ClearPipeline() error                { return nil }

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
	if !strings.Contains(grid, "[Tab] pane") {
		t.Error("grid view should show the footer keybindings")
	}
	// Footer is contextual: Stories is focused by default, so its actions show.
	if !strings.Contains(grid, "design review") {
		t.Error("footer should show Stories-pane context keys")
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

func TestHelpShowsPortalAndLANAddress(t *testing.T) {
	m, err := New("v-test", fakeStore{n: 3}, "http://localhost:7842/")
	if err != nil {
		t.Fatal(err)
	}
	// Force a known LAN ip so the test is deterministic.
	old := lanIP
	lanIP = func() string { return "192.168.1.50" }
	defer func() { lanIP = old }()

	m = sized(t, m)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	view := nm.(Model).View()
	if !strings.Contains(view, "http://localhost:7842/") {
		t.Error("help should show the design portal URL")
	}
	if !strings.Contains(view, "http://192.168.1.50:7842/") {
		t.Error("help should show the LAN-reachable portal URL")
	}
	if !strings.Contains(view, "192.168.1.50") {
		t.Error("help should show this machine's IP")
	}
}

func TestLANURLRewritesLoopback(t *testing.T) {
	cases := map[string]string{
		"http://localhost:7800/":  "http://10.0.0.2:7800/",
		"http://127.0.0.1:7800/x": "http://10.0.0.2:7800/x",
		"http://0.0.0.0:7800/":    "http://10.0.0.2:7800/",
	}
	for in, want := range cases {
		if got := lanURL(in, "10.0.0.2"); got != want {
			t.Errorf("lanURL(%q) = %q, want %q", in, got, want)
		}
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

func TestShipSafeRunsIntoOverlay(t *testing.T) {
	m := sized(t, newModel(t))
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	m = nm.(Model)
	if m.detailKind != "shipsafe" {
		t.Fatalf("S should open the ship-safe overlay, got detailKind %q", m.detailKind)
	}
	if cmd == nil {
		t.Fatal("S should run the audit")
	}
	sm, ok := cmd().(skillsMsg)
	if !ok {
		t.Fatalf("S should produce a skillsMsg, got a different message")
	}
	nm2, _ := m.Update(sm)
	if !strings.Contains(nm2.(Model).View(), "ship-safe") {
		t.Error("ship-safe output should render in the overlay")
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

func TestImplementOpensHarnessPicker(t *testing.T) {
	cwd, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(cwd)

	m := mustNew(t, storyPathStore{fakeStore{n: 1}})
	// two harnesses installed → i must open a picker, claude first (the default).
	m.lookPath = func(bin string) (string, error) {
		if bin == "claude" || bin == "copilot" {
			return "/usr/bin/" + bin, nil
		}
		return "", exec.ErrNotFound
	}
	var launched []string
	m.execFn = func(args []string) tea.Cmd { launched = args; return nil }
	m = sized(t, m)

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = nm.(Model)
	if m.picker == nil || m.pickerMode != "implement" {
		t.Fatal("i should open the implement harness picker")
	}
	if len(m.pickItems) != 2 || m.pickItems[0].id != "claude" {
		t.Fatalf("picker should list installed harnesses claude-first, got %+v", m.pickItems)
	}
	if launched != nil {
		t.Error("i must not launch before a harness is chosen")
	}
	// Enter on the default (claude) launches implement-stories for claude.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.picker != nil {
		t.Error("Enter should close the picker")
	}
	if harnessOf(launched) != "claude" {
		t.Errorf("should launch implement-stories with claude, got %v", launched)
	}
}

func TestImplementSingleHarnessSkipsPicker(t *testing.T) {
	cwd, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(cwd)

	m := mustNew(t, storyPathStore{fakeStore{n: 1}})
	m.lookPath = func(bin string) (string, error) {
		if bin == "opencode" {
			return "/usr/bin/opencode", nil
		}
		return "", exec.ErrNotFound
	}
	var launched []string
	m.execFn = func(args []string) tea.Cmd { launched = args; return nil }
	m = sized(t, m)

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = nm.(Model)
	if m.picker != nil {
		t.Error("a single harness should skip the picker")
	}
	if harnessOf(launched) != "opencode" {
		t.Errorf("should launch directly with opencode, got %v", launched)
	}
}

// harnessOf returns the harness binary from a launch argv, seeing past an env RTK wrapper.
func harnessOf(argv []string) string {
	for _, a := range argv {
		switch a {
		case "claude", "opencode", "copilot", "cursor", "cursor-agent":
			return a
		}
	}
	if len(argv) > 0 {
		return argv[0]
	}
	return ""
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
	var ran [][]string
	oldRunner := rtkInitRunner
	rtkInitRunner = func(flags []string) (string, error) { ran = append(ran, flags); return "", nil }
	defer func() { rtkInitRunner = oldRunner }()

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
	if !strings.Contains(m.View(), "○ Claude") {
		t.Errorf("rtk overlay should show claude as off:\n%s", m.View())
	}
	// Enter runs `rtk init …` async for the focused (first = claude) harness.
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("Enter should launch rtk init")
	}
	msg := cmd() // execute the async rtk init
	if len(ran) != 1 || strings.Join(ran[0], " ") != "init -g" {
		t.Errorf("should run `rtk init -g` for claude, got %v", ran)
	}
	nm, _ = m.Update(msg) // deliver rtkInitDoneMsg
	m = nm.(Model)
	if !on["claude"] {
		t.Error("rtk init completion should wire claude")
	}
	if m.picker == nil {
		t.Error("wiring should keep the picker open")
	}
	if !strings.Contains(m.View(), "✓ Claude") {
		t.Errorf("rtk overlay should now show claude wired:\n%s", m.View())
	}
	// esc closes.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).picker != nil {
		t.Error("esc should close the rtk overlay")
	}
}

func TestResumeCommandPassesSessionID(t *testing.T) {
	cases := []struct {
		se   state.Session
		want []string
	}{
		// claude records the JSONL path; resume needs the bare UUID
		{state.Session{Source: "claude", ID: "/Users/x/.claude/projects/p/abc-123.jsonl"}, []string{"claude", "--resume", "abc-123"}},
		{state.Session{Source: "opencode", ID: "ses_9"}, []string{"opencode", "--session", "ses_9"}},
		{state.Session{Source: "copilot", ID: "07b8"}, []string{"copilot", "--resume=07b8"}},
		// no id → fall back to a bare resume/launch
		{state.Session{Source: "claude", ID: ""}, []string{"claude", "--resume"}},
		{state.Session{Source: "opencode", ID: ""}, []string{"opencode"}},
		{state.Session{Source: "pi", ID: "x"}, []string{"pi"}},
	}
	for _, c := range cases {
		got := resumeCommand(c.se)
		if strings.Join(got, " ") != strings.Join(c.want, " ") {
			t.Errorf("resumeCommand(%+v) = %v, want %v", c.se, got, c.want)
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
	// The URL is a clickable OSC-8 hyperlink (safe at runtime; the earlier "auto-open"
	// was a test executing `open`, since fixed via the openFn seam).
	if !strings.Contains(view, "\x1b]8;;http://127.0.0.1:7811\x1b\\") {
		t.Error("portal URL should be a clickable OSC-8 hyperlink")
	}
}

type storyPathStore struct{ fakeStore }

func (s storyPathStore) Stories() []state.Story {
	return []state.Story{{ID: "auth-reset-0001", Path: "docs/stories/auth-reset-0001/Story.md"}}
}

func TestGateClearNudgesHarness(t *testing.T) {
	// Swap the nudge so we can observe it without touching a real multiplexer.
	orig := notifyContinue
	nudged := 0
	notifyContinue = func(func(string) string) int { nudged++; return 1 }
	defer func() { notifyContinue = orig }()

	pending := "wireframe"
	m := sized(t, mustNew(t, gateStore{fakeStore{n: 1}, &pending}))
	// First tick observes the pending gate (no nudge yet).
	nm, _ := m.Update(tickMsg{})
	m = nm.(Model)
	if nudged != 0 {
		t.Fatalf("no nudge while the gate is still pending, got %d", nudged)
	}
	// The gate clears (e.g. approved in the portal); the next tick nudges once.
	pending = ""
	nm, _ = m.Update(tickMsg{})
	m = nm.(Model)
	if nudged != 1 {
		t.Errorf("gate-clear should nudge exactly once, got %d", nudged)
	}
	if !strings.Contains(m.status, "nudged") {
		t.Errorf("status should note the nudge, got %q", m.status)
	}
	// A further tick with no gate must not nudge again.
	nm, _ = m.Update(tickMsg{})
	if nudged != 1 {
		t.Errorf("no repeat nudge once the gate stays clear, got %d", nudged)
	}
}

func TestImplementFocusedStoryLaunches(t *testing.T) {
	cwd, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	defer os.Chdir(cwd)

	var launched []string
	m := mustNew(t, storyPathStore{fakeStore{n: 1}})
	m.execFn = func(args []string) tea.Cmd { launched = args; return nil }
	m.lookPath = func(bin string) (string, error) {
		if bin == "claude" {
			return "/usr/bin/claude", nil
		}
		return "", exec.ErrNotFound
	}
	m = sized(t, m)
	// Stories pane is focused by default; [i] implements the selected story.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = nm.(Model)
	joined := strings.Join(launched, " ")
	if !strings.Contains(joined, "claude") || !strings.Contains(joined, "implement-stories") {
		t.Errorf("[i] should launch claude with implement-stories, got %v", launched)
	}
	// The gherkin handoff should have been written for the story dir.
	if _, err := os.Stat(filepath.Join(tmp, ".claude/state/gherkin-handoff.json")); err != nil {
		t.Error("implement should write the gherkin handoff")
	}
}

func TestImplementNoStoryNotes(t *testing.T) {
	// Sessions pane focused, no story context → a helpful status, no launch.
	var launched []string
	m := mustNew(t, fakeStore{n: 0})
	m.execFn = func(args []string) tea.Cmd { launched = args; return nil }
	m = sized(t, m)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}) // focus Sessions
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if launched != nil {
		t.Error("i with no story should not launch anything")
	}
	if !strings.Contains(nm.(Model).status, "story") {
		t.Errorf("expected a story hint, got %q", nm.(Model).status)
	}
}

func TestClickingHeaderURLOpensPortal(t *testing.T) {
	var opened []string
	m := mustNew(t, portalStore{fakeStore{n: 2}, "http://127.0.0.1:7800"})
	m.openFn = func(args []string) error { opened = args; return nil }
	// wide viewport so the URL is shown in the header
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)

	_, _, start, end := m.headerParts()
	if end <= start {
		t.Fatal("portal URL should occupy a header column range")
	}
	// Click in the middle of the URL on the header row.
	col := (start + end) / 2
	nm, cmd := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: col, Y: 0})
	if cmd == nil {
		t.Fatal("clicking the URL should return an open command")
	}
	cmd() // runs through the injected openFn (no real browser)
	if len(opened) == 0 || !strings.Contains(strings.Join(opened, " "), "7800") {
		t.Errorf("click should open the portal URL, got %v", opened)
	}

	// A click elsewhere in the header must NOT open it.
	opened = nil
	_, cmd2 := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 1, Y: 0})
	if cmd2 != nil {
		cmd2()
	}
	if opened != nil {
		t.Errorf("clicking the leaf/brand area must not open the portal, got %v", opened)
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
	// request-changes KEEPS the gate open (until approval) — it must not clear it, and it
	// must not say "continue" to the harness.
	if pending == "" {
		t.Error("r (request changes) should keep the gate open, not clear it")
	}
	if !strings.HasPrefix(m.status, "changes requested") {
		t.Errorf("status = %q, want a changes-requested note", m.status)
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
