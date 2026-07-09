// Package req is the Bubble Tea UI for `maple req`: it detects an AI harness, takes a
// free-text requirement, converts it to Gherkin stories via app/internal/req, and can
// launch /pipeline-runner implement-stories through a harness. Ported from tui/req.go.
package req

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kinncj/maple/app/internal/harness"
	core "github.com/kinncj/maple/app/internal/req"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

// palette is the flat colour set the req views need, resolved from the app theme.
type palette struct {
	Primary, Accent, Muted, Foreground, Background lipgloss.Color
	Success, Error, Warning                        lipgloss.Color
}

func loadPalette() palette {
	t, err := theme.Load()
	if err != nil {
		return palette{
			Primary: "#7aa2f7", Accent: "#bb9af7", Muted: "#565f89",
			Foreground: "#c0caf5", Background: "#1a1b26",
			Success: "#9ece6a", Error: "#f7768e", Warning: "#e0af68",
		}
	}
	m := t.ActiveMode()
	role := func(name, fallback string) lipgloss.Color {
		if r := m.Role(name); r.FG != "" {
			return lipgloss.Color(r.FG)
		}
		return lipgloss.Color(fallback)
	}
	state := func(name, fallback string) lipgloss.Color {
		if s := m.State(name); s.FG != "" {
			return lipgloss.Color(s.FG)
		}
		return lipgloss.Color(fallback)
	}
	base := m.Role("base")
	bg := lipgloss.Color(base.BG)
	if base.BG == "" {
		bg = "#1a1b26"
	}
	return palette{
		Primary:    role("title", "#7aa2f7"),
		Accent:     role("accent", "#bb9af7"),
		Muted:      role("faint", "#565f89"),
		Foreground: role("base", "#c0caf5"),
		Background: bg,
		Success:    state("done", "#9ece6a"),
		Error:      state("blocked", "#f7768e"),
		Warning:    state("running", "#e0af68"),
	}
}

type step int

const (
	stepPickAI step = iota
	stepEdit
	stepSending
	stepStories
	stepViewStory
	stepPickImpl
)

type model struct {
	pal          palette
	tools        []core.Tool
	selectedAI   core.Tool
	aiCursor     int
	textarea     textarea.Model
	lastReq      string
	step         step
	spinner      spinner.Model
	sendingStart time.Time
	err          error
	width        int
	height       int
	stories      []core.Story
	storyCursor  int
	viewingIdx   int
	scrollOffset int
	storyListTop int
	implReturnTo step
	implCursor   int
	cancelSend   context.CancelFunc
	streamCh     chan tea.Msg
	streamLog    []string
	implNote     string
}

type doneMsg struct {
	stories []core.Story
	err     error
}

// streamMsg is one line of live output from the harness during conversion.
type streamMsg struct{ line string }

// waitFor blocks on the stream channel and delivers the next message (or nil when the
// producer closes the channel), so the Bubble Tea loop re-arms one read at a time.
func waitFor(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

type spawnResultMsg struct {
	err error
}

// implLaunchedMsg means the harness opened in a side pane; maple stays on the stories view.
type implLaunchedMsg struct{ harness string }

// Run detects harnesses and runs the requirements TUI.
func Run() error {
	tools := core.AvailableTools()
	if len(tools) == 0 {
		return fmt.Errorf("no AI tool available — install claude, copilot, opencode, or cursor-agent")
	}
	m := newModel(tools)

	const handoff = ".claude/state/maple-edit.txt"
	if data, err := os.ReadFile(handoff); err == nil {
		m.textarea.SetValue(string(data))
		m.lastReq = string(data)
		_ = os.Remove(handoff)
		m.step = stepEdit
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func newModel(tools []core.Tool) *model {
	pal := loadPalette()
	ta := textarea.New()
	ta.Placeholder = "Describe what you need…\n\nExamples:\n  As a user I want to export filtered results as CSV\n  so that I can share them with stakeholders offline.\n\n  The export must respect active filters.\n  Only the last 90 days of data should be included."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.ShowLineNumbers = false

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(pal.Accent)

	first := stepEdit
	if len(tools) > 1 {
		first = stepPickAI
	}
	return &model{
		pal:        pal,
		tools:      tools,
		selectedAI: tools[0],
		textarea:   ta,
		spinner:    s,
		step:       first,
	}
}

func (m *model) Init() tea.Cmd { return textarea.Blink }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(msg.Height - 12)

	case spinner.TickMsg:
		if m.step == stepSending {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		switch m.step {
		case stepPickAI:
			switch msg.String() {
			case "up", "k":
				if m.aiCursor > 0 {
					m.aiCursor--
				}
			case "down", "j":
				if m.aiCursor < len(m.tools)-1 {
					m.aiCursor++
				}
			case "enter", " ":
				m.selectedAI = m.tools[m.aiCursor]
				m.step = stepEdit
				return m, textarea.Blink
			case "ctrl+c", "q", "esc":
				return m, tea.Quit
			}
			return m, nil

		case stepSending:
			if msg.String() == "ctrl+c" || msg.String() == "esc" {
				if m.cancelSend != nil {
					m.cancelSend()
				}
				return m, tea.Quit
			}
			return m, nil

		case stepEdit:
			switch msg.Type {
			case tea.KeyCtrlD:
				req := strings.TrimSpace(m.textarea.Value())
				if req == "" {
					return m, nil
				}
				m.lastReq = req
				m.step = stepSending
				m.sendingStart = time.Now()
				return m, tea.Batch(m.spinner.Tick, m.convert(req))
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit
			}

		case stepStories:
			n := len(m.stories)
			switch msg.String() {
			case "up", "k":
				if m.storyCursor > 0 {
					m.storyCursor--
				}
			case "down", "j":
				if m.storyCursor < n-1 {
					m.storyCursor++
				}
			case "enter", " ":
				m.viewingIdx = m.storyCursor
				m.scrollOffset = 0
				m.step = stepViewStory
			case "e":
				m.step = stepEdit
				return m, textarea.Blink
			case "R":
				if m.lastReq != "" {
					m.step = stepSending
					m.sendingStart = time.Now()
					return m, tea.Batch(m.spinner.Tick, m.convert(m.lastReq))
				}
			case "q", "esc":
				return m, tea.Quit
			case "i":
				if len(m.stories) > 0 {
					m.implReturnTo = stepStories
					m.implCursor = m.defaultHarnessCursor()
					m.step = stepPickImpl
				}
			}
			return m, nil

		case stepViewStory:
			story := m.stories[m.viewingIdx]
			lines := strings.Split(story.Gherkin, "\n")
			visible := m.visibleLines()
			maxScroll := len(lines) - visible
			if maxScroll < 0 {
				maxScroll = 0
			}
			switch msg.String() {
			case "up", "k":
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
			case "down", "j":
				if m.scrollOffset < maxScroll {
					m.scrollOffset++
				}
			case "g":
				m.scrollOffset = 0
			case "G":
				m.scrollOffset = maxScroll
			case "]", "tab", "l":
				if m.viewingIdx < len(m.stories)-1 {
					m.viewingIdx++
					m.scrollOffset = 0
				}
			case "[", "shift+tab", "h":
				if m.viewingIdx > 0 {
					m.viewingIdx--
					m.scrollOffset = 0
				}
			case "b", "esc":
				m.scrollOffset = 0
				m.step = stepStories
			case "q":
				return m, tea.Quit
			case "i":
				m.implReturnTo = stepViewStory
				m.implCursor = m.defaultHarnessCursor()
				m.step = stepPickImpl
			}
			return m, nil

		case stepPickImpl:
			switch msg.String() {
			case "up", "k":
				if m.implCursor > 0 {
					m.implCursor--
				}
			case "down", "j":
				if m.implCursor < len(m.tools)-1 {
					m.implCursor++
				}
			case "enter":
				if m.implCursor < len(m.tools) {
					m.step = m.implReturnTo
					return m, m.launchImplementation(m.tools[m.implCursor])
				}
			case "esc":
				m.step = m.implReturnTo
			case "q":
				return m, tea.Quit
			}
			return m, nil
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if m.step == stepStories && len(m.stories) > 0 {
				idx := msg.Y - m.storyListTop
				if idx >= 0 && idx < len(m.stories) {
					if idx == m.storyCursor {
						m.viewingIdx = idx
						m.scrollOffset = 0
						m.step = stepViewStory
					} else {
						m.storyCursor = idx
					}
				}
			}
		}

	case streamMsg:
		if m.step == stepSending {
			m.streamLog = append(m.streamLog, msg.line)
			const maxLog = 500
			if len(m.streamLog) > maxLog {
				m.streamLog = m.streamLog[len(m.streamLog)-maxLog:]
			}
			return m, waitFor(m.streamCh)
		}
		return m, nil

	case doneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.step = stepStories
			return m, nil
		}
		m.err = nil
		m.stories = msg.stories
		m.storyCursor = 0
		m.step = stepStories

	case implLaunchedMsg:
		m.implNote = msg.harness + " launched in a side pane → switch panes to watch it (herdr/tmux pane key)"
		return m, nil

	case spawnResultMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("could not open a new terminal.\nRun manually:\n%s", msg.err.Error())
			m.step = stepStories
			return m, nil
		}
		return m, tea.Quit
	}

	if m.step == stepEdit {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) visibleLines() int {
	v := m.height - 7
	if v < 5 {
		v = 5
	}
	return v
}

func (m *model) defaultHarnessCursor() int {
	for i, opt := range m.tools {
		if opt.Kind == m.selectedAI.Kind {
			return i
		}
	}
	return 0
}

func (m *model) convert(requirements string) tea.Cmd {
	ai := m.selectedAI
	var oldDirs []string
	for _, s := range m.stories {
		if s.SavedTo != "" {
			oldDirs = append(oldDirs, s.SavedTo)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelSend = cancel
	ch := make(chan tea.Msg, 128)
	m.streamCh = ch
	m.streamLog = nil
	go func() {
		defer cancel()
		defer close(ch)
		send := func(msg tea.Msg) {
			select {
			case ch <- msg:
			case <-ctx.Done():
			}
		}
		for _, d := range oldDirs {
			_ = os.RemoveAll(d)
		}
		gherkin, err := core.StreamToGherkin(ctx, requirements, ai, func(l string) { send(streamMsg{line: l}) })
		if err != nil {
			send(doneMsg{err: err})
			return
		}
		stories := core.ParseStories(gherkin)
		for i := range stories {
			stories[i].UI = core.InferUIStory(requirements, stories[i].Title, stories[i].Gherkin)
			if saved, saveErr := core.SaveStory(stories[i].Title, stories[i].Gherkin, stories[i].UI, i+1, time.Now()); saveErr == nil {
				stories[i].SavedTo = saved
			}
		}
		send(doneMsg{stories: stories})
	}()
	return tea.Batch(m.spinner.Tick, waitFor(ch))
}

func (m *model) launchImplementation(ai core.Tool) tea.Cmd {
	stories := m.stories
	if len(stories) == 0 {
		return func() tea.Msg { return doneMsg{err: fmt.Errorf("no generated stories to implement")} }
	}
	if _, err := os.Stat(taffyPathForHarness(ai.Kind)); err != nil {
		return func() tea.Msg {
			return doneMsg{err: fmt.Errorf("missing implement-stories workflow for %s. run `maple update` and try again", ai.Label)}
		}
	}
	if err := writeImplementationHandoff(stories); err != nil {
		return func() tea.Msg { return doneMsg{err: fmt.Errorf("failed to write gherkin handoff: %w", err)} }
	}
	writeQuickLaunchState("pipeline-runner implement-stories", "starting", ai.Kind)
	prompt := buildImplementationPrompt(ai.Kind, stories)
	args := buildLaunchCmd(ai.Kind, prompt, true)

	// Prefer a side pane (same as the dashboard L key) so maple stays visible beside the
	// harness. Only take over the current terminal when there's no multiplexer to split.
	if harness.InMultiplexer(os.Getenv) {
		label := ai.Label
		return func() tea.Msg {
			if _, _, err := harness.LaunchInPane(os.Getenv, ai.Kind, args); err != nil {
				return spawnResultMsg{err: err}
			}
			return implLaunchedMsg{harness: label}
		}
	}
	c := exec.Command(args[0], args[1:]...)
	c.Env = os.Environ()
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return spawnResultMsg{err: err}
	})
}
