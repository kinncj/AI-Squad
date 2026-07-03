package pane

import (
	"testing"

	"github.com/kinncj/maple/app/internal/tui/theme"
)

func newTestGroup(t *testing.T) *Group {
	t.Helper()
	a := New("A", &listSource{items: seq(20)})
	b := New("B", &listSource{items: seq(20)})
	c := New("C", &staticSource{seq(20)})
	g := NewGroup(a, b, c)
	// Render each so VisibleRows/Rect are populated at disjoint positions.
	mode := mustMode(t)
	a.RenderAt(0, 0, 20, 8, mode)
	b.RenderAt(20, 0, 20, 8, mode)
	c.RenderAt(0, 8, 20, 8, mode)
	return g
}

func mustMode(t *testing.T) theme.Mode {
	t.Helper()
	th, err := theme.Load()
	if err != nil {
		t.Fatalf("theme.Load: %v", err)
	}
	return th.ActiveMode()
}

func TestGroupFocusStartsOnFirst(t *testing.T) {
	g := newTestGroup(t)
	if g.FocusIndex() != 0 {
		t.Errorf("focus index = %d, want 0", g.FocusIndex())
	}
	if !g.Panes()[0].Focused() || g.Panes()[1].Focused() {
		t.Errorf("only pane 0 should be focused")
	}
}

func TestGroupFocusCyclesAndWraps(t *testing.T) {
	g := newTestGroup(t)
	g.FocusNext()
	if g.FocusIndex() != 1 {
		t.Errorf("after Next, focus = %d want 1", g.FocusIndex())
	}
	g.FocusNext()
	g.FocusNext() // wraps 2 -> 0
	if g.FocusIndex() != 0 {
		t.Errorf("Next should wrap to 0, got %d", g.FocusIndex())
	}
	g.FocusPrev() // wraps 0 -> 2
	if g.FocusIndex() != 2 {
		t.Errorf("Prev should wrap to 2, got %d", g.FocusIndex())
	}
	// Exactly one focused.
	focused := 0
	for _, p := range g.Panes() {
		if p.Focused() {
			focused++
		}
	}
	if focused != 1 {
		t.Errorf("exactly one pane should be focused, got %d", focused)
	}
}

func TestGroupRoutesSelectToFocusedPane(t *testing.T) {
	g := newTestGroup(t)
	g.FocusNext() // focus pane B
	g.SelectBy(3)
	if got := g.Panes()[1].Selected(); got != 3 {
		t.Errorf("focused pane B selection = %d, want 3", got)
	}
	if got := g.Panes()[0].Selected(); got != 0 {
		t.Errorf("unfocused pane A should be untouched, sel = %d", got)
	}
}

func TestGroupBottomAndTopRouteToFocused(t *testing.T) {
	g := newTestGroup(t)
	g.Bottom()
	if got := g.Focused().Selected(); got != 19 {
		t.Errorf("Bottom on focused = %d, want 19", got)
	}
	g.Top()
	if got := g.Focused().Selected(); got != 0 {
		t.Errorf("Top on focused = %d, want 0", got)
	}
}

func TestGroupFocusAtHitTest(t *testing.T) {
	g := newTestGroup(t)
	// pane B occupies x[20,40) y[0,8); a click at (25,2) focuses it.
	if !g.FocusAt(25, 2) {
		t.Fatal("click inside pane B should hit")
	}
	if g.FocusIndex() != 1 {
		t.Errorf("click in B should focus index 1, got %d", g.FocusIndex())
	}
	// A click outside every pane misses and leaves focus unchanged.
	if g.FocusAt(500, 500) {
		t.Error("click outside all panes should miss")
	}
	if g.FocusIndex() != 1 {
		t.Errorf("miss should not change focus, got %d", g.FocusIndex())
	}
}

func TestGroupScrollAtDoesNotChangeFocus(t *testing.T) {
	g := newTestGroup(t)
	// Focus stays on A (index 0); wheel over C (index 2) scrolls C only.
	if !g.ScrollAt(5, 10, 4) {
		t.Fatal("wheel over pane C should hit")
	}
	if g.FocusIndex() != 0 {
		t.Errorf("wheel must not change focus, got %d", g.FocusIndex())
	}
	if g.Panes()[2].Offset() == 0 {
		t.Errorf("pane C should have scrolled")
	}
}

func TestGroupPageScrollClamps(t *testing.T) {
	g := newTestGroup(t)
	g.ScrollPageBy(-3, 10)
	if g.PageOffset() != 0 {
		t.Errorf("page offset should clamp at 0, got %d", g.PageOffset())
	}
	g.ScrollPageBy(100, 10)
	if g.PageOffset() != 10 {
		t.Errorf("page offset should clamp at 10, got %d", g.PageOffset())
	}
}

func TestEmptyGroupIsSafe(t *testing.T) {
	g := NewGroup()
	if g.FocusIndex() != -1 || g.Focused() != nil {
		t.Errorf("empty group: index=%d focused=%v", g.FocusIndex(), g.Focused())
	}
	g.FocusNext() // must not panic
	g.SelectBy(1) // must not panic
}
