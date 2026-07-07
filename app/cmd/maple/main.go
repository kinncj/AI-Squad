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
		runInit(true)
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
	if initialized() {
		runDashboardLoop()
		return
	}
	runSetupMenu()
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
			runInit(true)
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
			runInit(true)
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

func runInit(force bool) {
	tpl, err := fs.Sub(embeddedTemplate, "template")
	if err != nil {
		die(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		die(err)
	}
	written, err := scaffold.Run(tpl, cwd, force, nowRFC3339())
	if err != nil {
		die(err)
	}
	for _, w := range written {
		fmt.Println("  ✓", w)
	}
	fmt.Printf("maple: initialised %d file(s) in %s\n", len(written), cwd)
}

func usage(w *os.File) {
	fmt.Fprint(w, `usage: maple [command]

  (no command)      run the dashboard TUI
  init [--force]    scaffold the MAPLE template into the current project
  req               turn a free-text requirement into Gherkin stories (needs an AI harness)
  update            re-copy the template over an existing project (force)
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
