package pane

// Group owns an ordered set of panes plus a focus index and a page-scroll offset.
// It routes movement, filter, and sort to the focused pane, cycles focus with the
// focus ring, and hit-tests mouse events against each pane's last-rendered Rect.
// Exactly one pane is focused at a time. See docs/adrs/ADR-003.
type Group struct {
	panes      []*Pane
	focus      int
	pageOffset int
}

// NewGroup builds a group and focuses the first pane.
func NewGroup(panes ...*Pane) *Group {
	g := &Group{panes: panes}
	g.applyFocus()
	return g
}

// Panes returns the group's panes in order.
func (g *Group) Panes() []*Pane { return g.panes }

// FocusIndex returns the index of the focused pane, or -1 when the group is empty.
func (g *Group) FocusIndex() int {
	if len(g.panes) == 0 {
		return -1
	}
	return g.focus
}

// Focused returns the focused pane, or nil when the group is empty.
func (g *Group) Focused() *Pane {
	if len(g.panes) == 0 {
		return nil
	}
	return g.panes[g.focus]
}

// SetFocus focuses pane i (clamped) and redraws the ring.
func (g *Group) SetFocus(i int) {
	if len(g.panes) == 0 {
		return
	}
	g.focus = clamp(i, 0, len(g.panes)-1)
	g.applyFocus()
}

// FocusNext moves focus to the next pane, wrapping (Tab).
func (g *Group) FocusNext() {
	if len(g.panes) == 0 {
		return
	}
	g.focus = (g.focus + 1) % len(g.panes)
	g.applyFocus()
}

// FocusPrev moves focus to the previous pane, wrapping (Shift-Tab).
func (g *Group) FocusPrev() {
	if len(g.panes) == 0 {
		return
	}
	g.focus = (g.focus - 1 + len(g.panes)) % len(g.panes)
	g.applyFocus()
}

func (g *Group) applyFocus() {
	for i, p := range g.panes {
		p.SetFocus(i == g.focus)
	}
}

// PageOffset returns the page-level scroll offset (for stacked layouts taller than
// the screen).
func (g *Group) PageOffset() int { return g.pageOffset }

// ScrollPageBy adjusts the page-level scroll offset, clamped to [0, limit].
func (g *Group) ScrollPageBy(delta, limit int) {
	g.pageOffset = clamp(g.pageOffset+delta, 0, max(0, limit))
}

// --- routed movement (to the focused pane, using its last-rendered height) ---

// ScrollBy scrolls the focused pane.
func (g *Group) ScrollBy(delta int) {
	if p := g.Focused(); p != nil {
		p.ScrollBy(delta, p.VisibleRows())
	}
}

// SelectBy moves the focused pane's selection.
func (g *Group) SelectBy(delta int) {
	if p := g.Focused(); p != nil {
		p.SelectBy(delta, p.VisibleRows())
	}
}

// Top jumps the focused pane to its first row.
func (g *Group) Top() {
	if p := g.Focused(); p != nil {
		p.Top()
	}
}

// Bottom jumps the focused pane to its last row.
func (g *Group) Bottom() {
	if p := g.Focused(); p != nil {
		p.Bottom(p.VisibleRows())
	}
}

// SetFilter applies a filter to the focused pane.
func (g *Group) SetFilter(q string) {
	if p := g.Focused(); p != nil {
		p.SetFilter(q)
	}
}

// CycleSort cycles the focused pane's sort key.
func (g *Group) CycleSort() {
	if p := g.Focused(); p != nil {
		p.CycleSort()
	}
}

// --- mouse (pointer-targeted) ---

// FocusAt focuses the pane whose last-rendered Rect contains (x,y). Returns true
// when a pane was hit.
func (g *Group) FocusAt(x, y int) bool {
	for i, p := range g.panes {
		if p.Rect().Contains(x, y) {
			g.SetFocus(i)
			return true
		}
	}
	return false
}

// ScrollAt scrolls the pane under the pointer without changing focus. Returns true
// when a pane was hit.
func (g *Group) ScrollAt(x, y, delta int) bool {
	for _, p := range g.panes {
		if p.Rect().Contains(x, y) {
			p.ScrollBy(delta, p.VisibleRows())
			return true
		}
	}
	return false
}
