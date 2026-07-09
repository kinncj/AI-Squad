// Package brand holds the maple wordmark and brand constants for the TUI, styled
// through the theme. It has no Bubble Tea dependency.
package brand

import (
	_ "embed"
	"strings"

	"github.com/kinncj/maple/app/internal/tui/theme"
)

//go:embed wordmark.txt
var wordmark string

// Leaf is the maple brand glyph.
const Leaf = "🍁"

// Tagline is the maple manifesto line shown in the header (matches the OG TUI).
const Tagline = "(M)ulti-Agent · (A)rtifact-Driven · (P)hase-Gated · (L)ocal-First · (E)nforced."

// LogoRows is the compact 4-line MAPLE wordmark with maple leaves, used in the
// dashboard header (matches the OG tui/logo_anim.go logoCompactRows).
var LogoRows = []string{
	"🍁 ▗▖  ▗▖ ▗▄▖ ▗▄▄▖ ▗▖   ▗▄▄▄▖ 🍁",
	"🍁 ▐▛▚▞▜▌▐▌ ▐▌▐▌ ▐▌▐▌   ▐▌    🍁",
	"🍁 ▐▌  ▐▌▐▛▀▜▌▐▛▀▘ ▐▌   ▐▛▀▀▘ 🍁",
	"🍁 ▐▌  ▐▌▐▌ ▐▌▐▌   ▐▙▄▄▖▐▙▄▄▖ 🍁",
}

// Logo returns the 4-line wordmark styled with the theme's leaf role, with the
// tagline appended (muted) to the last row.
func Logo(mode theme.Mode) string {
	leaf := mode.Role("leaf").Style()
	faint := mode.Role("faint").Style()
	rows := make([]string, len(LogoRows))
	for i, r := range LogoRows {
		rows[i] = leaf.Render(r)
	}
	rows[len(rows)-1] += "  " + faint.Render(Tagline)
	return strings.Join(rows, "\n")
}

// Wordmark returns the raw multi-line ASCII wordmark, trailing newlines trimmed.
func Wordmark() string { return strings.TrimRight(wordmark, "\n") }

// Render styles the wordmark with the theme's leaf role (used by the splash).
func Render(mode theme.Mode) string {
	return mode.Role("leaf").Style().Render(Wordmark())
}
