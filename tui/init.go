package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Init command ─────────────────────────────────────────────────────────────

func runInit(tools Tools, fsys fs.FS, force bool) error {
	// Skip TUI when running in CI or when /dev/tty is unavailable.
	if os.Getenv("CI") != "" || !hasTTY() {
		_, err := doInit(tools, fsys, force)
		return err
	}
	m := newInitModel(tools, fsys, force, false)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func runInitFromMenu(tools Tools, fsys fs.FS, force bool) error {
	m := newInitModel(tools, fsys, force, true)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ─── Wizard model ─────────────────────────────────────────────────────────────

type initStep int

const (
	stepWelcome initStep = iota
	stepConfirm
	stepRun
	stepDone
)

type initModel struct {
	tools      Tools
	templateFS fs.FS
	force      bool
	fromMenu   bool // skip welcome step; jump straight to confirm
	step       initStep
	spinner    spinner.Model
	logs       []string
	err        error
	width      int
	logoFrame  int
	logoDone   bool
}

type initDoneMsg struct{ logs []string; err error }
type initAutoDoneMsg struct{}

func newInitModel(tools Tools, fsys fs.FS, force bool, fromMenu bool) *initModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))
	initialStep := stepWelcome
	if fromMenu {
		initialStep = stepConfirm
	}
	return &initModel{
		tools:      tools,
		templateFS: fsys,
		force:      force,
		fromMenu:   fromMenu,
		step:       initialStep,
		spinner:    s,
	}
}

func (m *initModel) Init() tea.Cmd {
	return logoTick(0)
}

func (m *initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case logoTickMsg:
		if !m.logoDone {
			m.logoFrame++
			if m.logoFrame >= logoFrameCount {
				m.logoDone = true
			} else {
				return m, logoTick(m.logoFrame)
			}
		}

	// Suppress Enter until logo is done
	case tea.KeyMsg:
		if !m.logoDone {
			return m, nil
		}
		switch m.step {
		case stepWelcome:
			if msg.String() == "enter" || msg.String() == "y" {
				m.step = stepConfirm
			} else if msg.String() == "ctrl+c" || msg.String() == "q" {
				return m, tea.Quit
			}
		case stepConfirm:
			switch msg.String() {
			case "enter", "y":
				m.step = stepRun
				return m, tea.Batch(m.spinner.Tick, m.runSetup())
			case "ctrl+c", "q", "n":
				return m, tea.Quit
			}
		case stepDone:
			return m, tea.Quit
		}

	case spinner.TickMsg:
		if m.step == stepRun {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case initDoneMsg:
		m.step = stepDone
		m.logs = msg.logs
		m.err = msg.err
		if m.err == nil {
			return m, tea.Tick(800*time.Millisecond, func(time.Time) tea.Msg { return initAutoDoneMsg{} })
		}

	case initAutoDoneMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m *initModel) View() string {
	t := tokyoNight()
	var header string
	if m.logoDone {
		header = logo()
	} else {
		header = logoAnimFrame(m.logoFrame)
	}
	switch m.step {
	case stepWelcome:
		detected := strings.Join(m.tools.Summary(), "\n")
		body := lipgloss.NewStyle().Foreground(t.Foreground).Render(
			"\nDetected tools:\n" + detected + "\n\n" +
				"This will copy agents, skills, and hooks into the current directory.\n")
		prompt := lipgloss.NewStyle().Foreground(t.Accent).Render("Press Enter to continue, q to quit.")
		return header + body + "\n" + prompt + "\n"

	case stepConfirm:
		cwd, _ := os.Getwd()
		body := lipgloss.NewStyle().Foreground(t.Foreground).Render(
			"\nTarget: " + cwd + "\n\n" +
				"Will set up:\n" +
				m.platformList() + "\n")
		prompt := lipgloss.NewStyle().Foreground(t.Warning).Render("Press Enter to confirm, n to cancel.")
		return header + body + "\n" + prompt + "\n"

	case stepRun:
		return header + "\n" + m.spinner.View() + "  Setting up MAPLE...\n"

	case stepDone:
		if m.err != nil {
			msg := lipgloss.NewStyle().Foreground(t.Error).Render("\n✗ Setup failed: " + m.err.Error())
			return header + msg + "\n"
		}
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("\n✓ MAPLE initialized\n\n"))
		for _, l := range m.logs {
			sb.WriteString("  " + l + "\n")
		}
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("Opening dashboard…"))
		return header + sb.String() + "\n"
	}
	return ""
}

func (m *initModel) platformList() string {
	var items []string
	if m.tools.Claude != "" {
		items = append(items, "  • .claude/ agents + skills + hooks")
	}
	if m.tools.OpenCode != "" {
		items = append(items, "  • .opencode/ agents + skills")
	}
	if m.tools.GHCopilot {
		items = append(items, "  • .github/copilot-instructions.md")
	}
	items = append(items, "  • .cursor/ agents + skills (Cursor IDE)")
	items = append(items, "  • docs/ story templates, specs, design structure")
	items = append(items, "  • Makefile, lefthook.yml, scripts/sdlc/")
	if m.tools.Lefthook != "" {
		items = append(items, "  • lefthook install (hooks wired)")
	}
	return strings.Join(items, "\n")
}

func (m *initModel) runSetup() tea.Cmd {
	force := m.force
	return func() tea.Msg {
		logs, err := doInit(m.tools, m.templateFS, force)
		return initDoneMsg{logs: logs, err: err}
	}
}

// ─── Actual setup logic ───────────────────────────────────────────────────────

func doInit(tools Tools, fsys fs.FS, force bool) ([]string, error) {
	var logs []string
	log := func(s string) { logs = append(logs, s) }

	cwd, err := os.Getwd()
	if err != nil {
		return logs, err
	}

	// Makefile gets special treatment: only the MAPLE-managed section is updated
	// on force (update) so user customisations are preserved.
	makefileDst := filepath.Join(cwd, "Makefile")
	if force {
		if err := patchMakefile(fsys, makefileDst); err != nil {
			log("✗ Makefile (patch): " + err.Error())
		} else {
			log("↺ Makefile (MAPLE section updated)")
		}
	}

	// Always copy: docs structure, scripts, .github (Makefile handled above)
	pairs := []struct{ src, dst string }{
		{"docs", "docs"},
		{"scripts", "scripts"},
		{"Makefile", "Makefile"}, // skipped on first copy if already exists when force=false
		{"lefthook.yml", "lefthook.yml"},
		{".github", ".github"},
		{".gitignore", ".gitignore"},
		{".cursor", ".cursor"},
		// Root harness docs — always shipped. req.go points every agent at all four
		// regardless of which harness is running, so all four must exist.
		{"CLAUDE.md", "CLAUDE.md"},
		{"OPENCODE.md", "OPENCODE.md"},
		{"CURSOR.md", "CURSOR.md"},
		{"COPILOT.md", "COPILOT.md"},
	}

	// Platform-specific copies. In CI copy everything so the smoke test can
	// verify the embedded template is intact regardless of installed tools.
	ciMode := os.Getenv("CI") != ""
	if tools.Claude != "" || ciMode {
		pairs = append(pairs,
			struct{ src, dst string }{".claude", ".claude"},
		)
	}
	if tools.OpenCode != "" || ciMode {
		pairs = append(pairs,
			struct{ src, dst string }{".opencode", ".opencode"},
			struct{ src, dst string }{"AGENTS.md", "AGENTS.md"},
			struct{ src, dst string }{"opencode.json", "opencode.json"},
		)
	}

	for _, p := range pairs {
		dst := filepath.Join(cwd, p.dst)
		if err := copyFromFS(fsys, p.src, dst, force); err != nil {
			log("✗ " + p.dst + ": " + err.Error())
		} else {
			if force {
				log("↺ " + p.dst + " (updated)")
			} else {
				log("✓ " + p.dst)
			}
		}
	}

	// Write project.config.yaml if not present
	cfgPath := filepath.Join(cwd, "project.config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		projectName := filepath.Base(cwd)
		cfg := projectConfigYAML(projectName)
		if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
			log("✗ project.config.yaml: " + err.Error())
		} else {
			log("✓ project.config.yaml")
		}
	} else {
		log("~ project.config.yaml (already exists, skipped)")
	}

	// Wire lefthook
	if tools.Lefthook != "" {
		if out, err := runWithTimeout(timeoutLocal, nil, tools.Lefthook, "install"); err != nil {
			log("✗ lefthook install: " + strings.TrimSpace(string(out)))
		} else {
			log("✓ lefthook hooks wired")
		}
	}

	// Install + initialize RTK token optimizer (60-90% token savings via PreToolUse hook)
	rtkPath := tools.RTK
	if rtkPath == "" {
		log("installing rtk token optimizer…")
		if installed, err := installRTK(); err != nil {
			log("~ rtk install failed: " + err.Error())
			log("  install manually: https://github.com/rtk-ai/rtk")
		} else {
			rtkPath = installed
			log("✓ rtk installed → " + installed)
		}
	}
	if rtkPath != "" {
		// -g wires the hook into ~/.claude/settings.json (global, all projects).
		// --auto-patch skips the interactive prompt. Idempotent — safe to re-run on update.
		if out, err := runWithTimeout(timeoutNetwork, nil, rtkPath, "init", "-g", "--auto-patch"); err != nil {
			log("~ rtk init: " + strings.TrimSpace(string(out)))
		} else {
			log("✓ rtk hook wired (global ~/.claude/settings.json)")
		}
		// Wire OpenCode too if the plugin isn't already present.
		if out, err := runWithTimeout(timeoutNetwork, nil, rtkPath, "init", "-g", "--auto-patch", "--opencode"); err != nil {
			log("~ rtk opencode: " + strings.TrimSpace(string(out)))
		} else {
			log("✓ rtk opencode plugin wired")
		}
		// Confirm hook status.
		if out, err := runWithTimeout(timeoutLocal, nil, rtkPath, "init", "--show"); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "[ok]") || strings.Contains(line, "[warn]") || strings.Contains(line, "[--]") {
					log("  " + strings.TrimSpace(line))
				}
			}
		}
		log("  run 'maple rtk-audit' after a session to see token savings")
	}

	// Taffy workflows ship inside the template — nothing to install separately.
	log("✓ taffy workflows ready (.claude/taffy/, .opencode/taffy/, .cursor/taffy/)")
	log("✓ humanizer skill ready (remove AI-isms from documentation/prose)")

	return logs, nil
}

func copyFromFS(fsys fs.FS, src, dst string, force bool) error {
	info, err := fs.Stat(fsys, src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fs.WalkDir(fsys, src, func(fspath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := pathRel(src, fspath)
			if err != nil {
				return err
			}
			target := filepath.Join(dst, filepath.FromSlash(rel))
			if d.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			return writeFromFS(fsys, fspath, target, force)
		})
	}
	return writeFromFS(fsys, src, dst, force)
}

// pathRel computes the relative path using forward-slash conventions (fs.FS paths).
func pathRel(base, target string) (string, error) {
	if base == target {
		return ".", nil
	}
	prefix := base + "/"
	if !strings.HasPrefix(target, prefix) {
		return "", fmt.Errorf("%q not under %q", target, base)
	}
	return target[len(prefix):], nil
}

func writeFromFS(fsys fs.FS, src, dst string, force bool) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(dst); err == nil {
			return nil // skip existing
		}
	}
	r, err := fsys.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// .sh files need execute permission (embed strips it)
	mode := os.FileMode(0644)
	if strings.HasSuffix(src, ".sh") {
		mode = 0755
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(f, r); err != nil {
		return err
	}
	if strings.HasSuffix(src, ".sh") {
		_ = os.Chmod(dst, 0755)
	}
	return nil
}

// patchMakefile updates only the MAPLE-managed section of the user's Makefile.
// It reads the template Makefile to extract the section between the MAPLE markers,
// then replaces that section in the user's existing file (or appends it if absent).
func patchMakefile(fsys fs.FS, dstPath string) error {
	const beginMarker = "# ─── BEGIN MAPLE MANAGED"
	const endMarker = "# ─── END MAPLE MANAGED"

	// Read template MAPLE section
	tmplBytes, err := fs.ReadFile(fsys, "Makefile")
	if err != nil {
		return fmt.Errorf("read template Makefile: %w", err)
	}
	tmpl := string(tmplBytes)
	tStart := strings.Index(tmpl, beginMarker)
	tEnd := strings.Index(tmpl, endMarker)
	if tStart < 0 || tEnd < 0 {
		return fmt.Errorf("MAPLE markers not found in template Makefile")
	}
	mapleSection := tmpl[tStart : tEnd+len(endMarker)]

	// Read existing user Makefile (if any)
	existing, err := os.ReadFile(dstPath)
	if os.IsNotExist(err) {
		// First time — write the full template
		return os.WriteFile(dstPath, tmplBytes, 0644)
	}
	if err != nil {
		return err
	}
	cur := string(existing)

	eStart := strings.Index(cur, beginMarker)
	eEnd := strings.Index(cur, endMarker)
	if eStart >= 0 && eEnd >= 0 {
		// Replace existing MAPLE section
		cur = cur[:eStart] + mapleSection + cur[eEnd+len(endMarker):]
	} else {
		// Append MAPLE section (not present yet)
		cur = strings.TrimRight(cur, "\n") + "\n\n" + mapleSection + "\n"
	}
	return os.WriteFile(dstPath, []byte(cur), 0644)
}

func projectConfigYAML(name string) string {
	return fmt.Sprintf(`project:
  name: "%s"
  created_at: "%s"

sdlc:
  mode: standard          # standard | spike
  require_adr_for:
    - new_dependency
    - cross_boundary_change
    - data_model_change
    - auth_change
    - mcp_adoption
    - visual_identity_change

qa:
  bdd: cucumber           # cucumber | behave | none

design:
  target: web             # web | tui — UI medium the design phase targets
  ui_library: none        # mantine | tailwind | shadcn | none

github:
  project_number: null    # null = ask once; 0 = declined; N = configured board (maple project bootstrap)
  project_node_id: null    # Set by: maple project bootstrap
  status_field_id: null    # Status single-select field id; cached by gh-projects / maple project bootstrap
  milestone_granularity: null  # null = ask once; none = declined; minor = major+minor; patch = also per-patch
  # Issue label taxonomy (type:bug | type:feature | type:docs | type:refactor | type:chore)
  # lives in the gh-labels-milestones skill, not here. See "Version & Issue Tracking" in CLAUDE.md.
`, name, time.Now().Format(time.RFC3339))
}
