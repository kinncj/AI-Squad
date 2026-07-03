// Package dashboard is the maple TUI's top-level Bubble Tea model. It composes the
// pane primitive, theme, splash, and (in later sub-projects) the state adapters.
// This is the walking skeleton: it proves the architecture end-to-end with
// placeholder sources; real state sources replace demoSource in a later step.
package dashboard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kinncj/maple/app/internal/tui/pane"
	"github.com/kinncj/maple/app/internal/tui/splash"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

// demoSource is a placeholder Selectable source. It stands in for the real state
// adapters (stories, sessions, PRs, QA) until app/internal/state lands.
type demoSource struct{ rows []string }

func (d demoSource) Rows() []string { return d.rows }
func (d demoSource) RowCount() int  { return len(d.rows) }

// Model is the top-level dashboard model.
type Model struct {
	theme   *theme.Theme
	mode    theme.Mode
	group   *pane.Group
	width   int
	height  int
	splash  bool
	version string
}

// New builds the dashboard model. It fails only if the embedded theme is malformed.
func New(version string) (Model, error) {
	th, err := theme.Load()
	if err != nil {
		return Model{}, err
	}
	g := pane.NewGroup(
		pane.New("Stories", demoSource{sample("story", 12)}),
		pane.New("Sessions", demoSource{sample("session", 8)}),
		pane.New("Pull Requests", demoSource{sample("PR #", 20)}),
		pane.New("QA", demoSource{sample("scenario", 15)}),
	)
	return Model{
		theme:   th,
		mode:    th.ActiveMode(),
		group:   g,
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

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		m.handleMouse(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.splash {
		// Any key dismisses the splash, except quit.
		if k := msg.String(); k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}
		m.splash = false
		return m, nil
	}
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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
	}
	return m, nil
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

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.splash {
		return splash.Render(m.width, m.height, "maple "+m.version, m.mode)
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.grid(), m.footer())
}

// grid lays the four panes out in a 2×2 grid filling the viewport above the footer.
func (m Model) grid() string {
	panes := m.group.Panes()
	if len(panes) == 0 {
		return ""
	}
	gridH := m.height - 1 // reserve the footer row
	if gridH < 2 {
		gridH = 2
	}
	colW := m.width / 2
	rowH := gridH / 2
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
			h = gridH - rowH
		}
		cells[i] = p.RenderAt(col*colW, row*rowH, w, h, m.mode)
	}
	top := lipgloss.JoinHorizontal(lipgloss.Top, cells[0], cells[1])
	if len(cells) < 4 {
		return top
	}
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, cells[2], cells[3])
	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func (m Model) footer() string {
	help := "tab focus · ↑/↓ move · g/G top/bottom · q quit"
	return m.mode.Role("faint").Style().Render(help)
}
