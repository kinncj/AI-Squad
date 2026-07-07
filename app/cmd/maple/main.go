// Command maple is the rebuilt MAPLE TUI + CLI.
//
// With no arguments it runs the dashboard TUI. Subcommands:
//
//	maple init [--force]   scaffold the MAPLE template into the current project
//	maple version          print the version
//
// Developed on feature/better-ui-ux; see docs/adrs/ADR-002 and the rework spec.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/kinncj/maple/app/internal/gh"
	"github.com/kinncj/maple/app/internal/portalsock"
	"github.com/kinncj/maple/app/internal/resume"
	"github.com/kinncj/maple/app/internal/scaffold"
	"github.com/kinncj/maple/app/internal/selfupdate"
	"github.com/kinncj/maple/app/internal/state"
	"github.com/kinncj/maple/app/internal/tui/brand"
	"github.com/kinncj/maple/app/internal/tui/dashboard"
	"github.com/kinncj/maple/app/internal/tui/menu"
	reqtui "github.com/kinncj/maple/app/internal/tui/req"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI()
		return
	}
	switch args[0] {
	case "init":
		runInit(hasFlag(args[1:], "--force"))
	case "req":
		if err := reqtui.Run(); err != nil {
			die(err)
		}
	case "update":
		runUpdate(hasFlag(args[1:], "--yes") || hasFlag(args[1:], "-y"), hasFlag(args[1:], "--diff"))
	case "labels":
		if err := gh.RunLabels(); err != nil {
			die(err)
		}
	case "project":
		if err := gh.RunProject(); err != nil {
			die(err)
		}
	case "self-update", "upgrade":
		if err := selfupdate.Run(version); err != nil {
			die(err)
		}
	case "resume-session", "resume":
		harness := ""
		if len(args) > 1 {
			harness = args[1]
		}
		if err := resume.Session(harness); err != nil {
			die(err)
		}
	case "rtk-audit":
		runRTKAudit()
	case "emit":
		runEmit(args[1:])
	case "version", "--version", "-v":
		fmt.Println("maple", version)
	case "help", "--help", "-h":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "maple: unknown command %q\n\n", args[0])
		usage(os.Stderr)
		os.Exit(2)
	}
}

// runTUI is the no-argument entry point. Without a TTY it prints usage. When the
// project is already set up it runs the dashboard loop; otherwise it shows the
// interactive setup menu (Init / Requirements / Labels / Project / Help) — the old
// binary's behaviour — and enters the dashboard once initialised.
func runTUI() {
	if !isTTY() {
		usage(os.Stdout)
		return
	}
	// Make maple a mini agentic IDE: if we're not already in a multiplexer, relaunch
	// inside a configured tmux session so harnesses open as side panes. Opt out with
	// MAPLE_NO_TMUX=1 (then harnesses run in-terminal / suspend).
	if !inMultiplexer() && os.Getenv("MAPLE_NO_TMUX") == "" {
		if wrapInTmux() {
			return
		}
	}
	if initialized() {
		runDashboardLoop()
		return
	}
	runSetupMenu()
}

// inMultiplexer reports whether maple is already running inside a terminal multiplexer
// or a splittable terminal it can drive.
func inMultiplexer() bool {
	for _, k := range []string{"TMUX", "ZELLIJ", "WEZTERM_PANE", "KITTY_WINDOW_ID"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// wrapInTmux relaunches maple inside a fresh, maple-styled tmux session (side-pane
// borders, a status bar with pane-jump hints) and attaches to it. Returns false (so the
// caller proceeds normally) when tmux isn't available or the session can't be created.
func wrapInTmux() bool {
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		return false
	}
	self, err := os.Executable()
	if err != nil {
		return false
	}
	const session = "maple"
	_ = exec.Command(tmux, "kill-session", "-t", session).Run() // clear a stale one
	if err := exec.Command(tmux, "new-session", "-d", "-s", session, self).Run(); err != nil {
		return false
	}
	configureMapleTmux(tmux, session)
	fmt.Println(brand.Leaf + " starting maple in tmux — harnesses open in a side pane · C-b ←/→ to switch")
	attach := exec.Command(tmux, "attach", "-t", session)
	attach.Stdin, attach.Stdout, attach.Stderr = os.Stdin, os.Stdout, os.Stderr
	_ = attach.Run()
	return true
}

// configureMapleTmux styles the maple session: pane-border titles (so you can see
// which pane is maple vs a harness), mouse on, and a status bar showing how to switch
// panes. Scoped to the maple session — never touches the user's global tmux config.
func configureMapleTmux(tmux, s string) {
	opts := [][]string{
		{"set-option", "-t", s, "mouse", "on"},
		// Keep a pane visible if its harness exits/errors, so the output is readable
		// instead of the pane vanishing (close it with the tmux kill-pane binding).
		{"set-option", "-t", s, "-w", "remain-on-exit", "on"},
		{"set-option", "-t", s, "-w", "pane-border-status", "top"},
		{"set-option", "-t", s, "-w", "pane-border-format", " #{?pane_active,#[bold]▸ ,}#{pane_title} "},
		{"set-option", "-t", s, "status-style", "bg=default"},
		{"set-option", "-t", s, "status-left", "#[bold] 🍁 maple #[default]"},
		{"set-option", "-t", s, "status-left-length", "24"},
		{"set-option", "-t", s, "status-right", "#[dim]C-b ←/→ switch · C-b z zoom · C-b d detach "},
		{"set-option", "-t", s, "status-right-length", "60"},
		{"select-pane", "-t", s, "-T", "maple"},
	}
	for _, o := range opts {
		_ = exec.Command(tmux, o...).Run()
	}
}

// runSetupMenu loops the setup menu until the project is initialised (then hands off
// to the dashboard) or the user quits.
func runSetupMenu() {
	for {
		action, err := menu.Run(version)
		if err != nil {
			die(err)
		}
		switch action {
		case menu.ActionQuit:
			return
		case menu.ActionInit:
			runInit(false)
			if initialized() {
				runDashboardLoop()
				return
			}
		case menu.ActionUpdate:
			runUpdate(false, false)
		case menu.ActionReq:
			if err := reqtui.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "req:", err)
			}
		case menu.ActionLabels:
			if err := gh.RunLabels(); err != nil {
				fmt.Fprintln(os.Stderr, "labels:", err)
			}
		case menu.ActionProject:
			if err := gh.RunProject(); err != nil {
				fmt.Fprintln(os.Stderr, "project:", err)
			}
		default:
			return
		}
	}
}

// initialized reports whether the current directory has a MAPLE project config.
func initialized() bool {
	_, err := os.Stat("project.config.yaml")
	return err == nil
}

// startDesignPortal picks a free port and starts the design-review portal for the
// lifetime of the session. It returns the authoritative URL (the port maple assigned,
// so the header always matches the server maple actually started) and a stop func.
// The URL is "" when the portal can't be started (no script, no free port, or the
// script failed — e.g. python3 missing). Honours MAPLE_DESIGN_PORT.
func startDesignPortal() (string, func()) {
	noop := func() {}
	script := "scripts/design-review-portal.sh"
	if _, err := os.Stat(script); err != nil {
		return "", noop
	}
	port := findFreePort()
	if port == 0 {
		return "", noop
	}
	// The script prints the URL on success (and exits non-zero on failure, having
	// removed its url/pid files). Trust its exit status over any stale url file.
	out, err := exec.Command("bash", script, "start", strconv.Itoa(port)).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "maple: design portal failed to start: %v\n%s", err, out)
		return "", noop
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	return url, func() { _ = exec.Command("bash", script, "stop").Run() }
}

// startHeartbeat writes .claude/state/maple-alive with the current time every 2s and
// removes it when the session ends, so the design portal can tell whether maple is
// still connected (and show "maple offline" if it was closed/quit). Returns a stop
// func that removes the file.
func startHeartbeat() func() {
	const path = ".claude/state/maple-alive"
	beat := func() {
		_ = os.MkdirAll(".claude/state", 0o755)
		_ = os.WriteFile(path, []byte(nowRFC3339()+"\n"), 0o644)
	}
	beat()
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				beat()
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			_ = os.Remove(path)
		})
	}
}

// findFreePort returns a free TCP port in [7800,7900), or the MAPLE_DESIGN_PORT
// override, or 0 if none is available.
func findFreePort() int {
	if v := os.Getenv("MAPLE_DESIGN_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			return p
		}
	}
	for port := 7800; port < 7900; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

// runDashboardLoop runs the dashboard, and when it quits requesting a follow-up
// workflow (req/update/labels/project) runs that in-process and re-enters. Harness
// launches never come through here — they run in the current terminal via ExecProcess.
func runDashboardLoop() {
	portalURL, stopPortal := startDesignPortal()
	stopBeat := startHeartbeat()
	stopHold := portalsock.Hold(".") // live connectivity for the portal
	cleanup := func() { stopHold(); stopBeat(); stopPortal() }
	defer cleanup()
	// Also clean up if the terminal goes away (SIGHUP) or we're terminated, so the
	// portal server isn't orphaned and the portal sees maple go offline immediately.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigCh
		cleanup()
		os.Exit(0)
	}()
	defer signal.Stop(sigCh)
	for {
		model, err := dashboard.New(version, state.NewFS("."), portalURL)
		if err != nil {
			die(err)
		}
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
		final, err := p.Run()
		if err != nil {
			die(err)
		}
		action := dashboard.ExitNone
		if dm, ok := final.(dashboard.Model); ok {
			action = dm.Action()
		}
		switch action {
		case dashboard.ExitReq:
			if err := reqtui.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "req:", err)
			}
		case dashboard.ExitUpdate:
			runUpdate(false, false)
		case dashboard.ExitLabels:
			if err := gh.RunLabels(); err != nil {
				fmt.Fprintln(os.Stderr, "labels:", err)
			}
		case dashboard.ExitProject:
			if err := gh.RunProject(); err != nil {
				fmt.Fprintln(os.Stderr, "project:", err)
			}
		default:
			return
		}
	}
}

// isTTY reports whether stdin is an interactive terminal (not a pipe or /dev/null).
func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// templateFS returns the embedded MAPLE template tree.
func templateFS() fs.FS {
	tpl, err := fs.Sub(embeddedTemplate, "template")
	if err != nil {
		die(err)
	}
	return tpl
}

func runInit(force bool) {
	cwd, err := os.Getwd()
	if err != nil {
		die(err)
	}
	written, err := scaffold.Run(templateFS(), cwd, force, nowRFC3339())
	if err != nil {
		die(err)
	}
	for _, w := range written {
		fmt.Println("  ✓", w)
	}
	fmt.Printf("maple: initialised %d file(s) in %s\n", len(written), cwd)
}

// runUpdate previews the files an update would add/replace/patch, shows a diff on
// request, and applies only after the user confirms. --yes applies without prompting;
// --diff prints every diff up front.
func runUpdate(autoYes, showDiff bool) {
	cwd, err := os.Getwd()
	if err != nil {
		die(err)
	}
	if !initialized() {
		fmt.Println("maple: no project here — run `maple init` first")
		return
	}
	plan, err := scaffold.PlanUpdate(templateFS(), cwd)
	if err != nil {
		die(err)
	}
	if plan.Empty() {
		fmt.Println(brand.Leaf + " maple: already up to date — nothing to change")
		return
	}
	printUpdatePlan(plan)
	if showDiff {
		printUpdateDiffs(plan)
	}

	if autoYes {
		applyUpdate(plan, cwd)
		return
	}
	if !isTTY() {
		fmt.Println("\nnot a TTY — re-run in a terminal, or `maple update --yes` to apply.")
		return
	}
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nApply update? [y]es / [n]o / [d]iff: ")
		line, _ := in.ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
			applyUpdate(plan, cwd)
			return
		case "d", "diff":
			printUpdateDiffs(plan)
		case "n", "no", "":
			fmt.Println("update cancelled — nothing changed")
			return
		}
	}
}

func applyUpdate(plan *scaffold.Plan, cwd string) {
	written, err := plan.Apply(cwd)
	if err != nil {
		die(err)
	}
	fmt.Printf("✓ updated %d file(s)\n", len(written))
}

// printUpdatePlan lists the planned changes grouped by kind, with counts.
func printUpdatePlan(plan *scaffold.Plan) {
	added, changed, patched := plan.Summary()
	fmt.Printf("%s maple update — %d add · %d replace · %d patch\n\n", brand.Leaf, added, changed, patched)
	for _, c := range plan.Changes {
		fmt.Printf("  %-8s %s\n", c.Kind.String(), c.Path)
	}
	fmt.Println("\nproject.config.yaml and your Makefile targets are preserved.")
}

// printUpdateDiffs shows a unified diff per changed/patched file via the system diff.
func printUpdateDiffs(plan *scaffold.Plan) {
	for _, c := range plan.Changes {
		if c.Kind == scaffold.Added {
			fmt.Printf("\n=== %s (new file, %d bytes) ===\n", c.Path, len(c.New))
			continue
		}
		fmt.Printf("\n=== %s (%s) ===\n", c.Path, c.Kind.String())
		fmt.Print(unifiedDiff(c.Path, c.Old, c.New))
	}
}

// unifiedDiff renders a `diff -u` between old and new via the system diff tool, or a
// terse note when diff is unavailable.
func unifiedDiff(path string, oldB, newB []byte) string {
	a, err1 := os.CreateTemp("", "maple-old-*")
	b, err2 := os.CreateTemp("", "maple-new-*")
	if err1 != nil || err2 != nil {
		return "  (diff unavailable)\n"
	}
	defer os.Remove(a.Name())
	defer os.Remove(b.Name())
	a.Write(oldB)
	a.Close()
	b.Write(newB)
	b.Close()
	out, err := exec.Command("diff", "-u", "--label", "current/"+path, "--label", "new/"+path, a.Name(), b.Name()).CombinedOutput()
	if err != nil && len(out) == 0 {
		return "  (diff tool unavailable)\n"
	}
	return string(out)
}

func usage(w *os.File) {
	fmt.Fprint(w, `usage: maple [command]

  (no command)      run the dashboard TUI
  init [--force]    scaffold the MAPLE template into the current project
  req               turn a free-text requirement into Gherkin stories (needs an AI harness)
  update [--yes]    preview + apply template changes (Makefile section-patched)
  labels            bootstrap the canonical MAPLE label set (needs gh)
  project           create a GitHub Project v2 and wire project.config.yaml (needs gh)
  resume [harness]  re-launch a pinned session (claude/copilot/opencode/cursor)
  self-update       replace the binary with the latest GitHub release
  rtk-audit         show RTK hook wiring and token savings
  emit <event…>     push a live event to the design portal (agents: progress/stage)
  version           print the version
`)
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// runEmit pushes a real-time event to the design portal's control socket, so an agent
// can surface progress live. Accepts a raw JSON object, or `<event> [key=value …]`.
func runEmit(args []string) {
	if len(args) == 0 {
		die(fmt.Errorf("usage: maple emit <event> [key=value …]  |  maple emit '<json>'"))
	}
	ev := map[string]any{}
	if strings.HasPrefix(strings.TrimSpace(args[0]), "{") {
		if err := json.Unmarshal([]byte(strings.Join(args, " ")), &ev); err != nil {
			die(fmt.Errorf("emit: invalid JSON: %w", err))
		}
	} else {
		ev["event"] = args[0]
		for _, kv := range args[1:] {
			if k, v, ok := strings.Cut(kv, "="); ok {
				ev[k] = v
			}
		}
	}
	if _, ok := ev["ts"]; !ok {
		ev["ts"] = nowRFC3339()
	}
	if err := portalsock.Emit(".", ev); err != nil {
		die(fmt.Errorf("emit: %w (is the design portal running?)", err))
	}
	fmt.Printf("emitted %v\n", ev["event"])
}

// runRTKAudit shows RTK hook wiring, token savings, and the audit log.
func runRTKAudit() {
	rtkPath, err := exec.LookPath("rtk")
	if err != nil {
		die(fmt.Errorf("rtk not found — install with: maple init"))
	}
	for _, sub := range [][]string{{"init", "--show"}, {"gain"}, {"hook-audit"}} {
		cmd := exec.Command(rtkPath, sub...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		fmt.Println()
	}
}

func nowRFC3339() string { return time.Now().Format(time.RFC3339) }

func die(err error) {
	fmt.Fprintln(os.Stderr, "maple:", err)
	os.Exit(1)
}
