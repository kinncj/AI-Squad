// Package render holds framework-free rendering helpers for the maple TUI:
// display-width-aware truncation and padding, and the scroll affordance strings
// used by the pane primitive. It has no Bubble Tea dependency; it uses lipgloss
// only for display-width measurement, so it is unit-testable in isolation.
package render

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Ellipsis is appended by Truncate when content is cut.
const Ellipsis = "…"

// Width returns the terminal display width of s (wide runes count as 2).
func Width(s string) int { return lipgloss.Width(s) }

// Truncate shortens s to at most max display columns, appending an ellipsis when
// it cuts. max <= 0 yields the empty string.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if Width(s) <= max {
		return s
	}
	if max == 1 {
		return Ellipsis
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > max-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + Ellipsis
}

// PadRight pads s with spaces to exactly width display columns, truncating if s is
// wider than width.
func PadRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := Width(s)
	if w > width {
		return Truncate(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

// ScrollHint returns the top and bottom affordance captions for a scroll window.
// offset is the index of the first visible row, visible the number of rows shown,
// total the row count. top is non-empty when rows are hidden above; bottom shows the
// hidden-below count plus a "y/total" position readout. Either may be "".
func ScrollHint(offset, visible, total int) (top, bottom string) {
	if total <= 0 || visible <= 0 {
		return "", ""
	}
	if offset > 0 {
		top = "▲ " + strconv.Itoa(offset) + " more"
	}
	below := total - (offset + visible)
	if below > 0 {
		bottom = "▼ " + strconv.Itoa(below) + " more"
	}
	if top != "" || bottom != "" {
		pos := " · " + strconv.Itoa(min(offset+visible, total)) + "/" + strconv.Itoa(total)
		if bottom == "" {
			bottom = strings.TrimSpace(pos)
		} else {
			bottom += pos
		}
	}
	return top, bottom
}
