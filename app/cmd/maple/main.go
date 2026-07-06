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
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/kinncj/maple/app/internal/gh"
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

// runDashboardLoop runs the dashboard, and when it quits requesting a follow-up
// workflow (req/update/labels/project) runs that in-process and re-enters. Harness
// launches never come through here — they spawn a new terminal and never quit.
func runDashboardLoop() {
	for {
		model, err := dashboard.New(version, state.NewFS("."))
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
