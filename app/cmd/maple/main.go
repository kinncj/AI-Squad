// Command maple is the rebuilt MAPLE TUI.
//
// It is developed in parallel with the shipping tui/ binary on the
// feature/better-ui-ux branch and does not replace it until the rebuild reaches
// feature parity. See docs/adrs/ADR-002-tui-businessrepo-rebuild.md and
// docs/specs/2026-07-03-tui-rework-design.md.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kinncj/maple/app/internal/tui/dashboard"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	model, err := dashboard.New(version)
	if err != nil {
		fmt.Fprintln(os.Stderr, "maple:", err)
		os.Exit(1)
	}
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "maple:", err)
		os.Exit(1)
	}
}
