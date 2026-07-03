// Package splash renders the maple splash screen shown before the dashboard. It
// composes the brand wordmark, tagline, and version, centered in the viewport.
package splash

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kinncj/maple/app/internal/tui/brand"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

// Render centers the maple leaf, wordmark, tagline, and version within a
// width×height area using the theme mode.
func Render(width, height int, version string, mode theme.Mode) string {
	body := lipgloss.JoinVertical(lipgloss.Center,
		mode.Role("leaf").Style().Render(brand.Leaf),
		"",
		brand.Render(mode),
		"",
		mode.Role("subtitle").Style().Render(brand.Tagline),
		mode.Role("faint").Style().Render(version),
	)
	if width <= 0 || height <= 0 {
		return body
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}
