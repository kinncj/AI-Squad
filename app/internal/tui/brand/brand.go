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

// Tagline is the short brand line shown under the wordmark.
const Tagline = "the orchard for AI-built software"

// Wordmark returns the raw multi-line ASCII wordmark, trailing newlines trimmed.
func Wordmark() string { return strings.TrimRight(wordmark, "\n") }

// Render styles the wordmark with the theme's leaf role.
func Render(mode theme.Mode) string {
	return mode.Role("leaf").Style().Render(Wordmark())
}
