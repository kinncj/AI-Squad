package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// formatRateLimitCountdown renders time until resume_at as "in HH:MM:SS (3:04 PM)".
// Returns "window cleared" when now is past resume_at, "unknown" when unset/bad.
func formatRateLimitCountdown(resumeAt string, now time.Time) string {
	if resumeAt == "" {
		return "unknown"
	}
	tm, err := time.Parse(time.RFC3339, resumeAt)
	if err != nil {
		return "unknown"
	}
	if !now.Before(tm) {
		return "window cleared"
	}
	d := tm.Sub(now).Round(time.Second)
	return fmt.Sprintf("in %02d:%02d:%02d (%s)",
		int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60, tm.Local().Format("3:04 PM"))
}

// ─── Pane content renderers ───────────────────────────────────────────────────

func (m *dashboardModel) storiesContent(height int) string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Stories")
	if len(m.stories) == 0 {
		return title + "\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("no stories yet — run maple req")
	}
	lines := []string{title}
	cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
	for i, s := range m.stories {
		if i >= height-2 {
			break
		}
		phaseTag := lipgloss.NewStyle().Foreground(t.Muted).Render(truncate(s.phase, 8))
		idStr := lipgloss.NewStyle().Foreground(t.Foreground).Render(truncate(s.id, 20))
		var line string
		if i == m.storiesCur && m.focus == paneStories {
			line = cursor + " " + idStr + " " + phaseTag
		} else {
			line = "  " + idStr + " " + phaseTag
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m *dashboardModel) agentsContent(height int) string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Sessions")
	if len(m.sessions) == 0 {
		return title + "\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("no agent sessions for this project")
	}
	lines := []string{title}
	cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
	for i, s := range m.sessions {
		if i >= height-2 {
			break
		}
		badge := agentSourceBadge(s.source, t)
		sessionTitle := lipgloss.NewStyle().Foreground(t.Foreground).Render(truncate(s.title, 26))
		meta := ""
		if s.toolCount > 0 {
			meta = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" %dt", s.toolCount))
		}
		var line string
		if i == m.sessionsCur && m.focus == paneAgents {
			line = cursor + " " + badge + " " + sessionTitle + meta
		} else {
			line = "  " + badge + " " + sessionTitle + meta
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func agentSourceBadge(source string, t Theme) string {
	switch source {
	case "claude":
		return lipgloss.NewStyle().Foreground(t.Primary).Render("[cc]")
	case "opencode":
		return lipgloss.NewStyle().Foreground(t.Accent).Render("[oc]")
	case "copilot":
		return lipgloss.NewStyle().Foreground(t.Warning).Render("[gh]")
	case "maple":
		return lipgloss.NewStyle().Foreground(t.Success).Render("[ml]")
	default:
		return lipgloss.NewStyle().Foreground(t.Muted).Render("[??]")
	}
}

func (m *dashboardModel) prsContent(height int) string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("PRs")
	if m.prsLoading {
		return title + "\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("loading…")
	}
	if m.prsErr != "" {
		return title + "\n" + lipgloss.NewStyle().Foreground(t.Warning).Render(truncate(m.prsErr, 40))
	}
	if len(m.prList) == 0 {
		return title + "\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("no open PRs")
	}
	lines := []string{title}
	cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
	for i, pr := range m.prList {
		if i >= height-2 {
			break
		}
		stateCol := t.Success
		stateIcon := "✓"
		if pr.state == "OPEN" {
			stateCol = t.Accent
			stateIcon = "●"
		} else if pr.state == "CLOSED" {
			stateCol = t.Muted
			stateIcon = "✗"
		}
		num := lipgloss.NewStyle().Foreground(stateCol).Render(fmt.Sprintf("#%d %s", pr.number, stateIcon))
		ttl := lipgloss.NewStyle().Foreground(t.Foreground).Render(truncate(pr.title, 28))
		var line string
		if i == m.prsCur && m.focus == panePRs {
			line = cursor + " " + num + " " + ttl
		} else {
			line = "  " + num + " " + ttl
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m *dashboardModel) qaContent(height int) string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("QA / Tests")
	if len(m.qaEntries) == 0 {
		return title + "\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("no tests found")
	}
	summary := lipgloss.NewStyle().Foreground(t.Muted).Render(
		fmt.Sprintf("  %d test file(s)", len(m.qaEntries)))
	lines := []string{title, summary}
	cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
	for i, e := range m.qaEntries {
		if i >= height-3 {
			break
		}
		badge := testFrameworkBadge(e.framework, t)
		label := lipgloss.NewStyle().Foreground(t.Foreground).Render(truncate(filepath.Base(e.path), 24))
		extra := ""
		if e.count > 0 {
			extra = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" (%d)", e.count))
		}
		var line string
		if i == m.qaEntryCur && m.focus == paneQA {
			line = cursor + " " + badge + " " + label + extra
		} else {
			line = "  " + badge + " " + label + extra
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func testFrameworkBadge(fw string, t Theme) string {
	switch fw {
	case "gherkin":
		return lipgloss.NewStyle().Foreground(t.Success).Render("[gherkin]")
	case "go":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Render("[go]    ")
	case "jest":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#C21325")).Render("[jest]  ")
	case "vitest":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FCC72B")).Render("[vitest]")
	case "mocha":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8D6748")).Render("[mocha] ")
	case "npm":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#CB3837")).Render("[npm]   ")
	case "pytest":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#3572A5")).Render("[pytest]")
	case "unittest":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#3572A5")).Render("[py]    ")
	case "rspec":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#CC342D")).Render("[rspec] ")
	case "maven":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#C71A36")).Render("[maven] ")
	case "gradle":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#02303A")).Render("[gradle]")
	case "phpunit":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#777BB4")).Render("[php]   ")
	case "cargo":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#DEA584")).Render("[cargo] ")
	default:
		return lipgloss.NewStyle().Foreground(t.Muted).Render("[test]  ")
	}
}

func (m *dashboardModel) designView() string {
	t := m.theme
	innerH := m.height - 14
	if innerH < 4 {
		innerH = 4
	}
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Design Artifacts")
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", m.width-4))
	if len(m.designTree) == 0 {
		return title + "\n" + sep + "\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("docs/design/ is empty")
	}
	lines := []string{title, sep}
	cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
	start := m.designCur - innerH/2
	if start < 0 {
		start = 0
	}
	for i, entry := range m.designTree[start:] {
		if i >= innerH {
			break
		}
		abs := start + i
		var line string
		if abs == m.designCur {
			line = cursor + " " + lipgloss.NewStyle().Foreground(t.Foreground).Render(entry)
		} else {
			line = "  " + lipgloss.NewStyle().Foreground(t.Muted).Render(entry)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m *dashboardModel) logsView() string {
	t := m.theme
	innerH := m.height - 14
	if innerH < 4 {
		innerH = 4
	}
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Skill Logs (.claude/logs/skills.jsonl)")
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", m.width-4))
	if len(m.logLines) == 0 {
		return title + "\n" + sep + "\n" + lipgloss.NewStyle().Foreground(t.Muted).Render("no log entries yet")
	}
	lines := []string{title, sep}
	start := m.logsCur - innerH + 2
	if start < 0 {
		start = 0
	}
	for i, entry := range m.logLines[start:] {
		if i >= innerH {
			break
		}
		abs := start + i
		col := t.Muted
		if abs == m.logsCur {
			col = t.Foreground
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(col).Render("  "+truncate(entry, m.width-6)))
	}
	hint := lipgloss.NewStyle().Foreground(t.Muted).Render(
		fmt.Sprintf("  j/k scroll · %d/%d", m.logsCur+1, len(m.logLines)))
	lines = append(lines, hint)
	return strings.Join(lines, "\n")
}

func (m *dashboardModel) skillsBrowserView() string {
	t := m.theme

	// Tab headers
	installedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	searchStyle := lipgloss.NewStyle().Foreground(t.Muted)
	if !m.skillsTabSearch {
		installedStyle = lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Underline(true)
	} else {
		searchStyle = lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Underline(true)
	}
	tabs := "  " + installedStyle.Render("Installed") + "  " + searchStyle.Render("Search")
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + strings.Repeat("─", 62))
	lines := []string{tabs, sep}

	if m.npxPath == "" {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(t.Warning).Render("  npx not found — install Node.js from nodejs.org"),
		)
		return strings.Join(lines, "\n")
	}

	if !m.skillsTabSearch {
		// ── Installed tab ─────────────────────────────────────────
		if m.installedLoading {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Muted).Render("  loading installed skills…"))
		} else if m.installedErr != "" {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Error).Render("  "+m.installedErr))
		} else if len(m.installedSkills) == 0 {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Muted).Render("  no skills installed — switch to Search tab to add some"))
		} else {
			cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
			for i, sk := range m.installedSkills {
				name := lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Render(fmt.Sprintf("%-24s", sk.name))
				pkg := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%-28s", sk.pkg))
				scope := lipgloss.NewStyle().Foreground(t.Muted).Render(sk.scope)
				if i == m.installedCur {
					lines = append(lines, "  "+cursor+" "+name+" "+pkg+" "+scope)
				} else {
					lines = append(lines, "      "+name+" "+pkg+" "+scope)
				}
			}
		}
	} else {
		// ── Search tab ────────────────────────────────────────────
		search := lipgloss.NewStyle().Foreground(t.Muted).Render("  search: ") +
			lipgloss.NewStyle().Foreground(t.Foreground).Render(m.skillsQuery+"█")
		lines = append(lines, search)

		if m.skillsLoading {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Muted).Render("  searching…"))
		} else if m.skillsErr != "" {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Error).Render("  "+m.skillsErr))
		} else if len(m.skillsItems) == 0 && m.skillsSearched {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Muted).Render("  no results — try a different query"))
		} else if len(m.skillsItems) == 0 {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Muted).Render("  type a query and press Enter to search"))
		} else {
			cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
			for i, sk := range m.skillsItems {
				pkg := lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Render(fmt.Sprintf("%-42s", sk.pkg))
				installs := lipgloss.NewStyle().Foreground(t.Muted).Render(sk.installs + " installs")
				if i == m.skillsCur {
					lines = append(lines, "  "+cursor+" "+pkg+" "+installs)
					if sk.url != "" {
						lines = append(lines, "       "+lipgloss.NewStyle().Foreground(t.Muted).Render(sk.url))
					}
				} else {
					lines = append(lines, "      "+pkg+" "+installs)
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// storyDetailView renders the selected Story.md as a full-screen overlay.
func (m *dashboardModel) storyDetailView() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  " + m.storyTitle)
	dir := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + m.storyDir)
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + strings.Repeat("─", 62))

	visible := m.height - 14
	if visible < 4 {
		visible = 4
	}
	end := m.storyScroll + visible
	if end > len(m.storyLines) {
		end = len(m.storyLines)
	}
	window := m.storyLines[m.storyScroll:end]

	var sb strings.Builder
	sb.WriteString(title + "\n" + dir + "\n" + sep + "\n\n")
	for _, l := range window {
		sb.WriteString("  " + colorizeStoryLine(l, t) + "\n")
	}

	total := len(m.storyLines)
	if total > visible {
		pct := (m.storyScroll * 100) / (total - visible)
		sb.WriteString(fmt.Sprintf("\n  %s\n",
			lipgloss.NewStyle().Foreground(t.Muted).Render(
				fmt.Sprintf("(%d%%)  j/k scroll · e re-edit · Esc close", pct))))
	} else {
		sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(t.Muted).Render("e re-edit · Esc close") + "\n")
	}
	return sb.String()
}

// colorizeStoryLine applies minimal markdown-aware colours to a Story.md line.
func colorizeStoryLine(line string, t Theme) string {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "# "):
		return lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(line)
	case strings.HasPrefix(trimmed, "## "):
		return lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(line)
	case strings.HasPrefix(trimmed, "### "):
		return lipgloss.NewStyle().Foreground(t.Warning).Render(line)
	case strings.HasPrefix(trimmed, "- [ ]"):
		check := lipgloss.NewStyle().Foreground(t.Muted).Render("- [ ]")
		rest := lipgloss.NewStyle().Foreground(t.Foreground).Render(line[strings.Index(line, "- [ ]")+5:])
		return check + rest
	case strings.HasPrefix(trimmed, "- [x]"), strings.HasPrefix(trimmed, "- [X]"):
		check := lipgloss.NewStyle().Foreground(t.Success).Render("- [x]")
		rest := lipgloss.NewStyle().Foreground(t.Muted).Render(line[strings.Index(line, "- [")+5:])
		return check + rest
	case strings.HasPrefix(trimmed, "```"):
		return lipgloss.NewStyle().Foreground(t.Muted).Render(line)
	case strings.HasPrefix(trimmed, "Feature:"),
		strings.HasPrefix(trimmed, "Scenario:"),
		strings.HasPrefix(trimmed, "Given "),
		strings.HasPrefix(trimmed, "When "),
		strings.HasPrefix(trimmed, "Then "),
		strings.HasPrefix(trimmed, "And "),
		strings.HasPrefix(trimmed, "But "),
		strings.HasPrefix(trimmed, "@"):
		return colorizeGherkin(line, t)
	default:
		return lipgloss.NewStyle().Foreground(t.Foreground).Render(line)
	}
}

func (m *dashboardModel) helpView() string {
	t := m.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", 62))

	keyBindings := [][2]string{
		{"Tab / Shift+Tab", "cycle panes"},
		{"j / k  (↓ / ↑)", "navigate rows"},
		{"gg / G", "jump to top / bottom"},
		{"s  a  p  Q", "focus Stories / Agents / PRs / QA"},
		{"Enter", "open detail popup"},
		{"o  (Sessions pane)", "open session in Claude / OpenCode"},
		{"o  (PRs pane)", "open PR in browser"},
		{"S", "ship-safe audit (shipsafecli.com)"},
		{"d", "toggle Design pane (full-screen)"},
		{"D", "Design Review — approve wireframe/mockup (Stories pane)"},
		{"C", "Git Changes — view & navigate working-tree diffs"},
		{"l", "toggle Logs pane (full-screen)"},
		{"n", "new story → Gherkin requirements wizard"},
		{"u", "update — re-sync template files"},
		{"r", "reload all pane data"},
		{"F", "Skills marketplace (skills.sh)"},
		{"x", "Quick Prompt — pick a skill or agent and launch"},
		{"P", "Pipeline status — [a] approve gate · [v] open design portal"},
		{"/", "search within active pane"},
		{"?", "this help overlay"},
		{"q  /  Ctrl+C", "quit"},
	}

	cmdRef := [][2]string{
		{":q  :wq  :q!  :x", "quit"},
		{":e  :e!  :r  :reload", "reload data"},
		{":n  :req  :story", "new story wizard"},
		{":u  :update", "re-sync template"},
		{":labels", "bootstrap GitHub labels"},
		{":project", "create GitHub Project v2"},
		{":theme <name>", "switch colour theme"},
		{":colo <name>", "alias for :theme"},
		{":debug", "toggle debug log"},
		{":help  :h  :?", "this overlay"},
		{"", ""},
		{"themes:", "tokyo-night  catppuccin-mocha"},
		{"", "gruvbox  nord  everforest"},
	}

	renderCol := func(rows [][2]string) []string {
		var out []string
		for _, p := range rows {
			if p[0] == "" && p[1] == "" {
				out = append(out, "")
				continue
			}
			key := lipgloss.NewStyle().Foreground(t.Accent).Render(fmt.Sprintf("  %-24s", p[0]))
			val := lipgloss.NewStyle().Foreground(t.Foreground).Render(p[1])
			out = append(out, key+val)
		}
		return out
	}

	leftTitle := titleStyle.Render("  Keybindings")
	rightTitle := titleStyle.Render("  : Commands")
	leftLines := renderCol(keyBindings)
	rightLines := renderCol(cmdRef)

	// pad both columns to same length
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, "")
	}

	halfW := (m.width - 4) / 2
	colStyle := lipgloss.NewStyle().Width(halfW)

	var rows []string
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		colStyle.Render(leftTitle), colStyle.Render(rightTitle))
	rows = append(rows, header, sep)
	for i := range leftLines {
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			colStyle.Render(leftLines[i]), colStyle.Render(rightLines[i]))
		rows = append(rows, row)
	}
	rows = append(rows, "", lipgloss.NewStyle().Foreground(t.Muted).Render("  Press any key to close"))
	return strings.Join(rows, "\n")
}

func (m *dashboardModel) designReviewView() string {
	t := m.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", 62))

	var rows []string
	rows = append(rows, titleStyle.Render("  Design Review — "+m.designReview.StoryID), sep)

	cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
	for i, a := range m.designReview.Artifacts {
		col := t.Foreground
		if a.Kind != "a11y" && a.Status == "approved" {
			col = t.Success
		}
		if a.Kind != "a11y" && a.Exists && a.Status != "approved" {
			col = t.Warning
		}
		line := lipgloss.NewStyle().Foreground(col).Render(a.label())
		if i == m.designReviewCur {
			rows = append(rows, "  "+cursor+" "+line)
		} else {
			rows = append(rows, "    "+line)
		}
	}
	rows = append(rows, "")

	if m.designReviewCur < len(m.designReview.Artifacts) {
		a := m.designReview.Artifacts[m.designReviewCur]
		rows = append(rows, titleStyle.Render("  "+a.Path), sep)
		if !a.Exists {
			rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render("  (not generated yet)"))
		} else if b, err := os.ReadFile(a.Path); err == nil {
			lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
			budget := m.height - 14
			start := m.designReviewScroll
			if start > len(lines) {
				start = len(lines)
			}
			for j := start; j < len(lines) && j < start+budget; j++ {
				rows = append(rows, lipgloss.NewStyle().Foreground(t.Foreground).Render("  "+lines[j]))
			}
		}
	}
	return strings.Join(rows, "\n")
}

func gitStatusColor(t Theme, st string) lipgloss.Color {
	switch st {
	case "M":
		return t.Warning
	case "A":
		return t.Success
	case "D":
		return t.Error
	case "R", "C":
		return t.Accent
	default:
		return t.Muted
	}
}

func colorizeDiffLine(t Theme, line string, maxW int) string {
	var c lipgloss.Color
	switch diffLineKind(line) {
	case "add":
		c = t.Success
	case "del":
		c = t.Error
	case "hunk":
		c = t.Accent
	case "meta":
		c = t.Muted
	default:
		c = t.Foreground
	}
	return lipgloss.NewStyle().Foreground(c).Render(truncate(line, maxW))
}

func (m *dashboardModel) gitChangesView() string {
	t := m.theme
	if m.gitChanges.Err != "" {
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Error).Padding(0, 1)
		return "\n" + box.Render(lipgloss.NewStyle().Foreground(t.Error).Render("✗ "+m.gitChanges.Err))
	}
	if len(m.gitChanges.Files) == 0 {
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Muted).Padding(1, 2)
		return "\n" + box.Render(lipgloss.NewStyle().Foreground(t.Success).Render("✓ working tree clean — no changes to show"))
	}

	innerH := m.height - 10
	if innerH < 4 {
		innerH = 4
	}
	halfW := (m.width - 4) / 2

	fileLines := []string{lipgloss.NewStyle().Foreground(t.Primary).Bold(true).
		Render(fmt.Sprintf("Git Changes — %d files", len(m.gitChanges.Files)))}
	for i, f := range m.gitChanges.Files {
		if i >= innerH-2 {
			break
		}
		glyph := lipgloss.NewStyle().Foreground(gitStatusColor(t, f.Status)).Render(fmt.Sprintf("%-2s", f.Status))
		path := truncate(f.Path, halfW-8)
		if i == m.gitChangesCur {
			cur := lipgloss.NewStyle().Foreground(t.Accent).Render("▸")
			fileLines = append(fileLines, cur+" "+glyph+" "+lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(path))
		} else {
			fileLines = append(fileLines, "  "+glyph+" "+lipgloss.NewStyle().Foreground(t.Foreground).Render(path))
		}
	}
	fileLines = append(fileLines, "", lipgloss.NewStyle().Foreground(t.Muted).
		Render(fmt.Sprintf("%d changed · %d staged", len(m.gitChanges.Files), m.gitChanges.stagedCount())))

	var diffLines []string
	for j := m.gitDiffScroll; j < len(m.gitDiffLines) && j < m.gitDiffScroll+innerH; j++ {
		diffLines = append(diffLines, colorizeDiffLine(t, m.gitDiffLines[j], halfW-2))
	}

	pane := lipgloss.NewStyle().Width(halfW).Height(innerH).Border(lipgloss.RoundedBorder()).PaddingLeft(1)
	left := pane.BorderForeground(t.Primary).Render(strings.Join(fileLines, "\n"))
	right := pane.BorderForeground(t.Muted).Render(strings.Join(diffLines, "\n"))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// extractGherkinFromLines pulls the content inside the first ```gherkin ... ``` block.
// Falls back to returning all lines that look like Gherkin keywords.
func extractGherkinFromLines(lines []string) string {
	var inFence bool
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if !inFence && (t == "```gherkin" || t == "```") {
			inFence = true
			continue
		}
		if inFence {
			if t == "```" {
				break
			}
			out = append(out, l)
		}
	}
	if len(out) > 0 {
		return strings.TrimSpace(strings.Join(out, "\n"))
	}
	// No fence found — return any Gherkin-looking lines
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		for _, kw := range []string{"Feature:", "Background:", "Scenario", "Given ", "When ", "Then ", "And ", "But ", "@"} {
			if strings.HasPrefix(t, kw) {
				out = append(out, l)
				break
			}
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// openStoryDetail reads a Story.md and sets up the overlay.
func (m *dashboardModel) openStoryDetail(s storyRow) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		m.status = "✗ could not read " + s.path
		m.statusErr = true
		return
	}
	// Strip YAML frontmatter (between --- delimiters)
	content := string(raw)
	if strings.HasPrefix(content, "---") {
		end := strings.Index(content[3:], "\n---")
		if end >= 0 {
			content = strings.TrimSpace(content[3+end+4:])
		}
	}
	m.storyLines = strings.Split(content, "\n")
	m.storyScroll = 0
	m.storyTitle = s.id
	m.storyDir = filepath.Dir(s.path)
	m.showStory = true
}

// openSessionDetail loads the full activity timeline for a session.
func (m *dashboardModel) openSessionDetail(s sessionRow) {
	m.sessionTitle = s.title
	m.sessionSource = s.source
	m.sessionScroll = 0
	m.sessionLines = nil

	switch s.source {
	case "claude":
		m.sessionLines = loadClaudeSessionLines(s.id)
	case "opencode":
		m.sessionLines = loadOpenCodeSessionLines(s.id)
	case "copilot":
		m.sessionLines = loadCopilotSessionLines(s.id)
	default:
		m.sessionLines = []string{"(no detail available for source: " + s.source + ")"}
	}
	if len(m.sessionLines) == 0 {
		m.sessionLines = []string{"(no entries found)"}
	}
	m.showSession = true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// quickPromptView renders the Quick Prompt picker as a centered popup.
func (m *dashboardModel) quickPromptView() string {
	t := m.theme

	activeItems := m.taffyItems
	if m.quickMode == "items" {
		activeItems = m.quickItems
	}
	filtered := quickFilter(activeItems, m.quickSearch)

	// mode label and toggle hint in title
	modeLabel := ""
	toggleHint := ""
	if m.quickMode == "taffy" {
		modeLabel = lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("TAFFY")
		toggleHint = lipgloss.NewStyle().Foreground(t.Muted).Render("  [t] skills/agents")
	} else {
		modeLabel = lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("skills · agents")
		toggleHint = lipgloss.NewStyle().Foreground(t.Muted).Render("  [t] taffy")
	}

	searchBar := ""
	if m.quickSearch != "" {
		searchBar = "  " + lipgloss.NewStyle().Foreground(t.Warning).Render("/ "+m.quickSearch)
	} else {
		searchBar = "  " + lipgloss.NewStyle().Foreground(t.Muted).Render("/ type to filter…")
	}
	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("[x] Quick Launch") +
		"  " + modeLabel + toggleHint +
		"\n" + searchBar

	visibleItems := (m.height - 17) / 3
	if visibleItems < 3 {
		visibleItems = 3
	}
	descWidth := m.popupInnerWidth() - 4
	if descWidth < 10 {
		descWidth = 10
	}

	var bodyLines []string
	if len(filtered) == 0 {
		if m.quickSearch != "" {
			bodyLines = append(bodyLines,
				lipgloss.NewStyle().Foreground(t.Muted).Render("  No matches for \""+m.quickSearch+"\"."),
			)
		} else if m.quickMode == "taffy" {
			bodyLines = append(bodyLines,
				lipgloss.NewStyle().Foreground(t.Muted).Render("  No taffy workflows found. Run maple init."),
			)
		} else {
			bodyLines = append(bodyLines,
				lipgloss.NewStyle().Foreground(t.Muted).Render("  No skills or agents found. Run maple init."),
			)
		}
	} else {
		scroll := m.quickItemScroll
		end := scroll + visibleItems
		if end > len(filtered) {
			end = len(filtered)
		}
		for i, item := range filtered[scroll:end] {
			abs := scroll + i
			cursor := "  "
			nameStyle := lipgloss.NewStyle().Foreground(t.Foreground)
			descStyle := lipgloss.NewStyle().Foreground(t.Muted)
			if abs == m.quickItemCur {
				cursor = "▶ "
				nameStyle = lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
				descStyle = lipgloss.NewStyle().Foreground(t.Foreground)
			}

			var badge string
			switch item.kind {
			case "taffy":
				stages := ""
				if item.stageCount > 0 {
					stages = fmt.Sprintf(" %d stages", item.stageCount)
				}
				badge = lipgloss.NewStyle().Foreground(t.Success).Render("[taffy" + stages + "]")
			case "skill":
				badge = lipgloss.NewStyle().Foreground(t.Accent).Render("[skill]")
			default:
				badge = lipgloss.NewStyle().Foreground(t.Primary).Render("[agent]")
			}

			desc := item.description
			if len(desc) > descWidth {
				desc = desc[:descWidth-1] + "…"
			}
			bodyLines = append(bodyLines,
				cursor+badge+"  "+nameStyle.Render(item.name),
				"    "+descStyle.Render(desc),
				"",
			)
		}

		if len(filtered) > visibleItems {
			pct := 0
			if len(filtered)-visibleItems > 0 {
				pct = (m.quickItemScroll * 100) / (len(filtered) - visibleItems)
			}
			bodyLines = append(bodyLines,
				lipgloss.NewStyle().Foreground(t.Muted).Render(
					fmt.Sprintf("  %d/%d  (%d%%)", m.quickItemCur+1, len(filtered), pct)),
			)
		}
	}

	body := strings.Join(bodyLines, "\n")
	hint := "j/k scroll · Enter select · [t] toggle mode · type to search · Esc close"

	return m.popupBox(title, body, hint, t.Accent)
}

// quickLaunchView renders the quick launch overlay.
// Mode A (quickLaunchPickHarness==true): harness picker — no pinned session found.
// Mode B: prompt input — harness + optional pinned session known.
func (m *dashboardModel) quickLaunchView() string {
	t := m.theme
	tools := launcherTools()

	selStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	pinnedStyle := lipgloss.NewStyle().Foreground(t.Success)

	titleText := fmt.Sprintf("Quick Prompt: %s", lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("/"+m.quickLaunchName))
	var bodyLines []string
	var hint string

	if m.quickLaunchPickHarness {
		// ── Harness picker ──────────────────────────────────────────────────────
		bodyLines = append(bodyLines, mutedStyle.Render("Choose a harness:"), "")
		for i, tool := range tools {
			cursor := "  "
			style := mutedStyle
			if i == m.quickLaunchHarnessCur {
				cursor = "▶ "
				style = selStyle
			}
			line := cursor + style.Render(tool)
			if sid := m.pinnedSessions[tool]; sid != "" {
				short := sid
				if len(short) > 8 {
					short = short[:8] + "…"
				}
				line += "  " + pinnedStyle.Render("★ resumes "+short)
			}
			bodyLines = append(bodyLines, line)
		}
		hint = "j/k navigate · Enter select harness · Esc close"
	} else {
		// ── Prompt input ────────────────────────────────────────────────────────
		harness := m.quickLaunchHarness
		sessionID := m.pinnedSessions[harness]

		harnessLine := "  Harness:  " + selStyle.Render(harness)
		if sessionID != "" {
			short := sessionID
			if len(short) > 8 {
				short = short[:8] + "…"
			}
			harnessLine += "  " + pinnedStyle.Render("★ resuming session "+short)
		}

		cmd := "/" + m.quickLaunchName
		promptLabel := "  Add context (optional — press Enter to launch now):"
		hint = "type context · Enter launch · Esc back"
		promptValue := m.quickLaunchPrompt
		if m.quickLaunchKind == "taffy" {
			promptLabel = "  Describe this feature (required):"
			hint = "type feature requirements · Enter launch · Esc back"
			if strings.TrimSpace(promptValue) == "" {
				promptValue = "<feature requirements>"
			}
		}
		bodyLines = append(bodyLines,
			harnessLine,
			"",
			mutedStyle.Render(promptLabel),
			"  "+cmd+" "+promptValue+"█",
		)
	}

	return m.popupBox(titleText, strings.Join(bodyLines, "\n"), hint, t.Primary)
}

// popupBox renders a centered rounded-border popup over the content area.
// title, body (pre-formatted lines joined with \n), and hint are rendered inside.
func (m *dashboardModel) popupBox(title, body, hint string, borderColor lipgloss.Color) string {
	t := m.theme

	popW := (m.width * 3) / 4
	if popW > 114 {
		popW = 114
	}
	if popW < 52 {
		popW = 52
	}
	innerW := popW - 6 // border(2) + padding sides(4)

	availH := m.height - 6
	if availH < 10 {
		availH = 10
	}
	// Hard cap: box must not exceed available height.
	// border(2) + padding(2) + title(2) + sep(1) + blanks(2) + hint(1) = 10 overhead lines.
	maxBodyLines := availH - 10
	if maxBodyLines < 2 {
		maxBodyLines = 2
	}
	// Clip body to maxBodyLines so the box never overflows the terminal.
	bodyLines := strings.Split(body, "\n")
	if len(bodyLines) > maxBodyLines {
		bodyLines = bodyLines[:maxBodyLines]
	}
	body = strings.Join(bodyLines, "\n")

	titleBar := lipgloss.NewStyle().Foreground(borderColor).Bold(true).Render(title)
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", innerW))
	hintBar := lipgloss.NewStyle().Foreground(t.Muted).Render(hint)

	inner := strings.Join([]string{titleBar, sep, "", body, "", hintBar}, "\n")

	box := lipgloss.NewStyle().
		Width(innerW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Render(inner)

	return lipgloss.Place(m.width, availH, lipgloss.Center, lipgloss.Center, box)
}

// popupInnerWidth returns the body width available inside a popupBox.
func (m *dashboardModel) popupInnerWidth() int {
	popW := (m.width * 3) / 4
	if popW > 114 {
		popW = 114
	}
	if popW < 52 {
		popW = 52
	}
	return popW - 6 - 4 // minus border+padding sides, minus leading "    "
}

// pipelineStatusView shows the current taffy pipeline state from .claude/state/maple.json.
func (m *dashboardModel) pipelineStatusView() string {
	t := m.theme
	ps := m.pipelineState

	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("[P] Pipeline Status")

	var bodyLines []string

	if !ps.isTaffy() {
		bodyLines = append(bodyLines,
			lipgloss.NewStyle().Foreground(t.Muted).Render("  No active taffy pipeline."),
			"",
			lipgloss.NewStyle().Foreground(t.Muted).Render("  Launch one with [x] — select a taffy workflow or use /pipeline-runner <name>."),
		)
	} else {
		stale := ps.isStale()
		iconStyle := lipgloss.NewStyle().Foreground(t.Success)
		displayStatus := ps.Status
		switch ps.Status {
		case "PAUSED":
			iconStyle = lipgloss.NewStyle().Foreground(t.Accent)
		case "RATE_LIMITED":
			iconStyle = lipgloss.NewStyle().Foreground(t.Warning)
		case "FAILED":
			iconStyle = lipgloss.NewStyle().Foreground(t.Error)
		case "DONE":
			iconStyle = lipgloss.NewStyle().Foreground(t.Success)
		case "RUNNING":
			if stale {
				iconStyle = lipgloss.NewStyle().Foreground(t.Muted)
				displayStatus = "RUNNING (stale — agent may have exited)"
			}
		}

		bodyLines = append(bodyLines,
			fmt.Sprintf("  Workflow:    %s", lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Render(ps.Taffy)),
			fmt.Sprintf("  Stage:       %s", lipgloss.NewStyle().Foreground(t.Foreground).Render(ps.Stage)),
			fmt.Sprintf("  Status:      %s %s", iconStyle.Render(ps.statusIcon()), iconStyle.Render(displayStatus)),
		)
		if stale {
			bodyLines = append(bodyLines,
				"",
				lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  Last update was >%s ago. If the agent is gone, press [c] to clear.", stalePipelineThreshold)),
			)
		}
		if ps.isRateLimited() {
			now := time.Now()
			cleared := ps.windowCleared(now)
			warn := lipgloss.NewStyle().Foreground(t.Warning)
			bodyLines = append(bodyLines, "", warn.Render("  ⚑ Rate-limited: "+ps.RateLimitReason))
			bodyLines = append(bodyLines, warn.Render("  Resets "+formatRateLimitCountdown(ps.ResumeAt, now)))
			auto := "OFF"
			if ps.AutoResume {
				auto = "ON"
			}
			if cleared {
				bodyLines = append(bodyLines,
					lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("  window cleared — [r] resume now"))
			} else {
				bodyLines = append(bodyLines,
					lipgloss.NewStyle().Foreground(t.Muted).Render("  [r] force resume before the window clears"))
			}
			bodyLines = append(bodyLines,
				lipgloss.NewStyle().Foreground(t.Primary).Render("  [A] auto-resume: "+auto+"   [c] clear"))
		}
		if ps.AwaitingApproval != "" || m.approvalPending != "" {
			stage := ps.AwaitingApproval
			if m.approvalPending != "" {
				stage = m.approvalPending
			}
			bodyLines = append(bodyLines,
				"",
				lipgloss.NewStyle().Foreground(t.Accent).Render("  ⏸ Awaiting approval: "+stage),
				lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("  [a] approve  — advances pipeline to next stage"),
				lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  [v] open design review portal"),
				lipgloss.NewStyle().Foreground(t.Muted).Render("  Portal \"request changes\" records feedback and keeps this gate paused."),
			)
		}
		if ps.UpdatedAt != "" {
			bodyLines = append(bodyLines,
				"",
				lipgloss.NewStyle().Foreground(t.Muted).Render("  Updated: "+ps.UpdatedAt),
			)
		}
	}

	bodyLines = append(bodyLines, "", lipgloss.NewStyle().Foreground(t.Muted).Render("  Press any key to close"))

	inner := title + "\n\n" + strings.Join(bodyLines, "\n")
	innerW := m.width - 10
	if innerW < 40 {
		innerW = 40
	}
	box := lipgloss.NewStyle().
		Width(innerW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2).
		Render(inner)

	availH := m.height - 6
	if availH < 10 {
		availH = 10
	}
	return lipgloss.Place(m.width, availH, lipgloss.Center, lipgloss.Center, box)
}

// launcherView renders the session launcher overlay (L key).
func (m *dashboardModel) launcherView() string {
	t := m.theme
	tools := launcherTools()

	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("[L] Launch Session")
	var bodyLines []string

	pinnedStyle := lipgloss.NewStyle().Foreground(t.Success)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	selStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)

	if !m.launcherInput {
		bodyLines = append(bodyLines, mutedStyle.Render("  Select tool  [j/k] navigate · [Enter] type command · [Esc] close"), "")
		for i, tool := range tools {
			cursor := "  "
			style := mutedStyle
			if i == m.launcherCur {
				cursor = "▶ "
				style = selStyle
			}
			pinned := ""
			if id := m.pinnedSessions[tool]; id != "" {
				short := id
				if len(short) > 8 {
					short = short[:8]
				}
				pinned = " " + pinnedStyle.Render("★ pinned: "+short+"…")
			}
			bodyLines = append(bodyLines, cursor+style.Render(tool)+pinned)
		}
	} else {
		tool := ""
		if m.launcherCur < len(tools) {
			tool = tools[m.launcherCur]
		}
		bodyLines = append(bodyLines,
			mutedStyle.Render("  Launching: "+selStyle.Render(tool)),
			"",
			mutedStyle.Render("  Command (optional — leave empty to open interactively):"),
			"  "+m.launcherCmd+"█",
			"",
			mutedStyle.Render("  [Enter] launch · [Esc] back"),
		)
		if id := m.pinnedSessions[tool]; id != "" {
			short := id
			if len(short) > 8 {
				short = short[:8]
			}
			bodyLines = append(bodyLines, "", pinnedStyle.Render("  ★ Will resume pinned session "+short+"…"))
		}
	}

	inner := title + "\n\n" + strings.Join(bodyLines, "\n")
	innerW := m.width - 10
	if innerW < 50 {
		innerW = 50
	}
	box := lipgloss.NewStyle().
		Width(innerW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2).
		Render(inner)

	availH := m.height - 6
	if availH < 10 {
		availH = 10
	}
	return lipgloss.Place(m.width, availH, lipgloss.Center, lipgloss.Center, box)
}

// rtkHarnessView renders the RTK harness selector overlay (R key).
func (m *dashboardModel) rtkHarnessView() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("[R] RTK Token Optimizer — Wire Harnesses")
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	selStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	doneStyle := lipgloss.NewStyle().Foreground(t.Success)
	pendStyle := lipgloss.NewStyle().Foreground(t.Accent)

	var bodyLines []string
	if m.rtkHarnessRunning {
		bodyLines = append(bodyLines, mutedStyle.Render("  running rtk init …"), "")
	} else {
		bodyLines = append(bodyLines, mutedStyle.Render("  [j/k] navigate · [Space] toggle · [Enter] install · [Esc] close"), "")
	}

	for i, h := range allRTKHarnesses {
		cursor := "  "
		nameStyle := mutedStyle
		if i == m.rtkHarnessCur && !m.rtkHarnessRunning {
			cursor = "▶ "
			nameStyle = selStyle
		}
		var marker string
		switch {
		case m.rtkHarnessInstalled[h.key]:
			marker = " " + doneStyle.Render("✓ installed")
		case m.rtkHarnessToggled[h.key]:
			marker = " " + pendStyle.Render("◉ selected")
		default:
			marker = "   " + mutedStyle.Render("○")
		}
		bodyLines = append(bodyLines, cursor+nameStyle.Render(h.name)+marker)
	}

	if !m.rtkHarnessRunning {
		selected := 0
		for range m.rtkHarnessToggled {
			selected++
		}
		if selected > 0 {
			bodyLines = append(bodyLines, "", pendStyle.Render(fmt.Sprintf("  %d harness(es) selected — press Enter to install", selected)))
		}
	}

	inner := title + "\n\n" + strings.Join(bodyLines, "\n")
	innerW := m.width - 10
	if innerW < 60 {
		innerW = 60
	}
	box := lipgloss.NewStyle().
		Width(innerW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2).
		Render(inner)

	availH := m.height - 6
	if availH < 10 {
		availH = 10
	}
	return lipgloss.Place(m.width, availH, lipgloss.Center, lipgloss.Center, box)
}

// manualLaunchView renders a modal telling the user maple couldn't open a new
// terminal, and showing the command they should paste themselves.
func (m *dashboardModel) manualLaunchView() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Error).Bold(true).Render("⚠  Could not open a new terminal tab")
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	codeStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(t.Success)

	var cmdLine string
	for _, a := range m.manualLaunchArgs {
		cmdLine += " " + shQuote(a)
	}
	cmdLine = strings.TrimSpace(cmdLine)

	copyHint := mutedStyle.Render("[c] copy to clipboard")
	if m.manualLaunchCopied {
		copyHint = successStyle.Render("✓ copied!")
	}

	bodyLines := []string{
		mutedStyle.Render("maple needs a multiplexer (tmux / zellij) or a supported terminal"),
		mutedStyle.Render("to open the harness in a new tab. Open a tab manually and run:"),
		"",
		"  " + codeStyle.Render(cmdLine),
		"",
		"  " + copyHint + "  " + mutedStyle.Render("[Esc] dismiss"),
		"",
		mutedStyle.Render("Tip: run maple inside tmux or zellij — harnesses open automatically."),
	}

	inner := title + "\n\n" + strings.Join(bodyLines, "\n")
	innerW := m.width - 10
	if innerW < 60 {
		innerW = 60
	}
	box := lipgloss.NewStyle().
		Width(innerW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Error).
		Padding(1, 2).
		Render(inner)

	availH := m.height - 6
	if availH < 10 {
		availH = 10
	}
	return lipgloss.Place(m.width, availH, lipgloss.Center, lipgloss.Center, box)
}

// sessionDetailView renders the session detail as a centered popup.
func (m *dashboardModel) sessionDetailView() string {
	t := m.theme

	borderColor := t.Primary
	switch m.sessionSource {
	case "opencode":
		borderColor = t.Accent
	case "copilot":
		borderColor = t.Warning
	case "maple":
		borderColor = t.Success
	}

	badge := agentSourceBadge(m.sessionSource, t)
	title := badge + "  " + m.sessionTitle

	visible := m.height - 18
	if visible < 4 {
		visible = 4
	}
	end := m.sessionScroll + visible
	if end > len(m.sessionLines) {
		end = len(m.sessionLines)
	}
	window := m.sessionLines[m.sessionScroll:end]

	var bodyLines []string
	for _, l := range window {
		bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(t.Foreground).Render(l))
	}
	body := strings.Join(bodyLines, "\n")

	hint := "j/k scroll · Esc close"
	total := len(m.sessionLines)
	if total > visible && total > 1 {
		pct := (m.sessionScroll * 100) / (total - 1)
		hint = fmt.Sprintf("(%d%%)  j/k scroll · Esc close", pct)
	}

	return m.popupBox(title, body, hint, borderColor)
}

// shipSafeView renders the ship-safe audit output as a centered popup.
func (m *dashboardModel) shipSafeView() string {
	t := m.theme

	borderColor := t.Success
	statusLabel := "CLEAN"
	if m.shipSafeRunning {
		borderColor = t.Muted
		statusLabel = "auditing…"
	} else if m.shipSafeFailed {
		borderColor = t.Error
		statusLabel = "ISSUES FOUND"
	}

	title := lipgloss.NewStyle().Foreground(borderColor).Bold(true).Render("[ship-safe] "+statusLabel) +
		lipgloss.NewStyle().Foreground(t.Muted).Render("  npx ship-safe audit .")

	visible := m.height - 18
	if visible < 4 {
		visible = 4
	}
	end := m.shipSafeScroll + visible
	if end > len(m.shipSafeLines) {
		end = len(m.shipSafeLines)
	}
	window := m.shipSafeLines[m.shipSafeScroll:end]

	var bodyLines []string
	for _, l := range window {
		col := t.Foreground
		tl := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(tl, "✓"), strings.HasPrefix(tl, "✅"),
			strings.HasPrefix(tl, "PASS"), strings.HasPrefix(tl, "ok "),
			strings.Contains(tl, "no issues"):
			col = t.Success
		case strings.HasPrefix(tl, "✗"), strings.HasPrefix(tl, "❌"),
			strings.HasPrefix(tl, "FAIL"), strings.HasPrefix(tl, "ERROR"),
			strings.HasPrefix(tl, "CRITICAL"), strings.HasPrefix(tl, "HIGH"):
			col = t.Error
		case strings.HasPrefix(tl, "⚠"), strings.HasPrefix(tl, "WARN"),
			strings.HasPrefix(tl, "MEDIUM"), strings.HasPrefix(tl, "LOW"):
			col = t.Warning
		}
		bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(col).Render(l))
	}
	body := strings.Join(bodyLines, "\n")

	hint := "j/k scroll · Esc close"
	total := len(m.shipSafeLines)
	if total > visible && total > 1 {
		pct := (m.shipSafeScroll * 100) / (total - 1)
		hint = fmt.Sprintf("(%d%%)  j/k scroll · Esc close", pct)
	}

	return m.popupBox(title, body, hint, borderColor)
}

// qaFileDetailView renders a test file as a full-screen overlay.
func (m *dashboardModel) qaFileDetailView() string {
	t := m.theme
	badge := testFrameworkBadge(m.qaFileFramework, t)
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  " + filepath.Base(m.qaFileTitle))
	path := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + m.qaFileTitle)
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + strings.Repeat("─", 62))

	visible := m.height - 14
	if visible < 4 {
		visible = 4
	}
	end := m.qaFileScroll + visible
	if end > len(m.qaFileLines) {
		end = len(m.qaFileLines)
	}
	window := m.qaFileLines[m.qaFileScroll:end]

	var sb strings.Builder
	sb.WriteString(badge + " " + title + "\n" + path + "\n" + sep + "\n\n")
	for _, l := range window {
		var rendered string
		if m.qaFileFramework == "gherkin" {
			rendered = colorizeGherkin(l, t)
		} else {
			rendered = lipgloss.NewStyle().Foreground(t.Foreground).Render(l)
		}
		sb.WriteString("  " + rendered + "\n")
	}

	total := len(m.qaFileLines)
	if total > visible {
		pct := (m.qaFileScroll * 100) / (total - visible)
		sb.WriteString(fmt.Sprintf("\n  %s\n",
			lipgloss.NewStyle().Foreground(t.Muted).Render(
				fmt.Sprintf("(%d%%)  j/k scroll · r run test · Esc close", pct))))
	} else {
		sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(t.Muted).Render("r run test · Esc close") + "\n")
	}
	return sb.String()
}

// testOutputView renders the live/completed test runner output.
func (m *dashboardModel) testOutputView() string {
	t := m.theme

	titleCol := t.Success
	statusStr := "PASSED"
	if m.testOutRunning {
		titleCol = t.Muted
		statusStr = "running…"
	} else if m.testOutFailed {
		titleCol = t.Error
		statusStr = "FAILED"
	}

	title := lipgloss.NewStyle().Foreground(titleCol).Bold(true).Render("  [" + statusStr + "] " + m.testOutTitle)
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + strings.Repeat("─", 62))

	visible := m.height - 14
	if visible < 4 {
		visible = 4
	}
	end := m.testOutScroll + visible
	if end > len(m.testOutLines) {
		end = len(m.testOutLines)
	}
	window := m.testOutLines[m.testOutScroll:end]

	var sb strings.Builder
	sb.WriteString(title + "\n" + sep + "\n\n")
	for _, l := range window {
		col := t.Foreground
		tl := strings.TrimSpace(l)
		if strings.HasPrefix(tl, "FAIL") || strings.HasPrefix(tl, "--- FAIL") ||
			strings.Contains(tl, "FAILED") || strings.HasPrefix(tl, "Error") {
			col = t.Error
		} else if strings.HasPrefix(tl, "ok ") || strings.HasPrefix(tl, "--- PASS") ||
			strings.Contains(tl, "passed") || strings.HasPrefix(tl, "PASS") {
			col = t.Success
		}
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(col).Render(l) + "\n")
	}

	total := len(m.testOutLines)
	if total > visible {
		pct := (m.testOutScroll * 100) / (total - visible)
		sb.WriteString(fmt.Sprintf("\n  %s\n",
			lipgloss.NewStyle().Foreground(t.Muted).Render(
				fmt.Sprintf("(%d%%)  j/k scroll · Esc close", pct))))
	}
	return sb.String()
}

// prDetailView renders `gh pr view` output as a full-screen overlay.
func (m *dashboardModel) prDetailView() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  " + m.prDetailTitle)
	sep := lipgloss.NewStyle().Foreground(t.Muted).Render("  " + strings.Repeat("─", 62))

	if m.prDetailLoading {
		return title + "\n" + sep + "\n\n  " +
			lipgloss.NewStyle().Foreground(t.Muted).Render("loading…") + "\n"
	}

	visible := m.height - 14
	if visible < 4 {
		visible = 4
	}
	end := m.prDetailScroll + visible
	if end > len(m.prDetailLines) {
		end = len(m.prDetailLines)
	}
	window := m.prDetailLines[m.prDetailScroll:end]

	var sb strings.Builder
	sb.WriteString(title + "\n" + sep + "\n\n")
	for _, l := range window {
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(t.Foreground).Render(l) + "\n")
	}

	total := len(m.prDetailLines)
	if total > visible {
		pct := (m.prDetailScroll * 100) / (total - visible)
		sb.WriteString(fmt.Sprintf("\n  %s\n",
			lipgloss.NewStyle().Foreground(t.Muted).Render(
				fmt.Sprintf("(%d%%)  j/k scroll · o browser · a approve · Esc close", pct))))
	} else {
		sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(t.Muted).Render("o browser · a approve · Esc close") + "\n")
	}
	return sb.String()
}
