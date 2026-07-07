package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

// colorizeStory syntax-highlights the lines of a Story.md (markdown headings, task
// checkboxes, code fences, and embedded Gherkin) so the story detail overlay reads like
// the main-branch story view. Escapes are embedded per line; the detail pane renders
// non-selectable rows verbatim, so the colors show through.
func colorizeStory(lines []string, m theme.Mode) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = colorizeStoryLine(l, m)
	}
	return out
}

func colorizeStoryLine(line string, m theme.Mode) string {
	trimmed := strings.TrimSpace(line)
	title := m.Role("title").Style()
	accent := m.Role("accent").Style()
	faint := m.Role("faint").Style()
	base := m.Role("base").Style()
	warn := m.State("running").Style()  // yellow-ish
	ok := m.State("done").Style()       // green-ish

	switch {
	case strings.HasPrefix(trimmed, "# "):
		return title.Bold(true).Render(line)
	case strings.HasPrefix(trimmed, "## "):
		return accent.Bold(true).Render(line)
	case strings.HasPrefix(trimmed, "### "):
		return warn.Render(line)
	case strings.HasPrefix(trimmed, "- [ ]"):
		i := strings.Index(line, "- [ ]")
		return faint.Render("- [ ]") + base.Render(line[i+5:])
	case strings.HasPrefix(trimmed, "- [x]"), strings.HasPrefix(trimmed, "- [X]"):
		i := strings.Index(line, "- [")
		return ok.Render("- [x]") + faint.Render(line[i+5:])
	case strings.HasPrefix(trimmed, "```"):
		return faint.Render(line)
	case strings.HasPrefix(trimmed, "Feature:"), strings.HasPrefix(trimmed, "Background:"),
		strings.HasPrefix(trimmed, "Scenario:"), strings.HasPrefix(trimmed, "Scenario Outline:"),
		strings.HasPrefix(trimmed, "Given "), strings.HasPrefix(trimmed, "When "),
		strings.HasPrefix(trimmed, "Then "), strings.HasPrefix(trimmed, "And "),
		strings.HasPrefix(trimmed, "But "), strings.HasPrefix(trimmed, "Examples:"),
		strings.HasPrefix(trimmed, "@"), strings.HasPrefix(trimmed, "|"):
		return colorizeGherkin(line, m)
	default:
		return base.Render(line)
	}
}

// colorizeGherkin colors a single Gherkin line, keyword then remainder.
func colorizeGherkin(line string, m theme.Mode) string {
	trimmed := strings.TrimSpace(line)
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	title := m.Role("title").Style()
	accent := m.Role("accent").Style()
	faint := m.Role("faint").Style()
	base := m.Role("base").Style()
	warn := m.State("running").Style()
	ok := m.State("done").Style()

	kw := func(s lipgloss.Style, word, rest string) string {
		return indent + s.Bold(true).Render(word) + base.Render(rest)
	}
	switch {
	case strings.HasPrefix(trimmed, "Feature:"):
		return indent + title.Bold(true).Render("Feature:") + base.Bold(true).Render(trimmed[len("Feature:"):])
	case strings.HasPrefix(trimmed, "Background:"):
		return indent + warn.Render(trimmed)
	case strings.HasPrefix(trimmed, "Scenario Outline:"):
		return kw(accent, "Scenario Outline:", trimmed[len("Scenario Outline:"):])
	case strings.HasPrefix(trimmed, "Scenario:"):
		return kw(accent, "Scenario:", trimmed[len("Scenario:"):])
	case strings.HasPrefix(trimmed, "Given "):
		return kw(ok, "Given", trimmed[5:])
	case strings.HasPrefix(trimmed, "When "):
		return kw(warn, "When", trimmed[4:])
	case strings.HasPrefix(trimmed, "Then "):
		return kw(title, "Then", trimmed[4:])
	case strings.HasPrefix(trimmed, "And "):
		return kw(faint, "And", trimmed[3:])
	case strings.HasPrefix(trimmed, "But "):
		return kw(faint, "But", trimmed[3:])
	case strings.HasPrefix(trimmed, "Examples:"):
		return indent + warn.Render(trimmed)
	case strings.HasPrefix(trimmed, "@"):
		return indent + accent.Render(trimmed)
	case strings.HasPrefix(trimmed, "|"):
		return indent + faint.Render(trimmed)
	default:
		return base.Render(line)
	}
}
