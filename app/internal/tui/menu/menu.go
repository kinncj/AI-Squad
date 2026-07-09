// Package menu is the interactive setup menu shown when `maple` runs in a directory
// that isn't initialised yet. It offers Init / Requirements / Labels / Project / Help,
// enabling each based on the tools present. Ported from tui/menu.go.
package menu

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kinncj/maple/app/internal/tui/theme"
)

// Action is the workflow the user picked from the menu.
type Action int

const (
	ActionNone Action = iota
	ActionInit
	ActionUpdate
	ActionReq
	ActionLabels
	ActionProject
	ActionHelp
	ActionQuit
)

type item struct {
	action   Action
	label    string
	desc     string
	disabled bool
	why      string
}

type palette struct {
	Primary, Accent, Muted, Foreground, Success lipgloss.Color
}

func loadPalette() palette {
	p := palette{Primary: "#7aa2f7", Accent: "#bb9af7", Muted: "#565f89", Foreground: "#c0caf5", Success: "#9ece6a"}
	t, err := theme.Load()
	if err != nil {
		return p
	}
	m := t.ActiveMode()
	if r := m.Role("title"); r.FG != "" {
		p.Primary = lipgloss.Color(r.FG)
	}
	if r := m.Role("accent"); r.FG != "" {
		p.Accent = lipgloss.Color(r.FG)
	}
	if r := m.Role("faint"); r.FG != "" {
		p.Muted = lipgloss.Color(r.FG)
	}
	if r := m.Role("base"); r.FG != "" {
		p.Foreground = lipgloss.Color(r.FG)
	}
	if s := m.State("done"); s.FG != "" {
		p.Success = lipgloss.Color(s.FG)
	}
	return p
}

// tools captures which optional binaries are present.
type tools struct {
	hasAI bool
	hasGH bool
	names []string
}

func detectTools() tools {
	t := tools{}
	for _, bin := range []string{"claude", "copilot", "opencode", "cursor-agent"} {
		if _, err := exec.LookPath(bin); err == nil {
			t.hasAI = true
			t.names = append(t.names, bin+" ✓")
		}
	}
	if _, err := exec.LookPath("gh"); err == nil {
		t.hasGH = true
		t.names = append(t.names, "gh ✓")
	}
	if len(t.names) == 0 {
		t.names = []string{"no AI tools or gh detected"}
	}
	return t
}

func buildItems(t tools, initialized bool) []item {
	items := []item{{action: ActionInit, label: "Init", desc: "Set up MAPLE in this directory"}}
	if initialized {
		items = append(items, item{action: ActionUpdate, label: "Update", desc: "Re-sync agents, skills, and hooks with latest templates"})
	}
	req := item{action: ActionReq, label: "Requirements", desc: "Write requirements → Gherkin story"}
	if !t.hasAI {
		req.disabled, req.why = true, "needs claude / copilot / opencode"
	}
	items = append(items, req)
	labels := item{action: ActionLabels, label: "Labels", desc: "Bootstrap GitHub label set in current repo"}
	proj := item{action: ActionProject, label: "Project", desc: "Create GitHub Project v2"}
	if !t.hasGH {
		labels.disabled, labels.why = true, "needs gh CLI"
		proj.disabled, proj.why = true, "needs gh CLI"
	}
	items = append(items, labels, proj, item{action: ActionHelp, label: "Help", desc: "Show documentation"})
	return items
}

type model struct {
	pal         palette
	items       []item
	cursor      int
	result      Action
	showHelp    bool
	initialized bool
	cwd         string
	tools       tools
	version     string
}

// Run shows the setup menu and returns the chosen Action (ActionQuit if dismissed).
func Run(version string) (Action, error) {
	_, err := os.Stat("project.config.yaml")
	initialized := err == nil
	cwd, _ := os.Getwd()
	t := detectTools()
	m := &model{
		pal:         loadPalette(),
		items:       buildItems(t, initialized),
		initialized: initialized,
		cwd:         cwd,
		tools:       t,
		version:     version,
		result:      ActionQuit,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, runErr := p.Run()
	if runErr != nil {
		return ActionQuit, runErr
	}
	return final.(*model).result, nil
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		switch key.String() {
		case "ctrl+c", "q", "esc":
			m.result = ActionQuit
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "enter", " ":
			it := m.items[m.cursor]
			if it.disabled {
				return m, nil
			}
			if it.action == ActionHelp {
				m.showHelp = true
				return m, nil
			}
			m.result = it.action
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) moveCursor(dir int) {
	n := len(m.items)
	for i := 0; i < n; i++ {
		m.cursor = (m.cursor + dir + n) % n
		if !m.items[m.cursor].disabled {
			return
		}
	}
}

func (m *model) View() string {
	if m.showHelp {
		return m.helpView()
	}
	return m.menuView()
}

func (m *model) menuView() string {
	p := m.pal
	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render("🍁 MAPLE")
	ver := lipgloss.NewStyle().Foreground(p.Muted).Render(" · " + m.version)
	sb.WriteString("\n  " + title + ver + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  "+strings.Repeat("─", 54)) + "\n\n")

	cursor := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render("❯")
	for i, it := range m.items {
		var line string
		switch {
		case it.disabled:
			label := lipgloss.NewStyle().Foreground(p.Muted).Render(fmt.Sprintf("%-14s", it.label))
			why := lipgloss.NewStyle().Foreground(p.Muted).Italic(true).Render(it.why)
			line = "    " + label + "  " + why
		case i == m.cursor:
			label := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render(fmt.Sprintf("%-14s", it.label))
			desc := lipgloss.NewStyle().Foreground(p.Foreground).Render(it.desc)
			line = "  " + cursor + " " + label + "  " + desc
		default:
			label := lipgloss.NewStyle().Foreground(p.Foreground).Render(fmt.Sprintf("%-14s", it.label))
			desc := lipgloss.NewStyle().Foreground(p.Muted).Render(it.desc)
			line = "    " + label + "  " + desc
		}
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(p.Muted).Render("  "+strings.Repeat("─", 54)) + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  ↑/↓ j/k Navigate   Enter Select   q Quit") + "\n\n")

	cwd := m.cwd
	if len(cwd) > 50 {
		cwd = "…" + cwd[len(cwd)-49:]
	}
	initStatus := lipgloss.NewStyle().Foreground(p.Muted).Render("○ Not initialized")
	if m.initialized {
		initStatus = lipgloss.NewStyle().Foreground(p.Success).Render("● Initialized")
	}
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(p.Muted).Render(cwd) + "  " + initStatus + "\n")
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(p.Muted).Render(strings.Join(m.tools.names, "  ")) + "\n")
	return sb.String()
}

func (m *model) helpView() string {
	p := m.pal
	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render("Documentation")
	sb.WriteString("\n  " + title + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  "+strings.Repeat("─", 54)) + "\n\n")
	sections := [][2]string{
		{"Init", "Copies agents, skills, hooks, and config into the current directory."},
		{"Update", "Re-syncs managed files with the latest templates. Never overwrites project.config.yaml."},
		{"Requirements", "Type plain-text requirements, Ctrl+D to convert → Gherkin story in docs/stories/."},
		{"Labels", "Creates the canonical MAPLE GitHub label set via gh CLI."},
		{"Project", "Creates a GitHub Project v2 and writes its ids into project.config.yaml."},
	}
	for _, s := range sections {
		label := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render(fmt.Sprintf("  %-14s", s[0]))
		desc := lipgloss.NewStyle().Foreground(p.Foreground).Render(s[1])
		sb.WriteString(label + "  " + desc + "\n\n")
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  Press any key to return.") + "\n")
	return sb.String()
}
