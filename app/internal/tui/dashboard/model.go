// Package dashboard is the maple TUI's top-level Bubble Tea model. It composes the
// pane primitive, theme, splash, and (in later sub-projects) the state adapters.
// This is the walking skeleton: it proves the architecture end-to-end with
// placeholder sources; real state sources replace demoSource in a later step.
package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kinncj/maple/app/internal/state"
	"github.com/kinncj/maple/app/internal/tui/brand"
	"github.com/kinncj/maple/app/internal/tui/pane"
	"github.com/kinncj/maple/app/internal/tui/render"
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
	theme      *theme.Theme
	mode       theme.Mode
	group      *pane.Group
	stories    *storySource
	sessions   *sessionSource
	qa         *qaSource
	design     *pane.Pane
	logs       *pane.Pane
	detail     *pane.Pane
	fullscreen int
	store      Store
	width      int
	height     int
	splash     bool
	showHelp   bool
	filtering  bool
	filterBuf  string
	commanding bool
	cmdBuf     string
	status     string
	version    string
}

// New builds the dashboard model from a project store. Pass state.NewFS(".") in
// production or a fake in tests. It fails only if the embedded theme is malformed.
func New(version string, store Store) (Model, error) {
	th, err := theme.Load()
	if err != nil {
		return Model{}, err
	}
	stories := newStorySource(store.Stories())
	sessions := newSessionSource(store.Sessions())
	qa := newQASource(store.Tests())
	g := pane.NewGroup(
		pane.New("Stories", stories),
		pane.New("Sessions", sessions),
		pane.New("Pull Requests", newPRSource(store.PullRequests())),
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
		qa:       qa,
		design:   design,
		logs:     logs,
		store:    store,
		splash:   true,
		version:  version,
	}, nil
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
			m.setDetail("Story · "+st.ID, state.FileLines(st.Path))
		}
	case paneSessions:
		if se, ok := m.sessions.at(sel); ok {
			m.setDetail("Session · "+se.Title, state.SessionTranscript(se.ID))
		}
	case paneQA:
		if tst, ok := m.qa.at(sel); ok {
			m.setDetail("Test · "+tst.Path, state.FileLines(tst.Path))
		}
	}
}

// setDetail opens a scrollable detail overlay with the given title and content.
func (m *Model) setDetail(title string, lines []string) {
	m.detail = pane.New(title, linesSource{lines})
	m.detail.SetFocus(true)
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

// Init implements tea.Model. It starts the splash auto-dismiss timer.
func (m Model) Init() tea.Cmd {
	return tea.Tick(1400*time.Millisecond, func(time.Time) tea.Msg { return splashDoneMsg{} })
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
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		m.handleMouse(msg)
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

	// Help overlay: any key closes it.
	if m.showHelp {
		m.showHelp = false
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
			m.fullscreen = fsNone
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
	case "P":
		m.setDetail("Pipeline", m.store.PipelineLines())
	case "F":
		m.setDetail("Skills", m.store.Skills())
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
		m.group.SetFocus(panePRs)
	case "Q":
		m.group.SetFocus(paneQA)
	case "r":
		m.reload()
		m.status = "reloaded"
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
var pendingOverlays = map[string]string{
	"D": "Design Review",
	"n": "new-story wizard", "u": "update", "x": "Quick Prompt",
	"o": "open session/PR", "S": "ship-safe",
}

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

// runCommand dispatches a `:` command line. Unported commands set a status note.
func (m Model) runCommand(line string) (tea.Model, tea.Cmd) {
	m.cmdBuf = ""
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return m, nil
	}
	switch fields[0] {
	case "q", "q!", "wq", "x", "quit":
		return m, tea.Quit
	case "e", "e!", "r", "reload":
		m.reload()
		m.status = "reloaded"
	case "help", "h", "?":
		m.showHelp = true
	case "P", "pipeline":
		m.setDetail("Pipeline", m.store.PipelineLines())
	case "C", "changes":
		m.setDetail("Git Changes", m.store.GitChanges())
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
	case "n", "req", "story":
		m.status = "new-story — not yet ported"
	case "u", "update":
		m.status = "update — not yet ported"
	case "labels":
		m.status = "labels — not yet ported"
	case "project":
		m.status = "project — not yet ported"
	case "debug":
		m.status = "debug — not yet ported"
	default:
		m.status = "unknown command: :" + fields[0]
	}
	return m, nil
}

// reload re-reads live project state into the panes.
func (m *Model) reload() {
	if m.store == nil {
		return
	}
	panes := m.group.Panes()
	m.stories = newStorySource(m.store.Stories())
	m.sessions = newSessionSource(m.store.Sessions())
	m.qa = newQASource(m.store.Tests())
	panes[paneStories] = pane.New("Stories", m.stories)
	panes[paneSessions] = pane.New("Sessions", m.sessions)
	panes[panePRs] = pane.New("Pull Requests", newPRSource(m.store.PullRequests()))
	panes[paneQA] = pane.New("QA / Tests", m.qa)
	// Rebuild the group so focus wiring stays consistent.
	m.group = pane.NewGroup(panes...)
	m.design = pane.New("Design", linesSource{m.store.DesignTree()})
	m.logs = pane.New("Logs", linesSource{m.store.LogLines(500)})
	m.design.SetFocus(true)
	m.logs.SetFocus(true)
}

func (m Model) handleMouse(msg tea.MouseMsg) {
	if m.splash {
		return
	}
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonWheelUp {
			m.group.ScrollAt(msg.X, msg.Y, -1)
		} else if msg.Button == tea.MouseButtonWheelDown {
			m.group.ScrollAt(msg.X, msg.Y, 1)
		} else if msg.Button == tea.MouseButtonLeft {
			m.group.FocusAt(msg.X, msg.Y)
		}
	}
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
	header := m.header()
	bodyH := m.height - lipgloss.Height(header) - 1
	if bodyH < 2 {
		bodyH = 2
	}
	body := m.grid(bodyH)
	if m.showHelp {
		body = m.helpView(bodyH)
	} else if p := m.activePane(); p != nil {
		body = p.RenderAt(0, lipgloss.Height(header), m.width, bodyH, m.mode)
	}
	return kittyClearImages + lipgloss.JoinVertical(lipgloss.Left, header, body, m.footer())
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

	active := [][2]string{
		{"Tab / Shift+Tab", "cycle panes"},
		{"j / k · ↓ / ↑", "navigate rows"},
		{"g / G", "top / bottom"},
		{"s a p Q", "focus Stories / Sessions / PRs / QA"},
		{"Enter", "detail (story / session / test)"},
		{"C", "git changes (status + diff)"},
		{"P / F", "pipeline status / skills"},
		{"d / l", "Design / Logs full-screen"},
		{"/", "filter the focused pane"},
		{":", "command mode (:q :r :help …)"},
		{"r", "reload pane data"},
		{"?", "toggle this help"},
		{"esc", "close overlay"},
		{"q · Ctrl+C", "quit"},
	}
	coming := [][2]string{
		{"o", "open session / PR"},
		{"D", "Design Review"},
		{"n / u / F", "new story / update / skills"},
		{"x / P / S", "quick prompt / pipeline / ship-safe"},
		{":", "command mode"},
	}

	var lines []string
	lines = append(lines, m.mode.Role("title").Style().Render("Keybindings"), "")
	for _, r := range active {
		lines = append(lines, row(r[0], r[1], false))
	}
	lines = append(lines, "", m.mode.Role("subtitle").Style().Render("Coming in the rebuild"))
	for _, r := range coming {
		lines = append(lines, row(r[0], r[1], true))
	}
	lines = append(lines, "", faint.Render("press any key to close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.mode.Role("border_focus").FG)).
		Padding(1, 3).
		Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, box)
}

// header is the top bar: the MAPLE logo + tagline, a project/theme status line, and
// Gherkin/Taffy badges — matching the OG dashboard header.
func (m Model) header() string {
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
	info := faint.Render(fmt.Sprintf("  project: %s · theme: %s · maple %s", name, m.theme.Name, m.version))
	badges := accent.Render(fmt.Sprintf("  📋 Gherkin: %d · ▶ Taffy: %d (%d running)",
		len(m.store.Stories()), m.store.TaffyCount(), running))

	return lipgloss.JoinVertical(lipgloss.Left, brand.Logo(m.mode), info, badges)
}

// footer is the bottom bar: the filter input when active, a transient status when
// set, otherwise context key hints. Right side always shows "? help".
func (m Model) footer() string {
	var left string
	switch {
	case m.filtering:
		left = m.mode.Role("accent").Style().Render("/" + m.filterBuf + "▏")
	case m.commanding:
		left = m.mode.Role("accent").Style().Render(":" + m.cmdBuf + "▏")
	case m.status != "":
		left = m.mode.Role("subtitle").Style().Render(m.status)
	default:
		left = m.mode.Role("faint").Style().Render("tab focus · ↑/↓ move · / filter · r reload · q quit")
	}
	right := m.mode.Role("faint").Style().Render("? help")
	return bar(left, right, m.width)
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
