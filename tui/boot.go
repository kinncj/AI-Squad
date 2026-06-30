package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Boot check sequence (PRD §5.9.1) ────────────────────────────────────────

type bootCheck struct {
	label    string
	optional bool
	check    func() error
	result   error
	done     bool
}

type bootModel struct {
	theme   Theme
	checks  []*bootCheck
	current int
	width   int
	allDone bool

	// sweep animation for logo
	sweepFrame int
	sweepDone  bool
	pulseFrame int
	pulseDone  bool
}

type bootCheckDoneMsg struct{ idx int; err error }
type bootAllDoneMsg struct{}
type bootSweepTickMsg struct{}
type bootPulseTickMsg struct{}

func newBootModel(t Theme) *bootModel {
	return &bootModel{
		theme: t,
		checks: []*bootCheck{
			{label: "gh auth status", check: checkGHAuth},
			{label: "project.config.yaml", check: checkProjectConfig},
			{label: "Claude Code (claude)", check: checkClaudeBin},
			{label: "OpenCode (opencode)", optional: true, check: checkOpenCodeBin},
			{label: "npx (skills marketplace)", optional: true, check: checkNPXBin},
		},
	}
}

func checkGHAuth() error {
	out, err := runWithTimeout(timeoutAuth, nil, "gh", "auth", "status")
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = "not authenticated"
		}
		return fmt.Errorf("%s — run `gh auth login`", msg)
	}
	return nil
}

func checkProjectConfig() error {
	if _, err := os.Stat("project.config.yaml"); err != nil {
		return fmt.Errorf("not found — run `maple init`")
	}
	return nil
}

func checkClaudeBin() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("not found — install from claude.ai/code")
	}
	return nil
}

func checkOpenCodeBin() error {
	if _, err := exec.LookPath("opencode"); err != nil {
		return fmt.Errorf("not found — install from opencode.ai")
	}
	return nil
}

func checkNPXBin() error {
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("not found — install Node.js from nodejs.org to use skills marketplace")
	}
	return nil
}

func (m *bootModel) Init() tea.Cmd {
	return bootSweepTick()
}

func bootSweepTick() tea.Cmd {
	return tea.Tick(logoSweepDelay, func(time.Time) tea.Msg { return bootSweepTickMsg{} })
}

func bootPulseTick() tea.Cmd {
	return tea.Tick(logoPulseDelay, func(time.Time) tea.Msg { return bootPulseTickMsg{} })
}

func (m *bootModel) runCheck(i int) tea.Cmd {
	if i >= len(m.checks) {
		return func() tea.Msg { return bootAllDoneMsg{} }
	}
	chk := m.checks[i]
	return func() tea.Msg {
		time.Sleep(110 * time.Millisecond)
		return bootCheckDoneMsg{idx: i, err: chk.check()}
	}
}

func (m *bootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case bootSweepTickMsg:
		m.sweepFrame++
		if m.sweepFrame >= logoSweepFrameCount {
			m.sweepDone = true
			return m, tea.Batch(bootPulseTick(), m.runCheck(0))
		}
		return m, bootSweepTick()

	case bootPulseTickMsg:
		m.pulseFrame++
		if m.pulseFrame >= logoPulseFrameCount {
			m.pulseDone = true
		} else {
			return m, bootPulseTick()
		}

	case bootCheckDoneMsg:
		m.checks[msg.idx].done = true
		m.checks[msg.idx].result = msg.err
		m.current = msg.idx + 1
		return m, m.runCheck(m.current)

	case bootAllDoneMsg:
		m.allDone = true
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *bootModel) View() string {
	t := m.theme

	var hdr string
	switch {
	case !m.sweepDone:
		hdr = logoSweepFrame(m.sweepFrame, t.Primary)
	case !m.pulseDone:
		hdr = logoPulseFrame(m.pulseFrame, t.Primary, t.Accent)
	default:
		hdr = logoPulseFrame(logoPulseFrameCount, t.Primary, t.Accent)
	}

	var sb strings.Builder
	sb.WriteString(hdr)
	sb.WriteString("\n")

	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  System check")
	sb.WriteString(title + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("  "+strings.Repeat("─", 52)) + "\n\n")

	for i, c := range m.checks {
		var icon, text string
		switch {
		case !c.done:
			if i == m.current && m.sweepDone {
				icon = lipgloss.NewStyle().Foreground(t.Accent).Render("  ●")
			} else {
				icon = lipgloss.NewStyle().Foreground(t.Muted).Render("  ○")
			}
			text = lipgloss.NewStyle().Foreground(t.Muted).Render(c.label)
		case c.result == nil:
			icon = lipgloss.NewStyle().Foreground(t.Success).Render("  ✓")
			text = lipgloss.NewStyle().Foreground(t.Foreground).Render(c.label)
		case c.optional:
			icon = lipgloss.NewStyle().Foreground(t.Warning).Render("  ~")
			text = lipgloss.NewStyle().Foreground(t.Muted).Render(
				c.label + "  " + c.result.Error())
		default:
			icon = lipgloss.NewStyle().Foreground(t.Error).Render("  ✗")
			text = lipgloss.NewStyle().Foreground(t.Error).Render(
				c.label + "  " + c.result.Error())
		}
		sb.WriteString(icon + " " + text + "\n")
	}

	if m.allDone {
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("  Loading dashboard…") + "\n")
		if !detectTruecolor() {
			sb.WriteString(lipgloss.NewStyle().Foreground(t.Warning).Render(
				"  ⚠ 256-color mode detected — theme colors may differ from screenshots") + "\n")
		}
	}
	return sb.String()
}

// runBoot runs the boot check sequence and returns the theme to use for the
// dashboard. If any required check fails, it still proceeds (user can see
// warnings on dashboard). Returns false if user quit during boot.
func runBoot() (Theme, bool) {
	t, _ := detectOmarchyTheme()
	m := newBootModel(t)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return t, false
	}
	bm := final.(*bootModel)
	return bm.theme, bm.allDone
}
