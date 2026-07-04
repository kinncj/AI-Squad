// Package dashboard is the maple TUI's top-level Bubble Tea model. It composes the
// pane primitive, theme, splash, and (in later sub-projects) the state adapters.
// This is the walking skeleton: it proves the architecture end-to-end with
// placeholder sources; real state sources replace demoSource in a later step.
package dashboard

import (
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

// demoSource is a placeholder Selectable source. It stands in for the state adapters
// not yet built (sessions, PRs, QA); Stories already uses the real state layer.
type demoSource struct{ rows []string }

func (d demoSource) Rows() []string { return d.rows }
func (d demoSource) RowCount() int  { return len(d.rows) }

// Model is the top-level dashboard model.
type Model struct {
	theme     *theme.Theme
	mode      theme.Mode
	group     *pane.Group
	store     state.StoryStore
	width     int
	height    int
	splash    bool
	showHelp  bool
	filtering bool
	filterBuf string
	status    string
	version   string
}

// New builds the dashboard model from a story store. Pass state.NewFS(".") in
// production or a fake in tests. It fails only if the embedded theme is malformed.
func New(version string, store state.StoryStore) (Model, error) {
	th, err := theme.Load()
	if err != nil {
		return Model{}, err
	}
	g := pane.NewGroup(
		pane.New("Stories", newStorySource(store.Stories())),
		pane.New("Sessions", demoSource{sample("session", 8)}),
		pane.New("Pull Requests", demoSource{sample("PR #", 20)}),
		pane.New("QA", demoSource{sample("scenario", 15)}),
	)
	return Model{
		theme:   th,
		mode:    th.ActiveMode(),
		group:   g,
		store:   store,
		splash:  true,
		version: version,
	}, nil
}

func sample(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = prefix + " " + itoa(i+1)
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

	m.status = "" // any key clears a transient status line
	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = true
	case "/":
		m.filtering, m.filterBuf = true, m.group.Focused().Filter()
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

// pendingOverlays maps keys to the overlays still to be ported from tui/.
var pendingOverlays = map[string]string{
	"d": "Design pane", "D": "Design Review", "l": "Logs pane", "C": "Git Changes",
	"n": "new-story wizard", "u": "update", "F": "Skills marketplace", "x": "Quick Prompt",
	"P": "Pipeline status", "o": "open session/PR", "S": "ship-safe", "enter": "detail popup",
	":": "command mode",
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

// reload re-reads live project state into the panes.
func (m *Model) reload() {
	if m.store != nil {
		m.group.Panes()[paneStories] = pane.New("Stories", newStorySource(m.store.Stories()))
		// Rebuild the group so focus wiring stays consistent.
		m.group = pane.NewGroup(m.group.Panes()...)
	}
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
	// header (1 row) + body + footer (1 row).
	bodyH := m.height - 2
	if bodyH < 2 {
		bodyH = 2
	}
	body := m.grid(bodyH)
	if m.showHelp {
		body = m.helpView(bodyH)
	}
	return kittyClearImages + lipgloss.JoinVertical(lipgloss.Left, m.header(), body, m.footer())
}

// helpView renders the full keybinding + command reference, centered in bodyH rows.
func (m Model) helpView(bodyH int) string {
	title := m.mode.Role("title").Style()
	key := m.mode.Role("accent").Style()
	desc := m.mode.Role("base").Style()
	row := func(k, d string) string {
		return "  " + key.Render(pad(k, 18)) + desc.Render(d)
	}
	lines := []string{
		title.Render("  Keybindings"),
		"",
		row("Tab / Shift+Tab", "cycle panes"),
		row("j / k  ↓ / ↑", "navigate rows"),
		row("g / G", "jump to top / bottom"),
		row("s  a  p  Q", "focus Stories / Sessions / PRs / QA"),
		row("/", "filter the focused pane"),
		row("r", "reload pane data"),
		row("Enter", "open detail (coming)"),
		row("o", "open session / PR (coming)"),
		row("d / l", "Design / Logs full-screen (coming)"),
		row("D / C", "Design Review / Git Changes (coming)"),
		row("n / u / F", "new story / update / skills (coming)"),
		row("x / P / S", "quick prompt / pipeline / ship-safe (coming)"),
		row(":", "command mode (coming)"),
		row("?", "toggle this help"),
		row("q  Ctrl+C", "quit"),
		"",
		m.mode.Role("faint").Style().Render("  press any key to close"),
	}
	block := strings.Join(lines, "\n")
	return lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, block)
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// header is the top bar: brand + version on the left, context on the right.
func (m Model) header() string {
	left := m.mode.Role("leaf").Style().Render(brand.Leaf+" maple") +
		m.mode.Role("faint").Style().Render(" "+m.version)
	right := ""
	if p := m.group.Focused(); p != nil {
		right = m.mode.Role("subtitle").Style().Render(p.Title)
	}
	return bar(left, right, m.width)
}

// footer is the bottom bar: the filter input when active, a transient status when
// set, otherwise context key hints. Right side always shows "? help".
func (m Model) footer() string {
	var left string
	switch {
	case m.filtering:
		left = m.mode.Role("accent").Style().Render("/" + m.filterBuf + "▏")
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
