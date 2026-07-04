package req

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) View() string {
	p := m.pal
	hdr := m.compactHeader()
	cursor := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render("❯")

	switch m.step {
	case stepPickAI:
		return hdr + m.aiPickerView(cursor)
	case stepEdit:
		return hdr + m.editView()
	case stepSending:
		return hdr + m.sendingView()
	case stepStories:
		if m.err != nil {
			return hdr + m.errorView()
		}
		return hdr + m.storiesView(cursor)
	case stepViewStory:
		return hdr + m.storyDetailView()
	case stepPickImpl:
		return hdr + m.implPickerView(cursor)
	}
	return ""
}

func (m *model) compactHeader() string {
	p := m.pal
	left := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render("  maple")
	mid := lipgloss.NewStyle().Foreground(p.Muted).Render(" · requirements")
	if m.selectedAI.Label != "" {
		badge := lipgloss.NewStyle().
			Foreground(p.Background).Background(p.Primary).
			Padding(0, 1).Render(m.selectedAI.Label)
		mid += "  " + badge
	}
	sep := lipgloss.NewStyle().Foreground(p.Muted).Render("  " + strings.Repeat("─", 60))
	return left + mid + "\n" + sep + "\n"
}

func (m *model) aiPickerView(cursor string) string {
	p := m.pal
	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render("Select harness")
	sb.WriteString("  " + title + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  "+strings.Repeat("─", 44)) + "\n\n")
	for i, opt := range m.tools {
		kind := lipgloss.NewStyle().Foreground(p.Muted).Render("  " + opt.Kind)
		if i == m.aiCursor {
			label := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render(fmt.Sprintf("%-22s", opt.Label))
			sb.WriteString("  " + cursor + " " + label + kind + "\n")
		} else {
			label := lipgloss.NewStyle().Foreground(p.Foreground).Render(fmt.Sprintf("%-22s", opt.Label))
			sb.WriteString("     " + label + kind + "\n")
		}
	}
	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(p.Muted).Render("  j/k Navigate · Enter Select · q Quit") + "\n")
	return sb.String()
}

func (m *model) editView() string {
	p := m.pal
	charCount := lipgloss.NewStyle().Foreground(p.Muted).
		Render(fmt.Sprintf("  %d chars", len(m.textarea.Value())))
	help := lipgloss.NewStyle().Foreground(p.Muted).Render(
		"  Describe a story or a whole epic.\n" +
			"  Ctrl+D to convert → Gherkin   Esc to quit")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Primary).
		Padding(0, 1).
		Render(m.textarea.View())
	return charCount + "\n\n" + help + "\n\n" + box + "\n"
}

func (m *model) sendingView() string {
	p := m.pal
	elapsed := time.Since(m.sendingStart).Round(time.Second)
	aiTag := lipgloss.NewStyle().
		Foreground(p.Background).Background(p.Primary).
		Padding(0, 1).
		Render(m.selectedAI.Label)
	line1 := "  " + aiTag
	line2 := "  " + m.spinner.View() +
		lipgloss.NewStyle().Foreground(p.Accent).Render(" Converting to Gherkin…") +
		"  " + lipgloss.NewStyle().Foreground(p.Muted).Render(elapsed.String())
	hint := lipgloss.NewStyle().Foreground(p.Muted).Render("  This may take 10–30 seconds.  Esc to cancel.")
	return line1 + "\n\n" + line2 + "\n" + hint + "\n"
}

func (m *model) errorView() string {
	p := m.pal
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Error).
		Padding(0, 1).
		Width(m.width - 6).
		Render(lipgloss.NewStyle().Foreground(p.Error).Render(m.err.Error()))
	hint := lipgloss.NewStyle().Foreground(p.Muted).Render("  e edit again   q quit")
	return "\n" + box + "\n\n" + hint + "\n"
}

func (m *model) storiesView(cursor string) string {
	p := m.pal
	var sb strings.Builder
	count := len(m.stories)
	noun := "story"
	if count != 1 {
		noun = "stories"
	}
	elapsed := time.Since(m.sendingStart).Round(time.Second)
	title := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).
		Render(fmt.Sprintf("  %d %s generated", count, noun))
	timing := lipgloss.NewStyle().Foreground(p.Muted).Render(fmt.Sprintf("  in %s", elapsed))
	sb.WriteString(title + timing + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  "+strings.Repeat("─", 60)) + "\n\n")

	m.storyListTop = 5

	for i, s := range m.stories {
		var savedNote string
		if s.SavedTo != "" {
			savedNote = "  " + lipgloss.NewStyle().Foreground(p.Success).Render("✓ "+s.SavedTo)
		}
		if i == m.storyCursor {
			label := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render(s.Title)
			enterHint := lipgloss.NewStyle().Foreground(p.Muted).Render("  ↵")
			sb.WriteString("  " + cursor + " " + label + savedNote + enterHint + "\n")
		} else {
			label := lipgloss.NewStyle().Foreground(p.Foreground).Render(s.Title)
			sb.WriteString("     " + label + savedNote + "\n")
		}
	}

	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(p.Muted).Render(
		"  j/k Navigate · Enter/Click View · e Edit · R Regenerate · i Implement via TAFFY · q Quit") + "\n")
	return sb.String()
}

func (m *model) storyDetailView() string {
	p := m.pal
	story := m.stories[m.viewingIdx]
	total := len(m.stories)
	var sb strings.Builder

	nav := lipgloss.NewStyle().Foreground(p.Muted).
		Render(fmt.Sprintf("  story %d/%d  [/] next  h prev", m.viewingIdx+1, total))
	title := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render("  " + story.Title)
	sb.WriteString(title + "\n" + nav + "\n")
	if story.SavedTo != "" {
		saved := lipgloss.NewStyle().Foreground(p.Success).Render("  ✓ saved → " + story.SavedTo)
		sb.WriteString(saved + "\n")
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  "+strings.Repeat("─", 60)) + "\n\n")

	lines := strings.Split(story.Gherkin, "\n")
	visible := m.visibleLines()
	end := m.scrollOffset + visible
	if end > len(lines) {
		end = len(lines)
	}
	if m.scrollOffset > len(lines) {
		m.scrollOffset = 0
	}
	window := lines[m.scrollOffset:end]
	for _, l := range window {
		sb.WriteString("  " + colorizeGherkin(l, p) + "\n")
	}

	lineCount := len(lines)
	if lineCount > visible {
		pct := (m.scrollOffset * 100) / (lineCount - visible)
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(p.Muted).
			Render(fmt.Sprintf("  (%d%%)  j/k Scroll · gg/G top/bottom · i Implement · b/Esc Back · q Quit", pct)) + "\n")
	} else {
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(p.Muted).
			Render("  j/k Scroll · i Implement · b/Esc Back · q Quit") + "\n")
	}
	return sb.String()
}

func (m *model) implPickerView(cursor string) string {
	p := m.pal
	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render("Implement generated stories")
	sb.WriteString("  " + title + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  Select harness for /pipeline-runner implement-stories") + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  "+strings.Repeat("─", 68)) + "\n\n")
	for i, opt := range m.tools {
		kind := lipgloss.NewStyle().Foreground(p.Muted).Render("  " + opt.Kind)
		if i == m.implCursor {
			label := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render(fmt.Sprintf("%-22s", opt.Label))
			sb.WriteString("  " + cursor + " " + label + kind + "\n")
		} else {
			label := lipgloss.NewStyle().Foreground(p.Foreground).Render(fmt.Sprintf("%-22s", opt.Label))
			sb.WriteString("     " + label + kind + "\n")
		}
	}
	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(p.Muted).Render("  j/k Navigate · Enter Launch · Esc Back · q Quit") + "\n")
	return sb.String()
}

// colorizeGherkin applies palette colours to a single Gherkin line.
func colorizeGherkin(line string, p palette) string {
	trimmed := strings.TrimSpace(line)
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	kw := func(c lipgloss.Color, s string) string {
		return lipgloss.NewStyle().Foreground(c).Bold(true).Render(s)
	}
	rest := func(s string) string { return lipgloss.NewStyle().Foreground(p.Foreground).Render(s) }

	switch {
	case strings.HasPrefix(trimmed, "Feature:"):
		return indent + kw(p.Primary, "Feature:") + lipgloss.NewStyle().Foreground(p.Foreground).Bold(true).Render(trimmed[len("Feature:"):])
	case strings.HasPrefix(trimmed, "Background:"):
		return indent + kw(p.Warning, trimmed)
	case strings.HasPrefix(trimmed, "Scenario Outline:"):
		return indent + kw(p.Accent, "Scenario Outline:") + rest(trimmed[len("Scenario Outline:"):])
	case strings.HasPrefix(trimmed, "Scenario:"):
		return indent + kw(p.Accent, "Scenario:") + rest(trimmed[len("Scenario:"):])
	case strings.HasPrefix(trimmed, "Given "):
		return indent + kw(p.Success, "Given") + rest(trimmed[5:])
	case strings.HasPrefix(trimmed, "When "):
		return indent + kw(p.Warning, "When") + rest(trimmed[4:])
	case strings.HasPrefix(trimmed, "Then "):
		return indent + kw(p.Primary, "Then") + rest(trimmed[4:])
	case strings.HasPrefix(trimmed, "And "):
		return indent + kw(p.Muted, "And") + rest(trimmed[3:])
	case strings.HasPrefix(trimmed, "But "):
		return indent + kw(p.Muted, "But") + rest(trimmed[3:])
	case strings.HasPrefix(trimmed, "Examples:"):
		return indent + lipgloss.NewStyle().Foreground(p.Warning).Render(trimmed)
	case strings.HasPrefix(trimmed, "|"):
		return indent + lipgloss.NewStyle().Foreground(p.Muted).Render("|") + rest(trimmed[1:])
	case strings.HasPrefix(trimmed, "@"):
		return indent + lipgloss.NewStyle().Foreground(p.Accent).Render(trimmed)
	case strings.HasPrefix(trimmed, "#"):
		return indent + lipgloss.NewStyle().Foreground(p.Muted).Render(trimmed)
	default:
		return indent + rest(trimmed)
	}
}
