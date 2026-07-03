// Package pane is the single focus + scroll primitive for the maple TUI. A Pane is
// one focusable, scrollable region; a Group (see group.go) owns focus, two-level
// scroll, and mouse hit-testing across several panes. It depends only on theme and
// render — never on Bubble Tea or the dashboard — so it is unit-testable in
// isolation and every surface shares one tested implementation.
//
// See docs/adrs/ADR-003-tui-focus-scroll-pane-primitive.md.
package pane

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kinncj/maple/app/internal/tui/render"
	"github.com/kinncj/maple/app/internal/tui/theme"
)

// Source is the minimum a pane needs: the rows to display. Rows are already
// filtered and sorted by the source; the pane only windows and highlights them.
type Source interface {
	Rows() []string
}

// Selectable marks a source whose rows can be individually selected. Its presence
// switches the pane from scroll-only to cursor mode.
type Selectable interface {
	Source
	RowCount() int
}

// Filterable marks a source that accepts a free-text filter query.
type Filterable interface {
	SetFilter(query string)
}

// Sortable marks a source that can be re-sorted by a named key.
type Sortable interface {
	SortKeys() []string
	SetSort(key string)
}

// Rect is the on-screen rectangle a pane last rendered into, used for mouse
// hit-testing by Group.
type Rect struct{ X, Y, W, H int }

// Contains reports whether the point (x,y) falls inside the rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Pane is one focusable, scrollable region backed by a Source.
type Pane struct {
	Title string

	src     Source
	offset  int // index of the first visible row
	sel     int // selection index; -1 when the source is not Selectable
	filter  string
	sortIdx int
	focused bool

	contentH int  // visible content rows at last render
	rect     Rect // on-screen rectangle at last render
}

// New builds a pane for the given source. If the source is Selectable, the pane
// starts in cursor mode with row 0 selected.
func New(title string, src Source) *Pane {
	p := &Pane{Title: title, src: src, sel: -1}
	if _, ok := src.(Selectable); ok {
		p.sel = 0
	}
	return p
}

func (p *Pane) rowCount() int    { return len(p.src.Rows()) }
func (p *Pane) selectable() bool { _, ok := p.src.(Selectable); return ok }

func (p *Pane) maxOffset(visible int) int {
	m := p.rowCount() - visible
	if m < 0 {
		return 0
	}
	return m
}

// ScrollBy moves the viewport by delta rows, clamped to the content. visible is the
// number of content rows currently on screen.
func (p *Pane) ScrollBy(delta, visible int) {
	p.offset = clamp(p.offset+delta, 0, p.maxOffset(visible))
}

// SelectBy moves the selection by delta rows and scrolls so the selection stays on
// screen. On a non-selectable source it degrades to ScrollBy.
func (p *Pane) SelectBy(delta, visible int) {
	if !p.selectable() {
		p.ScrollBy(delta, visible)
		return
	}
	n := p.rowCount()
	if n == 0 {
		p.sel, p.offset = 0, 0
		return
	}
	p.sel = clamp(p.sel+delta, 0, n-1)
	switch {
	case p.sel < p.offset:
		p.offset = p.sel
	case visible > 0 && p.sel >= p.offset+visible:
		p.offset = p.sel - visible + 1
	}
	p.offset = clamp(p.offset, 0, p.maxOffset(visible))
}

// Top jumps to the first row.
func (p *Pane) Top() {
	p.offset = 0
	if p.selectable() {
		p.sel = 0
	}
}

// Bottom jumps to the last row. visible is the on-screen content height.
func (p *Pane) Bottom(visible int) {
	p.offset = p.maxOffset(visible)
	if p.selectable() {
		p.sel = max(0, p.rowCount()-1)
	}
}

// SetFilter records the filter query, forwards it to a Filterable source, and
// resets scroll/selection to the top of the new result set.
func (p *Pane) SetFilter(query string) {
	p.filter = query
	if f, ok := p.src.(Filterable); ok {
		f.SetFilter(query)
	}
	p.offset = 0
	if p.selectable() {
		p.sel = 0
	}
}

// Filter returns the current filter query.
func (p *Pane) Filter() string { return p.filter }

// CycleSort advances to the next sort key on a Sortable source. No-op otherwise.
func (p *Pane) CycleSort() {
	s, ok := p.src.(Sortable)
	if !ok {
		return
	}
	keys := s.SortKeys()
	if len(keys) == 0 {
		return
	}
	p.sortIdx = (p.sortIdx + 1) % len(keys)
	s.SetSort(keys[p.sortIdx])
}

// SortKey returns the active sort key, or "" when the source is not Sortable.
func (p *Pane) SortKey() string {
	s, ok := p.src.(Sortable)
	if !ok {
		return ""
	}
	keys := s.SortKeys()
	if len(keys) == 0 {
		return ""
	}
	return keys[p.sortIdx%len(keys)]
}

// SetFocus toggles the focus ring.
func (p *Pane) SetFocus(b bool) { p.focused = b }

// Focused reports whether the pane holds focus.
func (p *Pane) Focused() bool { return p.focused }

// Selected returns the selection index, or -1 when the source is not Selectable.
func (p *Pane) Selected() int { return p.sel }

// Offset returns the index of the first visible row.
func (p *Pane) Offset() int { return p.offset }

// VisibleRows returns the content height captured at the last render.
func (p *Pane) VisibleRows() int { return p.contentH }

// Rect returns the on-screen rectangle from the last render.
func (p *Pane) Rect() Rect { return p.rect }

// RenderAt draws the pane into a width×height box at (x,y) using the theme mode,
// records its Rect and content height, and returns the rendered string. The box
// uses the border_focus role when focused and the border role otherwise.
func (p *Pane) RenderAt(x, y, width, height int, mode theme.Mode) string {
	p.rect = Rect{X: x, Y: y, W: width, H: height}
	if width < 2 || height < 2 {
		p.contentH = 0
		return ""
	}
	innerW := width - 2
	innerH := height - 2

	title := render.Truncate(p.titleText(), innerW)
	titleStyled := mode.Role("title").Style().Render(title)

	// Content rows sit below the title; the scroll hint (if any) takes the last row.
	contentH := innerH - 1
	if contentH < 0 {
		contentH = 0
	}
	rows := p.src.Rows()
	top, bottom := render.ScrollHint(p.offset, contentH, len(rows))
	if top != "" || bottom != "" {
		contentH--
		if contentH < 0 {
			contentH = 0
		}
		top, bottom = render.ScrollHint(p.offset, contentH, len(rows))
	}
	p.contentH = contentH

	var lines []string
	lines = append(lines, titleStyled)
	for i := 0; i < contentH; i++ {
		idx := p.offset + i
		if idx >= len(rows) {
			lines = append(lines, "")
			continue
		}
		row := render.Truncate(rows[idx], innerW)
		if p.selectable() && idx == p.sel {
			row = mode.Role("selected").Style().Render(render.PadRight(rows[idx], innerW))
			row = render.Truncate(row, innerW)
		}
		lines = append(lines, row)
	}
	if top != "" || bottom != "" {
		hint := strings.TrimSpace(top + "  " + bottom)
		lines = append(lines, mode.Role("scrollhint").Style().Render(render.Truncate(hint, innerW)))
	}

	borderRole := "border"
	if p.focused {
		borderRole = "border_focus"
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(mode.Role(borderRole))).
		Width(innerW).
		Height(innerH)
	return box.Render(strings.Join(lines, "\n"))
}

func (p *Pane) titleText() string {
	t := p.Title
	if p.filter != "" {
		t += " /" + p.filter
	}
	if k := p.SortKey(); k != "" {
		t += " ⇅" + k
	}
	return t
}

func borderColor(r theme.RoleStyle) lipgloss.Color {
	if r.FG != "" {
		return lipgloss.Color(r.FG)
	}
	return lipgloss.Color("")
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
