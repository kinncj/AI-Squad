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
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	fmt.Fprintf(os.Stdout, "maple %s — rebuild in progress (feature/better-ui-ux)\n", version)
}
