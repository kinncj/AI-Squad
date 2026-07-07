// Package dashboard is the maple TUI's top-level Bubble Tea model. It composes the
// pane primitive, theme, splash, and (in later sub-projects) the state adapters.
// This is the walking skeleton: it proves the architecture end-to-end with
// placeholder sources; real state sources replace demoSource in a later step.
package dashboard

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kinncj/maple/app/internal/harness"
	"github.com/kinncj/maple/app/internal/state"
	"github.com/kinncj/maple/app/internal/tui/brand"
	"github.com/kinncj/maple/app/internal/tui/pane"
	"github.com/kinncj/maple/app/internal/tui/render"
	reqtui "github.com/kinncj/maple/app/internal/tui/req"
	"github.com/kinncj/maple/app/internal/tui/splash"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

// Store provides live project state to the dashboard. state.FS satisfies it.
type Store interface {
	Stories() []state.Story
	Sessions() []state.Session
	PullRequests() []state.PullRequest
	Tests() []state.Test
	DesignTree() []string
	LogLines(n int) []string
	GitChanges() []string
	PipelineLines() []string
	Skills() []string
	Agents() []string
	RTKHarnesses() map[string]bool
	SetRTKHarness(name string, on bool) error
	PinnedSessions() map[string]string
	SetPinnedSession(source, id string) error
	DesignArtifacts(storyID string) []state.Artifact
	ApprovalPending() string
	ApproveGate() error
	RejectGate() error
	PortalURL() string
	ProjectConfigExists() bool
	ClaudeDirExists() bool
	ProjectName() string
	TaffyCount() int
	PipelineStatus() string
}

// linesSource is a scroll-only (non-selectable) pane source backing the fullscreen
// Design and Logs views.
type linesSource struct{ rows []string }

func (l linesSource) Rows() []string { return l.rows }

// fullscreen targets.
const (
	fsNone = iota
	fsDesign
	fsLogs
)

// Model is the top-level dashboard model.
type Model struct {
	theme       *theme.Theme
	mode        theme.Mode
	group       *pane.Group
	stories     *storySource
	sessions    *sessionSource
	prs         *prSource
	qa          *qaSource
	design      *pane.Pane
	logs        *pane.Pane
	detail      *pane.Pane
	fullscreen  int
	store       Store
	execFn      func([]string) tea.Cmd
	openFn      func([]string) error // browser/URL opener; injected in tests so they never open a real browser
	lookPath    func(string) (string, error)
	picker      *pane.Pane
	pickItems   []pickItem
	pickerMode  string
	width       int
	height      int
	splash      bool
	booted      bool
	showHelp    bool
	filtering   bool
	filterBuf   string
	commanding  bool
	cmdBuf      string
	reviewStory string
	detailKind  string
	storyPath   string // Story.md path of the open story detail (for `i` implement)
	lastGate    string // last-seen pending approval stage, to detect gate-clear → nudge
	status      string
	version     string
	portalURL   string
	exitAction  ExitAction
}

// ExitAction is a follow-up workflow the dashboard requests when it quits. The outer
// runDashboardLoop runs it in-process (it needs the terminal) and re-enters the
// dashboard afterward. Harness launches do NOT use this — they run in the current
// terminal via tea.ExecProcess (suspend/resume) and the dashboard never quits.
type ExitAction int

const (
	ExitNone    ExitAction = iota
	ExitQuit               // plain quit — no follow-up
	ExitReq                // run `maple req`
	ExitUpdate             // run `maple update` (init --force)
	ExitLabels             // run `maple labels`
	ExitProject            // run `maple project`
)

// Action reports the follow-up workflow requested when the dashboard quit.
func (m Model) Action() ExitAction { return m.exitAction }

// New builds the dashboard model from a project store. portalURL is the design-review
// portal URL the caller started this session (may be ""); it takes precedence over the
// store's file-based URL so the header always matches the running server. Pass
// state.NewFS(".") in production or a fake in tests. Fails only on a malformed theme.
func New(version string, store Store, portalURL string) (Model, error) {
	th, err := theme.Load()
	if err != nil {
		return Model{}, err
	}
	stories := newStorySource(store.Stories())
	sessions := newSessionSource(store.Sessions(), store.PinnedSessions())
	prs := newPRSource(store.PullRequests())
	qa := newQASource(store.Tests())
	g := pane.NewGroup(
		pane.New("Stories", stories),
		pane.New("Sessions", sessions),
		pane.New("Pull Requests", prs),
		pane.New("QA / Tests", qa),
	)
	design := pane.New("Design", linesSource{store.DesignTree()})
	logs := pane.New("Logs", linesSource{store.LogLines(500)})
	design.SetFocus(true)
	logs.SetFocus(true)
	return Model{
		theme:    th,
		mode:     th.ActiveMode(),
		group:    g,
		stories:  stories,
		sessions: sessions,
		prs:      prs,
		qa:       qa,
		design:   design,
		logs:     logs,
		store:     store,
		splash:    true,
		version:   version,
		portalURL: portalURL,
	}, nil
}

// portal returns the effective design-portal URL: the session URL passed to New if
// set, otherwise the store's file-based URL.
func (m Model) portal() string {
	if m.portalURL != "" {
		return m.portalURL
	}
	return m.store.PortalURL()
}

// lanIP returns the machine's primary non-loopback IPv4 (so a teammate or an agent on
// another device can reach the portal), or "" if none. Swappable in tests.
var lanIP = func() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
}

// lanURL rewrites a loopback portal URL to the LAN ip so another device can open it.
func lanURL(url, ip string) string {
	return strings.NewReplacer("localhost", ip, "127.0.0.1", ip, "0.0.0.0", ip).Replace(url)
}

// activePane returns the overlay pane taking navigation and rendering: a detail
// overlay if open, else a fullscreen Design/Logs pane, else nil (the grid is live).
func (m Model) activePane() *pane.Pane {
	if m.detail != nil {
		return m.detail
	}
	return m.fullscreenPane()
}

// fullscreenPane returns the active fullscreen pane, or nil when none is open.
func (m Model) fullscreenPane() *pane.Pane {
	switch m.fullscreen {
	case fsDesign:
		return m.design
	case fsLogs:
		return m.logs
	}
	return nil
}

// openDetail opens a detail overlay for the focused pane's selected row.
func (m *Model) openDetail() {
	p := m.group.Focused()
	if p == nil {
		return
	}
	sel := p.Selected()
	if sel < 0 {
		return
	}
	switch m.group.FocusIndex() {
	case paneStories:
		if st, ok := m.stories.at(sel); ok {
			label := st.Title
			if label == "" {
				label = st.ID
			}
			m.setDetail("Story · "+label, colorizeStory(state.FileLines(st.Path), m.mode))
			m.detailKind = "story"
			m.storyPath = st.Path
		}
	case paneSessions:
		if se, ok := m.sessions.at(sel); ok {
			m.setDetail("Session · "+se.Title, state.SessionDetail(se))
		}
	case paneQA:
		if tst, ok := m.qa.at(sel); ok {
			m.setDetail("Test · "+tst.Path, state.FileLines(tst.Path))
		}
	}
}

// pickItem is one choice in the picker overlay. id identifies the item for toggle
// pickers; argv is the spawn command for spawn pickers.
type pickItem struct {
	label string
	argv  []string
	id    string
}

// pickerSource is a selectable list source backing the picker overlay.
type pickerSource struct{ rows []string }

func (p pickerSource) Rows() []string { return p.rows }
func (p pickerSource) RowCount() int  { return len(p.rows) }

// available reports whether a binary is on PATH (injectable for tests).
func (m *Model) available(bin string) bool {
	look := m.lookPath
	if look == nil {
		look = exec.LookPath
	}
	_, err := look(bin)
	return err == nil
}

// openPicker shows a selectable list; Enter spawns the chosen item.
func (m *Model) openPicker(title string, items []pickItem) {
	m.pickerMode = "spawn"
	m.setPicker(title, items, 0)
}

// setPicker (re)builds the picker pane from items, restoring the selection.
func (m *Model) setPicker(title string, items []pickItem, sel int) {
	m.pickItems = items
	rows := make([]string, len(items))
	for i, it := range items {
		rows[i] = it.label
	}
	m.picker = pane.New(title, pickerSource{rows})
	m.picker.SetFocus(true)
	if sel > 0 {
		m.picker.SelectBy(sel, len(rows))
	}
}

// closePicker dismisses the picker overlay.
func (m *Model) closePicker() {
	m.picker, m.pickItems, m.pickerMode = nil, nil, ""
}

// rtkHarnesses are the harnesses rtk can wire, in display order.
var rtkHarnesses = []string{"claude", "opencode", "copilot", "cursor"}

// openRTK shows the rtk per-harness toggle overlay.
func (m *Model) openRTK() {
	m.pickerMode = "rtk"
	m.rebuildRTK(0)
}

// rebuildRTK renders the rtk toggle list with ✓/○ marks from current state.
func (m *Model) rebuildRTK(sel int) {
	on := m.store.RTKHarnesses()
	items := make([]pickItem, len(rtkHarnesses))
	for i, h := range rtkHarnesses {
		mark := "○"
		if on[h] {
			mark = "✓"
		}
		items[i] = pickItem{label: mark + " " + h, id: h}
	}
	m.setPicker("RTK harnesses — [Enter] toggle · [esc] close", items, sel)
}

// toggleRTK flips the wired state of the item at i and refreshes the list.
func (m *Model) toggleRTK(i int) {
	h := m.pickItems[i].id
	on := m.store.RTKHarnesses()
	if err := m.store.SetRTKHarness(h, !on[h]); err != nil {
		m.status = "rtk: " + err.Error()
		return
	}
	m.rebuildRTK(i)
}

// firstHarness returns the first installed harness, or "".
func (m *Model) firstHarness() string {
	for _, h := range []string{"claude", "opencode", "copilot"} {
		if m.available(h) {
			return h
		}
	}
	return ""
}

// openQuickPrompt offers the project's skills (/name) and agents (@name); picking one
// launches the harness with it as the opening prompt.
func (m *Model) openQuickPrompt() {
	h := m.firstHarness()
	if h == "" {
		m.status = "no harness found (claude / opencode / copilot)"
		return
	}
	var items []pickItem
	for _, sk := range m.store.Skills() {
		items = append(items, pickItem{label: "/" + sk, argv: []string{h, "/" + sk}})
	}
	for _, ag := range m.store.Agents() {
		items = append(items, pickItem{label: "@" + ag, argv: []string{h, "@" + ag}})
	}
	if len(items) == 0 {
		m.status = "no skills or agents found"
		return
	}
	m.openPicker("Quick prompt — pick a skill or agent", items)
}

// openLauncher offers the installed harnesses to launch.
func (m *Model) openLauncher() {
	harnesses := []pickItem{
		{label: "claude", argv: []string{"claude"}},
		{label: "opencode", argv: []string{"opencode"}},
		{label: "copilot", argv: []string{"copilot"}},
	}
	var avail []pickItem
	for _, h := range harnesses {
		if m.available(h.argv[0]) {
			avail = append(avail, h)
		}
	}
	if len(avail) == 0 {
		m.status = "no harness found (claude / opencode / copilot)"
		return
	}
	m.openPicker("Launch a harness", avail)
}

// execDoneMsg is delivered when a launched process exits (harness resume/launch) or
// an external open (browser) returns. external opens don't suspend the TUI.
type execDoneMsg struct {
	args     []string
	err      error
	external bool
}

// realExec runs args in the CURRENT terminal: it suspends the dashboard, hands the
// terminal to the process, and resumes when it exits. This is how harnesses launch —
// in-place, never a new terminal window, and maple never truly quits.
func realExec(args []string) tea.Cmd {
	if len(args) == 0 {
		return nil
	}
	c := exec.Command(args[0], args[1:]...)
	c.Env = os.Environ()
	if rtk, err := exec.LookPath("rtk"); err == nil && rtk != "" {
		c.Env = append(c.Env, "RTK_HOOK_AUDIT=1")
	}
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execDoneMsg{args: args, err: err}
	})
}

// launch starts a harness. Inside a multiplexer (tmux/zellij) it opens a split
// pane/tab so the TUI stays live and the pane stays addressable for approval nudges;
// otherwise it runs in the current terminal (suspend/resume). execFn, when set (tests),
// short-circuits to the in-terminal path so tests never touch a real multiplexer.
func (m *Model) launch(args []string) tea.Cmd {
	if m.execFn != nil {
		return m.execFn(args)
	}
	name := harness.Key(args)
	ref, inPane, err := harness.LaunchInPane(os.Getenv, name, args)
	if inPane {
		if err != nil {
			m.status = "split failed (" + ref.Kind + "): " + err.Error()
		} else {
			m.status = "opened " + name + " in a " + ref.Kind + " split pane · [P]→[a] to approve gates"
		}
		return nil
	}
	// No tmux/zellij/wezterm/kitty detected — run in this terminal (suspends maple).
	m.status = "no multiplexer — running " + name + " in this terminal; use tmux for a side pane"
	return realExec(args)
}

// continueNote describes how the just-approved harness will proceed, given how many
// live panes accepted a "continue" nudge.
func continueNote(nudged int) string {
	if nudged > 0 {
		return fmt.Sprintf("nudged %d harness pane(s) to continue", nudged)
	}
	return "harness continues on its next poll (≤2s)"
}

// openExternal fires a detached command (e.g. opening a URL in the browser) without
// suspending the TUI.
func (m *Model) openExternal(args []string) tea.Cmd {
	open := m.openFn
	if open == nil {
		open = func(a []string) error { return exec.Command(a[0], a[1:]...).Start() }
	}
	return func() tea.Msg {
		if len(args) == 0 {
			return execDoneMsg{external: true}
		}
		err := open(args)
		return execDoneMsg{args: args, err: err, external: true}
	}
}

// openSession resumes the focused session in the current terminal, or opens the
// focused PR in the browser.
func (m *Model) openSession() tea.Cmd {
	switch m.group.FocusIndex() {
	case paneSessions:
		if se, ok := m.sessions.at(m.group.Focused().Selected()); ok {
			return m.launch(resumeCommand(se))
		}
	case panePRs:
		if pr, ok := m.prs.at(m.group.Focused().Selected()); ok {
			return m.openExternal([]string{"gh", "pr", "view", "--web", fmt.Sprintf("%d", pr.Number)})
		}
	}
	return nil
}

// pinFocusedSession toggles the pin on the focused session (per its harness source)
// and refreshes the pane so the ● marker updates.
func (m *Model) pinFocusedSession() {
	se, ok := m.sessions.at(m.group.Focused().Selected())
	if !ok {
		return
	}
	id := se.ID
	if m.store.PinnedSessions()[se.Source] == se.ID {
		id = "" // already pinned → unpin
	}
	if err := m.store.SetPinnedSession(se.Source, id); err != nil {
		m.status = "pin: " + err.Error()
		return
	}
	sel := m.group.FocusIndex()
	m.reload()
	m.group.SetFocus(sel)
	if id == "" {
		m.status = "unpinned " + se.Source + " session"
	} else {
		m.status = "pinned " + se.Source + " session"
	}
}

// browserOpen returns the OS command that opens a URL in the default browser.
func browserOpen(url string) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"open", url}
	case "windows":
		return []string{"cmd", "/c", "start", url}
	default:
		return []string{"xdg-open", url}
	}
}

// resumeCommand maps a session to its resume argv, passing the session id so the SELECTED
// session actually resumes (not a fresh harness). Claude records the JSONL path as the id,
// so its bare UUID is the base filename.
func resumeCommand(se state.Session) []string {
	switch se.Source {
	case "claude":
		uuid := strings.TrimSuffix(filepath.Base(se.ID), ".jsonl")
		if uuid == "" || uuid == "." {
			return []string{"claude", "--resume"}
		}
		return []string{"claude", "--resume", uuid}
	case "opencode":
		if se.ID != "" {
			return []string{"opencode", "--session", se.ID}
		}
		return []string{"opencode"}
	case "copilot":
		if se.ID != "" {
			return []string{"copilot", "--resume=" + se.ID}
		}
		return []string{"copilot"}
	default:
		return []string{se.Source}
	}
}

// setDetail opens a scrollable detail overlay with the given title and content.
func (m *Model) setDetail(title string, lines []string) {
	m.detail = pane.New(title, linesSource{lines})
	m.detail.SetFocus(true)
	m.detailKind = ""
}

// openPipeline shows the pipeline overlay, with an approval banner when a human gate
// is pending. Marks kind="pipeline" so `a` approves.
func (m *Model) openPipeline() {
	lines := m.store.PipelineLines()
	if stage := m.store.ApprovalPending(); stage != "" {
		lines = append([]string{
			"⚠ awaiting approval: " + stage,
			"[a] approve · [r] reject · [v] portal",
			"",
		}, lines...)
	}
	m.setDetail("Pipeline", lines)
	m.detailKind = "pipeline"
}

// openReview opens the design review overlay for a story and marks review mode so
// `a` approves.
func (m *Model) openReview(storyID string) {
	m.reviewStory = storyID
	m.setDetail("Design Review · "+storyID, formatArtifacts(m.store.DesignArtifacts(storyID)))
}

// approveReview approves every not-yet-approved artifact for the review story and
// refreshes the overlay. Returns how many it approved.
func (m *Model) approveReview() int {
	n := 0
	for _, a := range m.store.DesignArtifacts(m.reviewStory) {
		if a.Status != "approved" {
			if state.ApproveArtifact(a.Path) == nil {
				n++
			}
		}
	}
	m.openReview(m.reviewStory) // refresh statuses
	return n
}

// formatArtifacts renders design artifacts for the review overlay.
func formatArtifacts(arts []state.Artifact) []string {
	if len(arts) == 0 {
		return []string{"(no design artifacts for this story)", "", "run the design phase first"}
	}
	out := []string{"[a] approve all pending · esc close", ""}
	for _, a := range arts {
		mark := "•"
		if a.Status == "approved" {
			mark = "✓"
		}
		out = append(out, fmt.Sprintf("%s [%s] %s — %s", mark, a.Kind, filepath.Base(a.Path), a.Status))
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// splashDoneMsg auto-dismisses the splash after a short delay so it never blocks.
type splashDoneMsg struct{}

// tickMsg drives the local file-state refresh; netTickMsg drives the slower network
// (gh PR) refresh; prsLoadedMsg carries the async PR result back to the model.
type tickMsg struct{}
type netTickMsg struct{}
type prsLoadedMsg struct{ prs []state.PullRequest }

// Refresh cadences: local file reads are cheap and drive real-time approval/pipeline
// feedback; the network refresh (gh) is far slower so it runs infrequently and async.
const (
	tickInterval    = 2 * time.Second
	netTickInterval = 60 * time.Second
)

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(time.Time) tea.Msg { return tickMsg{} })
}
func netTickCmd() tea.Cmd {
	return tea.Tick(netTickInterval, func(time.Time) tea.Msg { return netTickMsg{} })
}

// loadPRsCmd fetches pull requests off the render path (gh is slow / network-bound).
func (m Model) loadPRsCmd() tea.Cmd {
	store := m.store
	return func() tea.Msg { return prsLoadedMsg{prs: store.PullRequests()} }
}

// Init implements tea.Model. It starts the splash timer and the refresh ticks, and
// kicks off the first async PR load.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(1400*time.Millisecond, func(time.Time) tea.Msg { return splashDoneMsg{} }),
		tickCmd(),
		netTickCmd(),
		m.loadPRsCmd(),
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case splashDoneMsg:
		if m.splash {
			m.splash = false
			return m, tea.ClearScreen
		}
	case tickMsg:
		// Real-time local refresh: pipeline status, approvals, stories, sessions,
		// tests, design, logs — so the dashboard reflects file/process changes without
		// a keypress. The header reads status live each render; this refreshes panes.
		if !m.splash {
			m.reload()
		}
		return m, tickCmd()
	case netTickMsg:
		return m, tea.Batch(m.loadPRsCmd(), netTickCmd())
	case prsLoadedMsg:
		m.prs = newPRSource(msg.prs)
		m.group.Panes()[panePRs].SetSource(m.prs)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case execDoneMsg:
		switch {
		case msg.err != nil:
			m.status = "launch failed: " + msg.err.Error()
		case msg.external:
			m.status = "opened " + strings.Join(msg.args, " ")
		default:
			m.reload() // a resumed harness may have changed state
			m.status = "back in maple"
		}
		if !msg.external {
			return m, tea.ClearScreen // the process took over the screen
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	// Splash: any key dismisses (quit still quits). ClearScreen wipes the image.
	if m.splash {
		if k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}
		m.splash = false
		return m, tea.ClearScreen
	}

	// Boot-check screen: Enter/any key continues to the dashboard; q quits.
	if !m.booted {
		if k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}
		m.booted = true
		return m, tea.ClearScreen
	}

	// Help overlay: any key closes it.
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// Picker overlay: navigate + Enter (launch the choice, or toggle in rtk mode).
	if m.picker != nil {
		switch k {
		case "esc":
			m.closePicker()
		case "up", "k":
			m.picker.SelectBy(-1, m.picker.VisibleRows())
		case "down", "j":
			m.picker.SelectBy(1, m.picker.VisibleRows())
		case "enter":
			i := m.picker.Selected()
			if i >= 0 && i < len(m.pickItems) {
				if m.pickerMode == "rtk" {
					m.toggleRTK(i) // toggle keeps the picker open
				} else {
					item := m.pickItems[i]
					m.closePicker()
					return m, m.launch(item.argv)
				}
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	// Filter input mode captures typing until enter/esc.
	if m.filtering {
		return m.handleFilterKey(msg), nil
	}

	// Command mode (vim-style `:`) captures typing until enter/esc.
	if m.commanding {
		return m.handleCommandKey(msg)
	}

	m.status = "" // any key clears a transient status line

	// Fullscreen Design/Logs: d/l toggle; while any overlay is open, nav scrolls it.
	switch k {
	case "d":
		m.detail = nil
		m.fullscreen = toggleFS(m.fullscreen, fsDesign)
		return m, nil
	case "l":
		m.detail = nil
		m.fullscreen = toggleFS(m.fullscreen, fsLogs)
		return m, nil
	}
	if p := m.activePane(); p != nil {
		switch k {
		case "esc":
			m.detail = nil
			m.detailKind = ""
			m.fullscreen = fsNone
			m.reviewStory = ""
		case "a":
			switch {
			case m.reviewStory != "":
				n := m.approveReview()
				m.status = fmt.Sprintf("approved %d artifact(s) · %s", n, continueNote(harness.NotifyContinue(os.Getenv)))
			case m.detailKind == "pipeline":
				if m.store.ApprovalPending() == "" {
					m.status = "no gate awaiting approval"
				} else if err := m.store.ApproveGate(); err == nil {
					m.openPipeline()
					m.status = "approved · " + continueNote(harness.NotifyContinue(os.Getenv))
				} else {
					m.status = "approve failed: " + err.Error()
				}
			}
		case "r":
			if m.detailKind == "pipeline" {
				if m.store.ApprovalPending() == "" {
					m.status = "no gate awaiting approval"
				} else if err := m.store.RejectGate(); err == nil {
					m.openPipeline()
					m.status = "rejected · " + continueNote(harness.NotifyContinue(os.Getenv))
				} else {
					m.status = "reject failed: " + err.Error()
				}
			}
		case "v":
			if m.detailKind == "pipeline" {
				if url := m.portal(); url != "" {
					return m, m.openExternal(browserOpen(url))
				}
				m.status = "no design portal running"
			}
		case "up", "k":
			p.ScrollBy(-1, p.VisibleRows())
		case "down", "j":
			p.ScrollBy(1, p.VisibleRows())
		case "g", "home":
			p.Top()
		case "G", "end":
			p.Bottom(p.VisibleRows())
		case "?":
			m.showHelp = true
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.openDetail()
	case "C":
		m.setDetail("Git Changes", m.store.GitChanges())
	case "D":
		if m.group.FocusIndex() == paneStories {
			if st, ok := m.stories.at(m.group.Focused().Selected()); ok {
				m.openReview(st.ID)
			}
		}
	case "P":
		m.openPipeline()
	case "F":
		m.setDetail("Skills", m.store.Skills())
	case "o":
		return m, m.openSession()
	case "L":
		m.openLauncher()
	case "x":
		m.openQuickPrompt()
	case "n":
		m.exitAction = ExitReq
		return m, tea.Quit
	case "i":
		return m.implementFocusedStory()
	case "u":
		m.exitAction = ExitUpdate
		return m, tea.Quit
	case "R":
		m.openRTK()
	case "S":
		return m, m.launch([]string{"npx", "ship-safe", "audit", "."})
	case "v":
		if url := m.portal(); url != "" {
			return m, m.openExternal(browserOpen(url))
		}
		m.status = "no design portal running"
	case "?":
		m.showHelp = true
	case "/":
		m.filtering, m.filterBuf = true, m.group.Focused().Filter()
	case ":":
		m.commanding, m.cmdBuf = true, ""
	case "tab":
		m.group.FocusNext()
	case "shift+tab":
		m.group.FocusPrev()
	case "up", "k":
		m.group.SelectBy(-1)
	case "down", "j":
		m.group.SelectBy(1)
	case "g", "home":
		m.group.Top()
	case "G", "end":
		m.group.Bottom()
	case "s":
		m.group.SetFocus(paneStories)
	case "a":
		m.group.SetFocus(paneSessions)
	case "p":
		// On the Sessions pane, p pins the selected session; elsewhere it focuses PRs.
		if m.group.FocusIndex() == paneSessions {
			m.pinFocusedSession()
		} else {
			m.group.SetFocus(panePRs)
		}
	case "Q":
		m.group.SetFocus(paneQA)
	case "r":
		m.reload()
		m.status = "reloaded"
		return m, m.loadPRsCmd()
	default:
		// Keys whose overlays are not yet ported in the rebuild. Registered so the
		// surface is discoverable; each lands as its overlay is built (sub-project 5).
		if name, ok := pendingOverlays[k]; ok {
			m.status = name + " — not yet ported in the rebuild (press ? for the full map)"
		}
	}
	return m, nil
}

// pane indices in the group, matching the New() order.
const (
	paneStories = iota
	paneSessions
	panePRs
	paneQA
)

// toggleFS returns fsNone if target is already active, else target.
func toggleFS(cur, target int) int {
	if cur == target {
		return fsNone
	}
	return target
}

// pendingOverlays maps keys to the overlays still to be ported from tui/.
// pendingOverlays maps keys to features not yet ported (empty — all wired).
var pendingOverlays = map[string]string{}

// handleFilterKey processes typing while the filter input is active.
func (m Model) handleFilterKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "esc":
		m.filtering, m.filterBuf = false, ""
		m.group.SetFilter("")
	case "enter":
		m.filtering = false
		m.group.SetFilter(m.filterBuf)
	case "backspace":
		if len(m.filterBuf) > 0 {
			r := []rune(m.filterBuf)
			m.filterBuf = string(r[:len(r)-1])
			m.group.SetFilter(m.filterBuf)
		}
	default:
		if len(msg.Runes) > 0 {
			m.filterBuf += string(msg.Runes)
			m.group.SetFilter(m.filterBuf)
		}
	}
	return m
}

// handleCommandKey processes typing while `:` command mode is active.
func (m Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.commanding, m.cmdBuf = false, ""
	case "enter":
		m.commanding = false
		return m.runCommand(strings.TrimSpace(m.cmdBuf))
	case "tab":
		// Complete to the shortest common prefix of the matches, or the sole match.
		if matches := commandMatches(m.cmdBuf); len(matches) == 1 {
			m.cmdBuf = matches[0]
		} else if len(matches) > 1 {
			m.cmdBuf = longestCommonPrefix(matches)
		}
	case "backspace":
		if len(m.cmdBuf) > 0 {
			r := []rune(m.cmdBuf)
			m.cmdBuf = string(r[:len(r)-1])
		}
	default:
		if len(msg.Runes) > 0 {
			m.cmdBuf += string(msg.Runes)
		}
	}
	return m, nil
}

// commandNames are the accepted `:` command words, shown by `:help`/completion. Every
// keybinding action has a spelled-out command so the whole surface is typeable.
var commandNames = []string{
	"quit", "reload", "help", "pipeline", "changes", "design", "logs", "review",
	"skills", "launch", "resume", "prompt", "rtk", "ship", "portal", "filter",
	"theme", "req", "update", "labels", "project",
}

// runCommand dispatches a `:` command line. Accepts the full action vocabulary plus
// short aliases, so anything reachable by a key is also reachable by name.
func (m Model) runCommand(line string) (tea.Model, tea.Cmd) {
	m.cmdBuf = ""
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return m, nil
	}
	switch fields[0] {
	case "q", "q!", "wq", "quit":
		return m, tea.Quit
	case "e", "e!", "r", "reload":
		m.reload()
		m.status = "reloaded"
		return m, m.loadPRsCmd()
	case "help", "h", "?", "commands":
		m.showHelp = true
	case "P", "pipeline", "taffy":
		m.openPipeline()
	case "C", "changes", "diff", "git":
		m.setDetail("Git Changes", m.store.GitChanges())
	case "D", "review", "design-review":
		if id := m.targetStoryID(); id != "" {
			m.openReview(id)
		} else {
			m.status = "no story to review"
		}
	case "d", "design":
		m.detail, m.detailKind = nil, ""
		m.fullscreen = toggleFS(m.fullscreen, fsDesign)
	case "l", "logs":
		m.detail, m.detailKind = nil, ""
		m.fullscreen = toggleFS(m.fullscreen, fsLogs)
	case "F", "skills":
		m.setDetail("Skills", m.store.Skills())
	case "L", "launch", "launcher":
		m.openLauncher()
	case "o", "open", "resume":
		return m, m.openSession()
	case "x", "prompt", "quick":
		m.openQuickPrompt()
	case "R", "rtk":
		m.openRTK()
	case "S", "ship", "ship-safe":
		return m, m.launch([]string{"npx", "ship-safe", "audit", "."})
	case "v", "portal":
		if url := m.portal(); url != "" {
			return m, m.openExternal(browserOpen(url))
		}
		m.status = "no design portal running"
	case "theme", "colo":
		if len(fields) < 2 {
			m.status = "usage: :theme <" + strings.Join(theme.Names(), "|") + ">"
			break
		}
		if th, err := theme.Switch(fields[1]); err == nil {
			m.theme, m.mode = th, th.ActiveMode()
			m.status = "theme: " + th.Name
		} else {
			m.status = "unknown theme: " + fields[1]
		}
	case "n", "req", "story", "new":
		m.exitAction = ExitReq
		return m, tea.Quit
	case "u", "update":
		m.exitAction = ExitUpdate
		return m, tea.Quit
	case "labels":
		m.exitAction = ExitLabels
		return m, tea.Quit
	case "project":
		m.exitAction = ExitProject
		return m, tea.Quit
	default:
		m.status = "unknown command :" + fields[0] + " — try :help"
	}
	return m, nil
}

// implementFocusedStory launches /pipeline-runner implement-stories for the open or
// selected story in a split pane, so the harness runs beside maple. Mirrors the `i`
// (Implement via TAFFY) action in `maple req`.
func (m Model) implementFocusedStory() (tea.Model, tea.Cmd) {
	path := m.targetStoryPath()
	if path == "" {
		m.status = "focus or open a story first, then [i] to implement"
		return m, nil
	}
	h := m.firstHarness()
	if h == "" {
		m.status = "no harness found to implement (claude / opencode / copilot)"
		return m, nil
	}
	m.status = "launching implement-stories for " + filepath.Base(filepath.Dir(path))
	return m, m.launch(reqtui.ImplementArgs(h, []string{filepath.Dir(path)}))
}

// targetStoryPath returns the Story.md path to act on: the open story detail, else the
// Stories-pane selection, else "".
func (m Model) targetStoryPath() string {
	if m.detailKind == "story" && m.storyPath != "" {
		return m.storyPath
	}
	if m.group.FocusIndex() == paneStories {
		if st, ok := m.stories.at(m.group.Focused().Selected()); ok {
			return st.Path
		}
	}
	return ""
}

// targetStoryID returns the story to act on for name-driven commands: the selection on
// the Stories pane if that's focused, otherwise the first story.
func (m Model) targetStoryID() string {
	if m.group.FocusIndex() == paneStories {
		if st, ok := m.stories.at(m.group.Focused().Selected()); ok {
			return st.ID
		}
	}
	if st, ok := m.stories.at(0); ok {
		return st.ID
	}
	return ""
}

// reload re-reads live project state into the panes.
// reload refreshes the local (file-based) project state in place, preserving each
// pane's selection and scroll. It never touches the network — PRs refresh separately
// on the slower net tick. When the pipeline overlay is open it re-reads it so stage
// changes and approvals show live.
func (m *Model) reload() {
	if m.store == nil {
		return
	}
	m.stories = newStorySource(m.store.Stories())
	m.sessions = newSessionSource(m.store.Sessions(), m.store.PinnedSessions())
	m.qa = newQASource(m.store.Tests())
	panes := m.group.Panes()
	panes[paneStories].SetSource(m.stories)
	panes[paneSessions].SetSource(m.sessions)
	panes[paneQA].SetSource(m.qa)
	m.design.SetSource(linesSource{m.store.DesignTree()})
	m.logs.SetSource(linesSource{m.store.LogLines(500)})
	// Refresh the pipeline overlay live only while it's actually open — otherwise a
	// tick would re-open it after the user pressed esc to leave.
	if m.detail != nil && m.detailKind == "pipeline" {
		m.openPipeline()
	}
	// When a gate that was pending has just cleared — whether the user approved in the
	// TUI or in the design portal — nudge the harness to continue. This unifies the
	// nudge on maple's full logic (recorded panes + tmux sibling-broadcast + all
	// terminals), so a portal approval reliably advances a yielded harness too.
	gate := m.store.ApprovalPending()
	if m.lastGate != "" && gate == "" {
		if n := notifyContinue(os.Getenv); n > 0 {
			m.status = fmt.Sprintf("gate cleared — nudged %d harness pane(s)", n)
		}
	}
	m.lastGate = gate
}

// notifyContinue nudges harness panes to continue; a package var so tests can observe
// gate-clear behaviour without touching a real multiplexer.
var notifyContinue = harness.NotifyContinue

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.splash || msg.Action != tea.MouseActionPress {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.group.ScrollAt(msg.X, msg.Y, -1)
	case tea.MouseButtonWheelDown:
		m.group.ScrollAt(msg.X, msg.Y, 1)
	case tea.MouseButtonLeft:
		// A click on the portal URL in the header opens it — a plain click works even
		// though mouse capture stops the terminal handling the OSC-8 link itself.
		if msg.Y == 0 {
			if _, _, start, end := m.headerParts(); end > start && msg.X >= start && msg.X < end {
				if url := m.portal(); url != "" {
					return m, m.openExternal(browserOpen(url))
				}
			}
		}
		m.group.FocusAt(msg.X, msg.Y)
	}
	return m, nil
}

// kittyClearImages deletes all placed Kitty graphics. Prepended once we leave the
// splash so an inline-image splash doesn't linger over the dashboard. Harmless on
// terminals without Kitty graphics.
const kittyClearImages = "\x1b_Ga=d\x1b\\"

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.splash {
		return splash.Render(m.mode, m.width, m.height, "maple "+m.version)
	}
	if !m.booted {
		return m.bootView()
	}
	header := m.header()
	bodyH := m.height - lipgloss.Height(header) - 1
	if bodyH < 2 {
		bodyH = 2
	}
	body := m.grid(bodyH)
	if m.showHelp {
		body = m.helpView(bodyH)
	} else if m.picker != nil {
		body = m.picker.RenderAt(0, lipgloss.Height(header), m.width, bodyH, m.mode)
	} else if p := m.activePane(); p != nil {
		body = p.RenderAt(0, lipgloss.Height(header), m.width, bodyH, m.mode)
	}
	return kittyClearImages + lipgloss.JoinVertical(lipgloss.Left, header, body, m.footer())
}

// bootChecks are the readiness checks shown on the boot screen.
type bootCheck struct {
	label string
	ok    bool
}

func (m Model) bootChecks() []bootCheck {
	return []bootCheck{
		{"project.config.yaml present", m.store.ProjectConfigExists()},
		{".claude/ directory present", m.store.ClaudeDirExists()},
		{"a harness is installed", m.firstHarness() != ""},
	}
}

// bootView renders the boot-check screen: a ✓/✗ readiness list, centered.
func (m Model) bootView() string {
	ok := m.mode.State("done").Style()
	bad := m.mode.State("blocked").Style()
	title := m.mode.Role("title").Style()
	faint := m.mode.Role("faint").Style()

	allOK := true
	lines := []string{title.Render(brand.Leaf + " maple — boot check"), ""}
	for _, c := range m.bootChecks() {
		if c.ok {
			lines = append(lines, ok.Render("✓ ")+c.label)
		} else {
			lines = append(lines, bad.Render("✗ ")+c.label)
			allOK = false
		}
	}
	hint := "Enter to continue · q to quit"
	if !allOK {
		hint = "some checks failed — Enter to continue anyway · q to quit"
	}
	lines = append(lines, "", faint.Render(hint))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.mode.Role("border_focus").FG)).
		Padding(1, 3).
		Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// helpView renders the keybinding reference as a bordered, column-aligned box,
// centered in bodyH rows.
func (m Model) helpView(bodyH int) string {
	const keyW = 16
	key := m.mode.Role("accent").Style()
	desc := m.mode.Role("base").Style()
	faint := m.mode.Role("faint").Style()

	// row pads the key to a fixed display width first, then styles both columns so
	// the description column aligns regardless of unicode in the key.
	row := func(k, d string, dim bool) string {
		ds := desc
		if dim {
			ds = faint
		}
		return key.Render(render.PadRight(k, keyW)) + "  " + ds.Render(d)
	}

	nav := [][2]string{
		{"Tab / Shift+Tab", "cycle panes"},
		{"j / k · ↓ / ↑", "navigate rows"},
		{"g / G", "top / bottom"},
		{"s a p Q", "focus Stories / Sessions / PRs / QA"},
		{"Enter", "open detail (story / session / test)"},
		{"/", "filter the focused pane"},
		{"esc", "close overlay"},
	}
	actions := [][2]string{
		{"P", "pipeline / taffy (a approve · r reject · v portal)"},
		{"D", "design review (a approves artifacts)"},
		{"C", "git changes (status + diff)"},
		{"F", "skills · d / l Design / Logs full-screen"},
		{"o", "open/resume session · PR in browser"},
		{"L", "launch a harness (tmux/zellij pane or in-term)"},
		{"x", "quick prompt (skill / agent)"},
		{"n / u", "new story (req) / update template"},
		{"R / S", "rtk wiring / ship-safe audit"},
		{"v", "open design portal in browser"},
		{"r", "reload now"},
		{": ", "command mode — every action by name (:help)"},
		{"? · q", "toggle help / quit"},
	}

	var lines []string
	lines = append(lines, m.mode.Role("title").Style().Render("Keybindings"), "")
	lines = append(lines, m.mode.Role("subtitle").Style().Render("Navigation"))
	for _, r := range nav {
		lines = append(lines, row(r[0], r[1], false))
	}
	lines = append(lines, "", m.mode.Role("subtitle").Style().Render("Actions"))
	for _, r := range actions {
		lines = append(lines, row(r[0], r[1], false))
	}

	// Connect: the design-portal URL and, when the machine has a LAN address, the same
	// portal reachable from another device — handy for reviewing on a phone/tablet or
	// pointing a remote agent at it.
	if url := m.portal(); url != "" {
		lines = append(lines, "", m.mode.Role("subtitle").Style().Render("Connect"))
		lines = append(lines, row("Design portal", url, false))
		if ip := lanIP(); ip != "" {
			if lu := lanURL(url, ip); lu != url {
				lines = append(lines, row("On this network", lu, false))
			}
			lines = append(lines, row("This machine", ip, false))
		}
	}

	lines = append(lines, "", faint.Render("command mode accepts: :"+strings.Join(commandNames[:8], " :")+" …"))
	lines = append(lines, faint.Render("press any key to close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.mode.Role("border_focus").FG)).
		Padding(1, 3).
		Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, box)
}

// header is the compact top bar: brand + project/theme on the left, Gherkin/Taffy
// badges on the right, in a single row.
func (m Model) header() string {
	left, right, _, _ := m.headerParts()
	return bar(left, right, m.width)
}

// headerParts builds the header's left and right segments and, when the portal URL is
// shown, the [start,end) screen columns it occupies — so a mouse click on it can open
// the portal (a plain click, since mouse capture stops the terminal handling OSC-8).
func (m Model) headerParts() (left, right string, urlStart, urlEnd int) {
	leaf := m.mode.Role("leaf").Style()
	faint := m.mode.Role("faint").Style()
	accent := m.mode.Role("accent").Style()

	running := 0
	if state.InFlight(m.store.PipelineStatus()) {
		running = 1
	}
	name := m.store.ProjectName()
	if name == "" {
		name = "—"
	}
	base := leaf.Render(brand.Leaf+" maple") +
		faint.Render(fmt.Sprintf(" %s · project: %s · theme: %s", m.version, name, m.theme.Name))
	if strings.EqualFold(m.store.PipelineStatus(), "RATE_LIMITED") {
		base += "  " + m.mode.State("rate_limited").Render("RATE-LIMITED")
	}
	right = accent.Render(fmt.Sprintf("📋 Gherkin: %d · ▶ Taffy: %d (%d running)",
		len(m.store.Stories()), m.store.TaffyCount(), running))
	left = base
	// Portal URL as a clickable OSC-8 hyperlink, but only when it fits without
	// truncation (render.Truncate isn't escape-aware and would corrupt the sequence).
	if url := m.portal(); url != "" {
		seg := "  ⬡ " + url
		if lipgloss.Width(base)+lipgloss.Width(seg)+lipgloss.Width(right)+1 <= m.width {
			urlStart = lipgloss.Width(base) + lipgloss.Width("  ⬡ ")
			urlEnd = urlStart + lipgloss.Width(url)
			left = base + osc8(url, accent.Render(seg))
		}
	}
	return left, right, urlStart, urlEnd
}

// osc8 wraps visible text in an OSC-8 terminal hyperlink so ⌘/ctrl-click opens url.
// Terminals without OSC-8 support ignore the escapes and just show the text.
func osc8(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// footer is the bottom bar: the filter/command input when active (with live command
// suggestions), a transient status when set, otherwise context-aware key hints for the
// current overlay/pane plus a live focus summary.
func (m Model) footer() string {
	accent := m.mode.Role("accent").Style()
	faint := m.mode.Role("faint").Style()
	switch {
	case m.filtering:
		return bar(accent.Render("/"+m.filterBuf+"▏"),
			faint.Render("filter · Enter apply · esc cancel"), m.width)
	case m.commanding:
		return bar(accent.Render(":"+m.cmdBuf+"▏"),
			faint.Render(m.commandHint()), m.width)
	case m.status != "":
		return bar(m.mode.Role("subtitle").Style().Render(m.status), faint.Render("[?] help"), m.width)
	default:
		return bar(faint.Render(m.contextKeys()), faint.Render(m.contextSummary()), m.width)
	}
}

// contextKeys returns the key hints relevant to whatever is focused: the open overlay
// first, else the focused pane's actions, always with the global keys.
func (m Model) contextKeys() string {
	const global = "[Tab] pane · [:] cmd · [?] help · [q] quit"
	switch {
	case m.detailKind == "pipeline":
		return "PIPELINE  [a] approve · [r] reject · [v] portal · [esc] close"
	case m.reviewStory != "":
		return "DESIGN REVIEW  [a] approve artifacts · [esc] close"
	case m.detailKind == "story":
		return "STORY  [i] implement (taffy) · [j/k] scroll · [esc] close"
	case m.detail != nil:
		return "[j/k] scroll · [g/G] top/bottom · [esc] close"
	case m.fullscreen == fsDesign:
		return "DESIGN  [j/k] scroll · [d/esc] close · " + global
	case m.fullscreen == fsLogs:
		return "LOGS  [j/k] scroll · [l/esc] close · " + global
	}
	var ctx string
	switch m.group.FocusIndex() {
	case paneStories:
		ctx = "[Enter] story · [i] implement · [D] design review · [n] new"
	case paneSessions:
		ctx = "[Enter] transcript · [o] resume · [p] pin · [L] launch"
	case panePRs:
		ctx = "[Enter]/[o] open PR · [r] refresh"
	case paneQA:
		ctx = "[Enter] open test · [r] rescan"
	}
	return ctx + " · " + global
}

// contextSummary is the right-aligned live status: the focused pane and its position,
// or the pipeline status when a run is active.
func (m Model) contextSummary() string {
	if st := m.store.PipelineStatus(); state.InFlight(st) {
		if stage := m.store.ApprovalPending(); stage != "" {
			return "⏸ awaiting: " + stage
		}
	}
	p := m.group.Focused()
	if p == nil {
		return ""
	}
	total := m.focusedRowCount()
	if total == 0 {
		return p.Title + " · empty"
	}
	return fmt.Sprintf("%s · %d/%d", p.Title, p.Selected()+1, total)
}

// focusedRowCount returns the row count of the focused pane's source.
func (m Model) focusedRowCount() int {
	switch m.group.FocusIndex() {
	case paneStories:
		return m.stories.RowCount()
	case paneSessions:
		return m.sessions.RowCount()
	case panePRs:
		return m.prs.RowCount()
	case paneQA:
		return m.qa.RowCount()
	}
	return 0
}

// commandHint lists the commands whose names start with what's typed (all of them
// before any input), for live discovery + Tab-completion in `:` mode.
func (m Model) commandHint() string {
	matches := commandMatches(m.cmdBuf)
	if len(matches) == 0 {
		return "no match · esc cancel"
	}
	prefix := "Tab completes · "
	if m.cmdBuf == "" {
		prefix = ":"
	}
	return prefix + strings.Join(matches, " :")
}

// commandMatches returns the command names starting with prefix (all when empty).
func commandMatches(prefix string) []string {
	var out []string
	for _, c := range commandNames {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

// longestCommonPrefix returns the longest shared leading substring of ss.
func longestCommonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	p := ss[0]
	for _, s := range ss[1:] {
		for !strings.HasPrefix(s, p) {
			p = p[:len(p)-1]
			if p == "" {
				return ""
			}
		}
	}
	return p
}

// bar places left and right segments on a single full-width row, filling the gap
// with spaces and truncating if the segments don't fit.
func bar(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return render.Truncate(left+" "+right, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

// grid lays the four panes out in a 2×2 grid filling bodyH rows below the header.
func (m Model) grid(bodyH int) string {
	panes := m.group.Panes()
	if len(panes) == 0 {
		return ""
	}
	colW := m.width / 2
	rowH := bodyH / 2
	cells := make([]string, len(panes))
	for i, p := range panes {
		col := i % 2
		row := i / 2
		w := colW
		if col == 1 {
			w = m.width - colW // last column absorbs the rounding remainder
		}
		h := rowH
		if row == 1 {
			h = bodyH - rowH
		}
		cells[i] = p.RenderAt(col*colW, 1+row*rowH, w, h, m.mode)
	}
	top := lipgloss.JoinHorizontal(lipgloss.Top, cells[0], cells[1])
	if len(cells) < 4 {
		return top
	}
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, cells[2], cells[3])
	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}
