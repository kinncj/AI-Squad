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
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kinncj/maple/app/internal/gh"
	"github.com/kinncj/maple/app/internal/scaffold"
	"github.com/kinncj/maple/app/internal/state"
	"github.com/kinncj/maple/app/internal/tui/dashboard"
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
	case "labels":
		if err := gh.RunLabels(); err != nil {
			die(err)
		}
	case "project":
		if err := gh.RunProject(); err != nil {
			die(err)
		}
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

func runTUI() {
	model, err := dashboard.New(version, state.NewFS("."))
	if err != nil {
		die(err)
	}
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		die(err)
	}
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

  (no command)   run the dashboard TUI
  init [--force] scaffold the MAPLE template into the current project
  labels         bootstrap the canonical MAPLE label set (needs gh)
  project        create a GitHub Project v2 and wire project.config.yaml (needs gh)
  version        print the version
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

func nowRFC3339() string { return time.Now().Format(time.RFC3339) }

func die(err error) {
	fmt.Fprintln(os.Stderr, "maple:", err)
	os.Exit(1)
}
