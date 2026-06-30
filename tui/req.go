package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Requirements command ─────────────────────────────────────────────────────

// availableAITools returns supported req harnesses currently available on PATH.
// gh copilot (explain/suggest) is excluded — it only handles shell commands.
func availableAITools(tools Tools) []aiOption {
	var opts []aiOption
	for _, h := range tools.Harnesses() {
		if h.SupportsReq {
			opts = append(opts, aiOption{h.ReqLabel, h.Key, h.Bin})
		}
	}
	return opts
}

func runReq(tools Tools) error {
	// Refresh tool detection on each req launch so newly-installed harnesses
	// show up immediately in the picker without restarting maple.
	tools = Detect()
	opts := availableAITools(tools)
	if len(opts) == 0 {
		return fmt.Errorf("no AI tool available — install claude, copilot, opencode, or cursor-agent")
	}
	t, _ := detectOmarchyTheme()
	m := newReqModel(tools, t)

	// Pick up story content handed off from the dashboard re-edit flow.
	const handoff = ".claude/state/maple-edit.txt"
	if data, err := os.ReadFile(handoff); err == nil {
		m.textarea.SetValue(string(data))
		m.lastReq = string(data)
		_ = os.Remove(handoff) // consume so next run starts blank
		m.step = reqStepEdit
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// ─── Model ────────────────────────────────────────────────────────────────────

type reqStep int

const (
	reqStepPickAI    reqStep = iota // shown only when >1 AI tool available
	reqStepEdit                     // textarea for requirements
	reqStepSending                  // waiting for AI
	reqStepStories                  // list of converted stories
	reqStepViewStory                // full content of one story
	reqStepPickImplementationHarness
)

type gherkinStory struct {
	title   string
	gherkin string
	ui      bool
	savedTo string
}

type reqModel struct {
	theme        Theme
	tools        Tools
	aiOptions    []aiOption
	selectedAI   aiOption
	aiCursor     int
	textarea     textarea.Model
	lastReq      string // saved so R can regenerate
	step         reqStep
	spinner      spinner.Model
	sendingStart time.Time
	sendingTick  int
	err          error
	width        int
	height       int
	logoFrame    int
	logoDone     bool
	stories      []gherkinStory
	storyCursor  int
	viewingIdx   int
	scrollOffset int
	storyListTop int // Y row where first story item is rendered (for mouse clicks)
	implReturnTo reqStep
	implCursor   int
	cancelSend   context.CancelFunc
}

type reqDoneMsg struct {
	stories []gherkinStory
	err     error
}

type sendingTickMsg struct{}

func sendingTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return sendingTickMsg{} })
}

func newReqModel(tools Tools, t Theme) *reqModel {
	ta := textarea.New()
	ta.Placeholder = "Describe what you need…\n\nExamples:\n  As a user I want to export filtered results as CSV\n  so that I can share them with stakeholders offline.\n\n  The export must respect active filters.\n  Only the last 90 days of data should be included.\n  The file must be downloadable immediately."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.ShowLineNumbers = false

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(t.Accent)

	opts := availableAITools(tools)
	firstStep := reqStepEdit
	if len(opts) > 1 {
		firstStep = reqStepPickAI
	}

	var selected aiOption
	if len(opts) > 0 {
		selected = opts[0]
	}

	return &reqModel{
		theme:      t,
		tools:      tools,
		aiOptions:  opts,
		selectedAI: selected,
		textarea:   ta,
		spinner:    s,
		step:       firstStep,
	}
}

func (m *reqModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, logoTick(0))
}

func (m *reqModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(msg.Height - 12)
		if m.step == reqStepViewStory && len(m.stories) > 0 {
			lines := strings.Split(m.stories[m.viewingIdx].gherkin, "\n")
			maxScroll := len(lines) - m.visibleLines()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset > maxScroll {
				m.scrollOffset = maxScroll
			}
		}

	case logoTickMsg:
		if !m.logoDone {
			m.logoFrame++
			if m.logoFrame >= logoFrameCount {
				m.logoDone = true
			} else {
				return m, logoTick(m.logoFrame)
			}
		}

	case spinner.TickMsg:
		if m.step == reqStepSending {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case sendingTickMsg:
		if m.step == reqStepSending {
			m.sendingTick++
			return m, sendingTick()
		}

	case tea.KeyMsg:
		if !m.logoDone {
			return m, nil
		}

		switch m.step {
		case reqStepPickAI:
			switch msg.String() {
			case "up", "k":
				if m.aiCursor > 0 {
					m.aiCursor--
				}
			case "down", "j":
				if m.aiCursor < len(m.aiOptions)-1 {
					m.aiCursor++
				}
			case "enter", " ":
				m.selectedAI = m.aiOptions[m.aiCursor]
				m.step = reqStepEdit
				return m, textarea.Blink
			case "ctrl+c", "q", "esc":
				return m, tea.Quit
			}
			return m, nil

		case reqStepSending:
			switch msg.String() {
			case "ctrl+c", "esc":
				if m.cancelSend != nil {
					m.cancelSend()
				}
				return m, tea.Quit
			}
			return m, nil

		case reqStepEdit:
			switch msg.Type {
			case tea.KeyCtrlD:
				req := strings.TrimSpace(m.textarea.Value())
				if req == "" {
					return m, nil
				}
				m.lastReq = req
				m.step = reqStepSending
				m.sendingStart = time.Now()
				m.sendingTick = 0
				return m, tea.Batch(m.spinner.Tick, sendingTick(), m.convert(req))
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit
			}

		case reqStepStories:
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
				m.step = reqStepViewStory
			case "e":
				// keep previous text — let user refine
				m.step = reqStepEdit
				return m, textarea.Blink
			case "R":
				if m.lastReq != "" {
					m.step = reqStepSending
					m.sendingStart = time.Now()
					m.sendingTick = 0
					return m, tea.Batch(m.spinner.Tick, sendingTick(), m.convert(m.lastReq))
				}
			case "q", "esc":
				return m, tea.Quit
			case "i":
				if len(m.stories) > 0 {
					m.implReturnTo = reqStepStories
					m.implCursor = m.defaultHarnessCursor()
					m.step = reqStepPickImplementationHarness
				}
			}
			return m, nil

		case reqStepViewStory:
			story := m.stories[m.viewingIdx]
			lines := strings.Split(story.gherkin, "\n")
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
				m.step = reqStepStories
			case "q":
				return m, tea.Quit
			case "i":
				if len(m.stories) > 0 {
					m.implReturnTo = reqStepViewStory
					m.implCursor = m.defaultHarnessCursor()
					m.step = reqStepPickImplementationHarness
				}
			}
			return m, nil

		case reqStepPickImplementationHarness:
			switch msg.String() {
			case "up", "k":
				if m.implCursor > 0 {
					m.implCursor--
				}
			case "down", "j":
				if m.implCursor < len(m.aiOptions)-1 {
					m.implCursor++
				}
			case "enter":
				if m.implCursor < len(m.aiOptions) {
					m.step = m.implReturnTo
					return m, m.launchImplementationCmd(m.aiOptions[m.implCursor])
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
			if m.step == reqStepStories && len(m.stories) > 0 {
				idx := msg.Y - m.storyListTop
				if idx >= 0 && idx < len(m.stories) {
					if idx == m.storyCursor {
						// second click on already-selected row → open
						m.viewingIdx = idx
						m.scrollOffset = 0
						m.step = reqStepViewStory
					} else {
						m.storyCursor = idx
					}
				}
			}
		}

	case reqDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.step = reqStepStories
			return m, nil
		}
		m.err = nil
		m.stories = msg.stories
		m.storyCursor = 0
		m.step = reqStepStories

	case spawnSucceededMsg:
		return m, tea.Quit

	case spawnFailedMsg:
		m.err = fmt.Errorf("could not open a new terminal.\nRun manually:\n%s", strings.Join(msg.args, " "))
		m.step = reqStepStories
	}

	if m.step == reqStepEdit {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *reqModel) visibleLines() int {
	// compact header = 2 lines, title+sep+blank = 3, footer = 2
	v := m.height - 7
	if v < 5 {
		v = 5
	}
	return v
}

// compactHeader returns a single slim header bar used after the logo animation.
func (m *reqModel) compactHeader() string {
	t := m.theme
	left := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  maple")
	mid := lipgloss.NewStyle().Foreground(t.Muted).Render(" · requirements")
	if m.selectedAI.label != "" {
		badge := lipgloss.NewStyle().
			Foreground(t.Background).Background(t.Primary).
			Padding(0, 1).Render(m.selectedAI.label)
		mid += "  " + badge
	}
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + strings.Repeat("─", 60))
	return left + mid + "\n" + sep + "\n"
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m *reqModel) View() string {
	t := m.theme
	var hdr string
	if m.logoDone {
		hdr = m.compactHeader()
	} else {
		hdr = logoAnimFrame(m.logoFrame)
	}

	cursor := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("❯")

	switch m.step {
	case reqStepPickAI:
		return hdr + m.aiPickerView(t, cursor)

	case reqStepEdit:
		return hdr + m.editView(t)

	case reqStepSending:
		return hdr + m.sendingView(t)

	case reqStepStories:
		if m.err != nil {
			return hdr + m.errorView(t)
		}
		return hdr + m.storiesView(t, cursor)

	case reqStepViewStory:
		return hdr + m.storyDetailView(t)

	case reqStepPickImplementationHarness:
		return hdr + m.implementationHarnessPickerView(t, cursor)
	}
	return ""
}

func (m *reqModel) aiPickerView(t Theme, cursor string) string {
	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Select harness")
	sb.WriteString("  " + title + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("  "+strings.Repeat("─", 44)) + "\n\n")
	for i, opt := range m.aiOptions {
		kind := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + opt.kind)
		if i == m.aiCursor {
			label := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(fmt.Sprintf("%-22s", opt.label))
			sb.WriteString("  " + cursor + " " + label + kind + "\n")
		} else {
			label := lipgloss.NewStyle().Foreground(t.Foreground).Render(fmt.Sprintf("%-22s", opt.label))
			sb.WriteString("     " + label + kind + "\n")
		}
	}
	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("  j/k Navigate · Enter Select · q Quit") + "\n")
	return sb.String()
}

func (m *reqModel) editView(t Theme) string {
	charCount := lipgloss.NewStyle().Foreground(t.Muted).
		Render(fmt.Sprintf("  %d chars", len(m.textarea.Value())))
	help := lipgloss.NewStyle().Foreground(t.Muted).Render(
		"  Describe a story or a whole epic.\n" +
			"  Ctrl+D to convert → Gherkin   Esc to quit")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(0, 1).
		Render(m.textarea.View())
	return charCount + "\n\n" + help + "\n\n" + box + "\n"
}

func (m *reqModel) sendingView(t Theme) string {
	elapsed := time.Since(m.sendingStart).Round(time.Second)
	aiTag := lipgloss.NewStyle().
		Foreground(t.Background).Background(t.Primary).
		Padding(0, 1).
		Render(m.selectedAI.label)
	line1 := "  " + aiTag
	line2 := "  " + m.spinner.View() +
		lipgloss.NewStyle().Foreground(t.Accent).Render(" Converting to Gherkin…") +
		"  " + lipgloss.NewStyle().Foreground(t.Muted).Render(elapsed.String())
	hint := lipgloss.NewStyle().Foreground(t.Muted).Render("  This may take 10–30 seconds.  Esc to cancel.")
	return line1 + "\n\n" + line2 + "\n" + hint + "\n"
}

func (m *reqModel) errorView(t Theme) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Error).
		Padding(0, 1).
		Width(m.width - 6).
		Render(lipgloss.NewStyle().Foreground(t.Error).Render(m.err.Error()))
	hint := lipgloss.NewStyle().Foreground(t.Muted).Render("  e edit again   q quit")
	return "\n" + box + "\n\n" + hint + "\n"
}

func (m *reqModel) storiesView(t Theme, cursor string) string {
	var sb strings.Builder
	count := len(m.stories)
	noun := "story"
	if count != 1 {
		noun = "stories"
	}
	elapsed := time.Since(m.sendingStart).Round(time.Second)
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).
		Render(fmt.Sprintf("  %d %s generated", count, noun))
	timing := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  in %s", elapsed))
	sb.WriteString(title + timing + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("  "+strings.Repeat("─", 60)) + "\n\n")

	// compact header(2) + title(1) + sep(1) + blank(1) = row 5
	m.storyListTop = 5

	for i, s := range m.stories {
		var savedNote string
		if s.savedTo != "" {
			savedNote = "  " + lipgloss.NewStyle().Foreground(t.Success).Render("✓ "+s.savedTo)
		}
		if i == m.storyCursor {
			label := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(s.title)
			enterHint := lipgloss.NewStyle().Foreground(t.Muted).Render("  ↵")
			sb.WriteString("  " + cursor + " " + label + savedNote + enterHint + "\n")
		} else {
			label := lipgloss.NewStyle().Foreground(t.Foreground).Render(s.title)
			sb.WriteString("     " + label + savedNote + "\n")
		}
	}

	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(t.Muted).Render(
		"  j/k Navigate · Enter/Click View · e Edit · R Regenerate · i Implement via TAFFY · q Quit") + "\n")
	return sb.String()
}

func (m *reqModel) storyDetailView(t Theme) string {
	story := m.stories[m.viewingIdx]
	total := len(m.stories)
	var sb strings.Builder

	// Title + nav indicator
	nav := lipgloss.NewStyle().Foreground(t.Muted).
		Render(fmt.Sprintf("  story %d/%d  [/] next  h prev", m.viewingIdx+1, total))
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  " + story.title)
	sb.WriteString(title + "\n" + nav + "\n")
	if story.savedTo != "" {
		saved := lipgloss.NewStyle().Foreground(t.Success).Render("  ✓ saved → " + story.savedTo)
		sb.WriteString(saved + "\n")
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("  "+strings.Repeat("─", 60)) + "\n\n")

	// Scrollable Gherkin with syntax highlighting
	lines := strings.Split(story.gherkin, "\n")
	visible := m.visibleLines()
	end := m.scrollOffset + visible
	if end > len(lines) {
		end = len(lines)
	}
	window := lines[m.scrollOffset:end]

	for _, l := range window {
		sb.WriteString("  " + colorizeGherkin(l, t) + "\n")
	}

	// Scroll indicator
	lineCount := len(lines)
	if lineCount > visible {
		pct := (m.scrollOffset * 100) / (lineCount - visible)
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(t.Muted).
			Render(fmt.Sprintf("  (%d%%)  j/k Scroll · gg/G top/bottom · i Implement · b/Esc Back · q Quit", pct)) + "\n")
	} else {
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(t.Muted).
			Render("  j/k Scroll · i Implement · b/Esc Back · q Quit") + "\n")
	}
	return sb.String()
}

func (m *reqModel) implementationHarnessPickerView(t Theme, cursor string) string {
	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Implement generated stories")
	sb.WriteString("  " + title + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("  Select harness for /pipeline-runner implement-stories") + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("  "+strings.Repeat("─", 68)) + "\n\n")
	for i, opt := range m.aiOptions {
		kind := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + opt.kind)
		if i == m.implCursor {
			label := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(fmt.Sprintf("%-22s", opt.label))
			sb.WriteString("  " + cursor + " " + label + kind + "\n")
		} else {
			label := lipgloss.NewStyle().Foreground(t.Foreground).Render(fmt.Sprintf("%-22s", opt.label))
			sb.WriteString("     " + label + kind + "\n")
		}
	}
	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("  j/k Navigate · Enter Launch · Esc Back · q Quit") + "\n")
	return sb.String()
}

// colorizeGherkin applies theme colours to a single Gherkin line.
func colorizeGherkin(line string, t Theme) string {
	trimmed := strings.TrimSpace(line)
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	switch {
	case strings.HasPrefix(trimmed, "Feature:"):
		kw := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Feature:")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Render(trimmed[len("Feature:"):])
		return indent + kw + rest
	case strings.HasPrefix(trimmed, "Background:"):
		return indent + lipgloss.NewStyle().Foreground(t.Warning).Bold(true).Render(trimmed)
	case strings.HasPrefix(trimmed, "Scenario Outline:"):
		kw := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Scenario Outline:")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed[len("Scenario Outline:"):])
		return indent + kw + rest
	case strings.HasPrefix(trimmed, "Scenario:"):
		kw := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Scenario:")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed[len("Scenario:"):])
		return indent + kw + rest
	case strings.HasPrefix(trimmed, "Given "):
		kw := lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("Given")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed[5:])
		return indent + kw + rest
	case strings.HasPrefix(trimmed, "When "):
		kw := lipgloss.NewStyle().Foreground(t.Warning).Bold(true).Render("When")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed[4:])
		return indent + kw + rest
	case strings.HasPrefix(trimmed, "Then "):
		kw := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Then")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed[4:])
		return indent + kw + rest
	case strings.HasPrefix(trimmed, "And "):
		kw := lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render("And")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed[3:])
		return indent + kw + rest
	case strings.HasPrefix(trimmed, "But "):
		kw := lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render("But")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed[3:])
		return indent + kw + rest
	case strings.HasPrefix(trimmed, "Examples:"):
		return indent + lipgloss.NewStyle().Foreground(t.Warning).Render(trimmed)
	case strings.HasPrefix(trimmed, "|"):
		// Table rows — mute the pipes, normal text
		return indent + lipgloss.NewStyle().Foreground(t.Muted).Render("|") +
			lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed[1:])
	case strings.HasPrefix(trimmed, "@"):
		return indent + lipgloss.NewStyle().Foreground(t.Accent).Render(trimmed)
	case strings.HasPrefix(trimmed, "#"):
		return indent + lipgloss.NewStyle().Foreground(t.Muted).Render(trimmed)
	default:
		return indent + lipgloss.NewStyle().Foreground(t.Foreground).Render(trimmed)
	}
}

// ─── AI conversion ────────────────────────────────────────────────────────────

const gherkinPrompt = `You are a Gherkin authoring expert. Analyze the requirements below.

Determine if this describes:
  A) A single user story → output ONE story block
  B) An epic (multiple related stories) → split into separate stories, one block each

For EACH story use this exact format:
=== STORY: <Concise Story Title> ===
Feature: <feature name>

  @story @priority:medium
  Scenario: <happy path name>
    Given ...
    When ...
    Then ...

  Scenario: <edge case name>
    Given ...
    When ...
    Then ...

Rules:
- Use Feature, Background (if needed), Scenario, Given/When/Then/And/But
- Plain language that non-technical stakeholders can understand
- Separate scenarios for happy path and key edge cases
- Preserve concrete technical requirements from the input (frameworks, storage, APIs, design systems, visual style) as explicit, testable behavior in Background or Scenario steps
- If requirements describe a UI, SPA, frontend, design/look-and-feel, set assumptions and scenarios that clearly indicate a visual feature
- Output ONLY the === STORY: === blocks — no explanation, no prose
- Do not output markdown fences, token usage, change summaries, request stats, or any telemetry

Requirements:
%s`

func convertToGherkin(ctx context.Context, requirements string, ai aiOption) (string, error) {
	prompt := fmt.Sprintf(gherkinPrompt, requirements)
	out, err := invokeAI(ctx, ai, prompt)
	if err != nil {
		return "", fmt.Errorf("%s: %w", ai.label, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (m *reqModel) defaultHarnessCursor() int {
	for i, opt := range m.aiOptions {
		if opt.kind == m.selectedAI.kind {
			return i
		}
	}
	return 0
}

func reqTaffyPathForHarness(kind string) string {
	switch kind {
	case "opencode":
		return ".opencode/taffy/implement-stories.yaml"
	case "cursor":
		return ".cursor/taffy/implement-stories.yaml"
	default:
		return ".claude/taffy/implement-stories.yaml"
	}
}

func writeReqImplementationHandoff(stories []gherkinStory) error {
	type handoffStory struct {
		Title string `json:"title"`
		Path  string `json:"path"`
		UI    bool   `json:"ui"`
	}
	var out []handoffStory
	for _, s := range stories {
		if strings.TrimSpace(s.savedTo) == "" {
			continue
		}
		out = append(out, handoffStory{Title: s.title, Path: s.savedTo, UI: s.ui})
	}
	_ = os.MkdirAll(".claude/state", 0o755)
	data, err := json.MarshalIndent(map[string]interface{}{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"stories":      out,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(".claude/state/gherkin-handoff.json", append(data, '\n'), 0o644)
}

func (m *reqModel) buildImplementationPrompt(harness string) string {
	var paths []string
	for _, s := range m.stories {
		if strings.TrimSpace(s.savedTo) != "" {
			paths = append(paths, s.savedTo)
		}
	}
	var sb strings.Builder
	sb.WriteString("/pipeline-runner implement-stories")
	sb.WriteString(governanceBootstrapBlock(harness))
	sb.WriteString("\n\n<maple-gherkin-handoff>\n")
	sb.WriteString("These stories were generated and approved via `maple req`.\n")
	sb.WriteString("Implement and test only these story directories:\n")
	for _, p := range paths {
		sb.WriteString("- " + p + "\n")
	}
	sb.WriteString("Use `.claude/state/gherkin-handoff.json` as the source of truth for targets.\n")
	sb.WriteString("Do not regenerate or rewrite stories. Do not rerun spec-kit.\n")
	sb.WriteString("Run orchestrator + delegation to execute implementation and tests for these stories.\n")
	sb.WriteString("Enforce the full 8-phase pipeline and Karpathy gate.\n")
	sb.WriteString("</maple-gherkin-handoff>\n")
	sb.WriteString("\n<maple-governance>\n")
	sb.WriteString("Hard requirements:\n")
	sb.WriteString("- Strictly follow repository instruction files: harness root markdown (`CLAUDE.md`/`OPENCODE.md`/`CURSOR.md`/`COPILOT.md`), `.github/copilot-instructions.md`, and `.github/instructions/stories.instructions.md`.\n")
	sb.WriteString("- Preserve the BusinessRepo model and required repository layout defined by those files.\n")
	sb.WriteString("- Never place acceptance tests in `/app`; write them under `/tests` and `/tests/features`.\n")
	sb.WriteString("- Runtime code and tests must never import from `docs/`, `.github/`, or `.claude/` paths.\n")
	sb.WriteString("- Copying/adapting approved artifacts into app/test source is allowed; direct path imports/references to docs are not.\n")
	sb.WriteString("- Respect the existing Cucumber stack already present in the generated stories.\n")
	sb.WriteString("- If a story has `cucumber/*_steps.py`, continue with Python behave-style steps and do NOT switch to TypeScript `@cucumber/cucumber`.\n")
	sb.WriteString("- Do not invent alternative test frameworks when story artifacts already define one.\n")
	sb.WriteString("</maple-governance>\n")
	sb.WriteString(`
<maple-pipeline>
You were launched from MAPLE req.
Keep .claude/state/maple.json updated as you work by writing (merge, never overwrite other keys):
  {"taffy":"pipeline-runner implement-stories","stage":"<current step>","status":"RUNNING","updated_at":"<ISO-8601 timestamp>"}
Set status to "DONE" when finished, "FAILED" if you cannot complete.
</maple-pipeline>

<maple-progress>
Never go silent during this TAFFY implementation run:
- Post an immediate kickoff update before the first long-running tool/agent call.
- Post a concise progress heartbeat every 60-120 seconds while actively running stages.
- On each heartbeat, refresh .claude/state/maple.json updated_at and current stage.
- Every heartbeat must include concrete progress evidence:
  - changed files/artifacts since last update (explicit paths), or
  - a specific blocker that prevented changes.
- Use this heartbeat format:
  Progress: <stage name / phase>
  Done since last update: <brief>
  Current action: <brief>
  Blockers: <none or brief blocker>
  Next update: <ETA>
- Do not send heartbeat-only timestamp churn with no artifact/blocker details.
- If the stage requires writing artifacts and write access/tools are unavailable, set status FAILED with a clear error and stop.
- If blocked/waiting, state exactly what is pending and keep posting heartbeats.
- For design-review stages, continuously produce previewable artifacts (.excalidraw/.html/.svg/.png/.jpg/.md) and keep .claude/state/design-artifacts.json updated so the review portal reflects progress live.
- Before reporting DONE, verify and report concrete artifact paths for: app/domain changes, tests changes, and tests/features + step files.
- If required test or gherkin artifacts are missing, set status FAILED and report missing paths.
- If generated runtime code imports docs/, .github/, or .claude/ files by path, set status FAILED and report offending import paths.
</maple-progress>`)
	if u := activeDesignPortalURL(); u != "" {
		sb.WriteString("\n<maple-design-portal>\n")
		sb.WriteString("The MAPLE design review portal is running at: " + u + "\n")
		sb.WriteString("Browse docs/design/ artifacts there to reference approved wireframes, mockups, and identity tokens.\n")
		sb.WriteString("Wireframes are at: " + u + "/wireframes/\n")
		sb.WriteString("</maple-design-portal>\n")
	}
	return sb.String()
}

func (m *reqModel) launchImplementationCmd(ai aiOption) tea.Cmd {
	return func() tea.Msg {
		if len(m.stories) == 0 {
			return reqDoneMsg{err: fmt.Errorf("no generated stories to implement")}
		}
		if _, err := os.Stat(reqTaffyPathForHarness(ai.kind)); err != nil {
			return reqDoneMsg{err: fmt.Errorf("missing implement-stories workflow for %s. run `maple update` and try again", ai.label)}
		}
		if err := writeReqImplementationHandoff(m.stories); err != nil {
			return reqDoneMsg{err: fmt.Errorf("failed to write gherkin handoff: %w", err)}
		}
		writeQuickLaunchState("pipeline-runner implement-stories", "starting", ai.kind)
		cmd := m.buildImplementationPrompt(ai.kind)
		args := buildLaunchCmd(ai.kind, cmd, loadPinnedSessions(), true)
		msg := trySpawnCmdForHarness(ai.kind, args)
		return msg()
	}
}

// invokeAI runs the selected AI tool and returns its stdout output.
// The prompt is passed as a positional argument (not stdin) to avoid inheriting
// the Bubble Tea raw-mode terminal state in the subprocess.
func invokeAI(ctx context.Context, ai aiOption, prompt string) ([]byte, error) {
	var (
		out []byte
		err error
	)
	switch ai.kind {
	case "claude":
		// -p = --print (non-interactive), prompt is positional arg.
		// --output-format text suppresses JSON wrappers.
		// --no-session-persistence avoids writing session files.
		out, err = runCtx(ctx, timeoutAI, nil, "", ai.path, "-p", "--output-format", "text", "--no-session-persistence", prompt)
	case "copilot":
		out, err = runCtx(ctx, timeoutAI, nil, "", ai.path, "-i", prompt)
	case "opencode":
		out, err = runCtx(ctx, timeoutAI, nil, prompt, ai.path, "run")
	case "cursor":
		return invokeCursor(ctx, ai.path, prompt)
	default:
		return nil, fmt.Errorf("unsupported AI tool: %s", ai.kind)
	}

	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return out, nil
}

func invokeCursor(ctx context.Context, binPath, prompt string) ([]byte, error) {
	type attempt struct {
		args  []string
		stdin string
	}
	candidates := []attempt{
		{args: []string{"-p", "--output-format", "text", "--trust", prompt}},
		{args: []string{"-p", prompt}},
		{args: []string{"--prompt", prompt}},
		{args: []string{"run", prompt}},
		{args: []string{"run"}, stdin: prompt},
	}

	var lastErr string
	for _, c := range candidates {
		out, err := runCtx(ctx, timeoutAI, nil, c.stdin, binPath, c.args...)
		if err == nil {
			return out, nil
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		lastErr = msg
	}
	if lastErr == "" {
		lastErr = "cursor-agent invocation failed"
	}
	return nil, fmt.Errorf("%s", lastErr)
}

func (m *reqModel) convert(requirements string) tea.Cmd {
	ai := m.selectedAI
	// snapshot old saved dirs before the async call so we can clean them up
	var oldDirs []string
	for _, s := range m.stories {
		if s.savedTo != "" {
			oldDirs = append(oldDirs, s.savedTo)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelSend = cancel
	return func() tea.Msg {
		defer cancel()
		// remove old story directories before saving new ones
		for _, d := range oldDirs {
			_ = os.RemoveAll(d)
		}
		gherkin, err := convertToGherkin(ctx, requirements, ai)
		if err != nil {
			return reqDoneMsg{err: err}
		}
		stories := parseStories(gherkin)
		for i := range stories {
			stories[i].ui = inferUIStory(requirements, stories[i].title, stories[i].gherkin)
			savedTo, saveErr := saveStory(stories[i].title, stories[i].gherkin, stories[i].ui, i+1)
			if saveErr == nil {
				stories[i].savedTo = savedTo
			}
		}
		return reqDoneMsg{stories: stories}
	}
}

// ─── Story parsing ────────────────────────────────────────────────────────────

var storyHeaderRe = regexp.MustCompile(`(?m)^=== STORY: (.+?) ===$`)
var telemetryLineRe = regexp.MustCompile(`^(Changes|Requests|Tokens)\b`)

// parseStories splits AI output into individual gherkinStory values.
// If no === STORY: === delimiters are found the whole output is treated as one story.
func parseStories(output string) []gherkinStory {
	output = sanitizeModelOutput(output)
	matches := storyHeaderRe.FindAllStringIndex(output, -1)
	if len(matches) == 0 {
		cleaned := cleanGherkin(output)
		return []gherkinStory{{
			title:   extractFeatureTitle(cleaned),
			gherkin: cleaned,
		}}
	}

	var stories []gherkinStory
	for i, match := range matches {
		titleMatch := storyHeaderRe.FindStringSubmatch(output[match[0]:match[1]])
		title := strings.TrimSpace(titleMatch[1])

		start := match[1]
		if start < len(output) && output[start] == '\n' {
			start++
		}
		var content string
		if i+1 < len(matches) {
			content = output[start:matches[i+1][0]]
		} else {
			content = output[start:]
		}
		cleaned := cleanGherkin(content)
		stories = append(stories, gherkinStory{
			title:   title,
			gherkin: cleaned,
		})
	}
	return stories
}

func sanitizeModelOutput(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var out []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "```" || trimmed == "```gherkin" {
			continue
		}
		if telemetryLineRe.MatchString(trimmed) {
			continue
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// extractFeatureTitle pulls the Feature: name from gherkin text, or returns "Untitled".
func extractFeatureTitle(s string) string {
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "Feature:") {
			t := strings.TrimSpace(strings.TrimPrefix(l, "Feature:"))
			if t != "" {
				return t
			}
		}
	}
	return "Untitled"
}

func stripFences(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if (i == 0 && (trimmed == "```gherkin" || trimmed == "```")) || trimmed == "```" || trimmed == "```gherkin" {
			continue
		}
		if telemetryLineRe.MatchString(trimmed) {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func cleanGherkin(s string) string {
	trimmed := strings.TrimSpace(stripFences(s))
	lines := strings.Split(trimmed, "\n")
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if telemetryLineRe.MatchString(t) {
			continue
		}
		if strings.HasPrefix(t, "```") {
			continue
		}
		out = append(out, l)
	}
	cleaned := strings.TrimSpace(strings.Join(out, "\n"))
	if idx := strings.Index(cleaned, "Feature:"); idx > 0 {
		cleaned = strings.TrimSpace(cleaned[idx:])
	}
	return cleaned
}

func inferUIStory(requirements, title, gherkin string) bool {
	corpus := strings.ToLower(requirements + "\n" + title + "\n" + gherkin)
	for _, k := range []string{
		" ui ", "spa", "frontend", "front-end", "react", "vue", "angular",
		"screen", "button", "form", "layout", "look and feel", "design",
		"visual", "glass", "apple", "theme", "css", "tailwind",
	} {
		if strings.Contains(corpus, k) {
			return true
		}
	}
	return false
}

// ─── Story file writer ────────────────────────────────────────────────────────

// saveStory creates the full story directory structure:
//
//	docs/stories/<slug>-<ts>-<idx:04d>/
//	  Story.md                  ← metadata + narrative + embedded Gherkin
//	  cucumber/
//	    <slug>.feature          ← pure Gherkin (extracted, for test runners)
//	    <slug>_steps.py         ← behave step stubs (generated once, never overwritten)
func saveStory(title, gherkin string, ui bool, idx int) (string, error) {
	slug := slugify(title)
	ts := time.Now().Format("20060102150405")
	storyDir := filepath.Clean(fmt.Sprintf("docs/stories/%s-%s-%04d", slug, ts, idx))
	cucumberDir := filepath.Join(storyDir, "cucumber")

	if err := os.MkdirAll(cucumberDir, 0755); err != nil {
		return "", err
	}

	displayTitle := title
	if len(displayTitle) > 80 {
		displayTitle = displayTitle[:80]
	}

	// ── Story.md ─────────────────────────────────────────────────────────────
	storyContent := fmt.Sprintf(`---
id: "%s-%04d"
title: "%s"
epic: "%s"
priority: "medium"
ui: %t
adr_required: false
milestone: null
labels:
  - "type:feature"
  - "priority:medium"
  - "phase:discover"
issue_number: null
issue_url: null
created_at: "%s"
---

# %s

## Scenarios

`+"```gherkin\n%s\n```"+`

## Definition of Done

- [ ] Unit tests green
- [ ] Integration tests green
- [ ] Cucumber scenarios green
- [ ] ADRs linked where required

## ADR Links

(populated by adr-author agent)
`,
		slug, idx, displayTitle, slug, ui, time.Now().Format(time.RFC3339),
		displayTitle, gherkin,
	)

	storyPath := filepath.Join(storyDir, "Story.md")
	if err := os.WriteFile(storyPath, []byte(storyContent), 0644); err != nil {
		return storyDir, err
	}

	// ── cucumber/<slug>.feature ───────────────────────────────────────────────
	featurePath := filepath.Join(cucumberDir, slug+".feature")
	if err := os.WriteFile(featurePath, []byte(gherkin+"\n"), 0644); err != nil {
		return storyDir, err
	}

	// ── cucumber/<slug>_steps.py  (generated once, never overwritten) ─────────
	stepsPath := filepath.Join(cucumberDir, slug+"_steps.py")
	if _, err := os.Stat(stepsPath); os.IsNotExist(err) {
		steps := extractSteps(gherkin)
		if len(steps) > 0 {
			_ = os.WriteFile(stepsPath, []byte(generateStepDefs(displayTitle, steps)), 0644)
		}
	}

	return storyDir, nil
}

// ─── Gherkin step extraction & stub generation ────────────────────────────────

type stepLine struct {
	keyword string // given | when | then
	text    string
}

var stepLineRe = regexp.MustCompile(`^\s+(Given|When|Then|And|But)\s+(.+)$`)

// extractSteps parses unique Given/When/Then steps from a Gherkin block.
// And/But steps are normalised to the last main keyword.
func extractSteps(gherkin string) []stepLine {
	var steps []stepLine
	seen := map[string]bool{}
	last := "given"
	for _, line := range strings.Split(gherkin, "\n") {
		m := stepLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		kw, text := m[1], strings.TrimSpace(m[2])
		var decorator string
		switch kw {
		case "And", "But":
			decorator = last
		default:
			decorator = strings.ToLower(kw)
			last = decorator
		}
		key := decorator + "|" + text
		if !seen[key] {
			seen[key] = true
			steps = append(steps, stepLine{keyword: decorator, text: text})
		}
	}
	return steps
}

// generateStepDefs produces a behave step definitions skeleton.
func generateStepDefs(featureTitle string, steps []stepLine) string {
	var sb strings.Builder
	sb.WriteString("# Step definitions for: " + featureTitle + "\n")
	sb.WriteString("# Framework: behave  https://behave.readthedocs.io\n")
	sb.WriteString("# Each stub raises NotImplementedError until implemented.\n")
	sb.WriteString("from behave import given, when, then  # noqa: F401\n\n\n")
	for _, s := range steps {
		fn := stepFuncName(s.text)
		sb.WriteString(fmt.Sprintf("@%s(u%q)\n", s.keyword, s.text))
		sb.WriteString(fmt.Sprintf("def %s(context):\n", fn))
		sb.WriteString(fmt.Sprintf("    raise NotImplementedError(u\"STEP: %s %s\")\n\n\n", s.keyword, s.text))
	}
	return sb.String()
}

func stepFuncName(text string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s := strings.ToLower(text)
	s = re.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	runes := []rune(s)
	if len(runes) > 60 {
		runes = runes[:60]
		for len(runes) > 0 && runes[len(runes)-1] == '_' {
			runes = runes[:len(runes)-1]
		}
	}
	return "step_" + string(runes)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	runes := []rune(s)
	if len(runes) > 40 {
		runes = runes[:40]
		for len(runes) > 0 && !unicode.IsLetter(runes[len(runes)-1]) && !unicode.IsDigit(runes[len(runes)-1]) {
			runes = runes[:len(runes)-1]
		}
	}
	return string(runes)
}
