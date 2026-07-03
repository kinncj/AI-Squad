// Package theme loads the maple terminal theme from an embedded JSON document and
// exposes it as lipgloss styles. It is framework-light: it imports lipgloss for
// style construction but has no Bubble Tea dependency, so it can be unit-tested in
// isolation. See docs/adrs/ADR-003-tui-focus-scroll-pane-primitive.md.
package theme

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

//go:embed maple-theme.json
var themeJSON []byte

// RoleStyle is a structural role: a foreground/background plus text attributes and
// an optional decorative glyph.
type RoleStyle struct {
	FG    string   `json:"fg"`
	BG    string   `json:"bg"`
	Attrs []string `json:"attrs"`
	Glyph string   `json:"glyph"`
}

// StateStyle is a status role: a colour, a glyph, a human label, and attributes.
type StateStyle struct {
	FG    string   `json:"fg"`
	Glyph string   `json:"glyph"`
	Label string   `json:"label"`
	Attrs []string `json:"attrs"`
}

// Mode is one theme variant (dark or light).
type Mode struct {
	Structure map[string]RoleStyle  `json:"structure"`
	States    map[string]StateStyle `json:"states"`
}

// Role returns the named structural role, or the zero RoleStyle if absent.
func (m Mode) Role(name string) RoleStyle { return m.Structure[name] }

// State returns the named status role, or the zero StateStyle if absent.
func (m Mode) State(name string) StateStyle { return m.States[name] }

// Theme is the parsed maple theme.
type Theme struct {
	Name        string          `json:"name"`
	DefaultMode string          `json:"default_mode"`
	Modes       map[string]Mode `json:"modes"`
}

// Load parses the embedded maple theme.
func Load() (*Theme, error) {
	var t Theme
	if err := json.Unmarshal(themeJSON, &t); err != nil {
		return nil, fmt.Errorf("theme: parse embedded json: %w", err)
	}
	if t.Name == "" || len(t.Modes) == 0 {
		return nil, fmt.Errorf("theme: embedded json is empty or malformed")
	}
	if _, ok := t.Modes[t.DefaultMode]; !ok {
		return nil, fmt.Errorf("theme: default_mode %q not present in modes", t.DefaultMode)
	}
	return &t, nil
}

// ActiveMode returns the DefaultMode variant. Load guarantees it exists.
func (t *Theme) ActiveMode() Mode { return t.Modes[t.DefaultMode] }

// Style builds a lipgloss.Style from a structural role.
func (r RoleStyle) Style() lipgloss.Style {
	s := lipgloss.NewStyle()
	if r.FG != "" {
		s = s.Foreground(lipgloss.Color(r.FG))
	}
	if r.BG != "" {
		s = s.Background(lipgloss.Color(r.BG))
	}
	return applyAttrs(s, r.Attrs)
}

// Style builds a lipgloss.Style from a status role.
func (s StateStyle) Style() lipgloss.Style {
	st := lipgloss.NewStyle()
	if s.FG != "" {
		st = st.Foreground(lipgloss.Color(s.FG))
	}
	return applyAttrs(st, s.Attrs)
}

// Render decorates text with the state's glyph and label styling, e.g. "◐ running".
func (s StateStyle) Render(text string) string {
	glyph := s.Glyph
	if glyph != "" {
		glyph += " "
	}
	return s.Style().Render(glyph + text)
}

// applyAttrs maps the string attribute names onto a lipgloss.Style. Unknown
// attributes are ignored so the theme JSON can carry forward-compatible hints.
func applyAttrs(s lipgloss.Style, attrs []string) lipgloss.Style {
	for _, a := range attrs {
		switch a {
		case "bold":
			s = s.Bold(true)
		case "faint":
			s = s.Faint(true)
		case "italic":
			s = s.Italic(true)
		case "underline":
			s = s.Underline(true)
		case "reverse":
			s = s.Reverse(true)
		}
	}
	return s
}
