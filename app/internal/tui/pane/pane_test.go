package pane

import (
	"sort"
	"strings"
	"testing"

	"github.com/kinncj/maple/app/internal/tui/theme"
)

// staticSource implements only Source.
type staticSource struct{ rows []string }

func (s *staticSource) Rows() []string { return s.rows }

// listSource implements Source + Selectable + Filterable + Sortable.
type listSource struct {
	items   []string
	filter  string
	sortKey string
	sorts   int // count of SetSort calls, to prove routing
}

func (l *listSource) visible() []string {
	var out []string
	for _, it := range l.items {
		if l.filter == "" || strings.Contains(it, l.filter) {
			out = append(out, it)
		}
	}
	if l.sortKey == "desc" {
		sort.Sort(sort.Reverse(sort.StringSlice(out)))
	} else {
		sort.Strings(out)
	}
	return out
}
func (l *listSource) Rows() []string     { return l.visible() }
func (l *listSource) RowCount() int      { return len(l.visible()) }
func (l *listSource) SetFilter(q string) { l.filter = q }
func (l *listSource) SortKeys() []string { return []string{"asc", "desc"} }
func (l *listSource) SetSort(k string)   { l.sortKey = k; l.sorts++ }

func seq(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = string(rune('a'+i%26)) + "-row"
	}
	return out
}

func TestNewSelectionModeDependsOnSource(t *testing.T) {
	if p := New("s", &staticSource{seq(3)}); p.Selected() != -1 {
		t.Errorf("static source pane should not be selectable, sel=%d", p.Selected())
	}
	if p := New("l", &listSource{items: seq(3)}); p.Selected() != 0 {
		t.Errorf("selectable source pane should start at row 0, sel=%d", p.Selected())
	}
}

func TestScrollByClamps(t *testing.T) {
	p := New("s", &staticSource{seq(10)})
	p.ScrollBy(-5, 4) // can't go below 0
	if p.Offset() != 0 {
		t.Errorf("offset = %d, want 0", p.Offset())
	}
	p.ScrollBy(100, 4) // max offset = 10-4 = 6
	if p.Offset() != 6 {
		t.Errorf("offset = %d, want 6", p.Offset())
	}
}

func TestSelectByFollowsScroll(t *testing.T) {
	p := New("l", &listSource{items: seq(20)})
	visible := 5
	// move selection to row 12; offset must scroll so 12 is visible.
	for i := 0; i < 12; i++ {
		p.SelectBy(1, visible)
	}
	if p.Selected() != 12 {
		t.Fatalf("sel = %d, want 12", p.Selected())
	}
	if p.Selected() < p.Offset() || p.Selected() >= p.Offset()+visible {
		t.Errorf("selection %d not in viewport [%d,%d)", p.Selected(), p.Offset(), p.Offset()+visible)
	}
	// move back up; selection follows again.
	for i := 0; i < 12; i++ {
		p.SelectBy(-1, visible)
	}
	if p.Selected() != 0 || p.Offset() != 0 {
		t.Errorf("after scrolling up: sel=%d offset=%d, want 0/0", p.Selected(), p.Offset())
	}
}

func TestSelectByOnStaticSourceScrolls(t *testing.T) {
	p := New("s", &staticSource{seq(10)})
	p.SelectBy(3, 4)
	if p.Selected() != -1 {
		t.Errorf("static pane has no selection, got %d", p.Selected())
	}
	if p.Offset() != 3 {
		t.Errorf("static SelectBy should scroll, offset=%d want 3", p.Offset())
	}
}

func TestTopAndBottom(t *testing.T) {
	p := New("l", &listSource{items: seq(20)})
	p.Bottom(5)
	if p.Selected() != 19 {
		t.Errorf("Bottom sel = %d, want 19", p.Selected())
	}
	if p.Offset() != 15 {
		t.Errorf("Bottom offset = %d, want 15", p.Offset())
	}
	p.Top()
	if p.Selected() != 0 || p.Offset() != 0 {
		t.Errorf("Top sel/offset = %d/%d, want 0/0", p.Selected(), p.Offset())
	}
}

func TestSetFilterRoutesAndResets(t *testing.T) {
	src := &listSource{items: []string{"apple", "apricot", "banana", "cherry"}}
	p := New("l", src)
	p.SelectBy(3, 10) // move selection down first
	p.SetFilter("ap")
	if src.filter != "ap" {
		t.Errorf("filter not routed to source, got %q", src.filter)
	}
	if p.Filter() != "ap" {
		t.Errorf("pane filter = %q, want ap", p.Filter())
	}
	if p.Offset() != 0 || p.Selected() != 0 {
		t.Errorf("filter should reset scroll/selection, got offset=%d sel=%d", p.Offset(), p.Selected())
	}
	if got := len(src.Rows()); got != 2 {
		t.Errorf("filtered rows = %d, want 2 (apple, apricot)", got)
	}
}

func TestCycleSortRoutes(t *testing.T) {
	src := &listSource{items: seq(5)}
	p := New("l", src)
	if p.SortKey() != "asc" {
		t.Errorf("initial sort = %q, want asc", p.SortKey())
	}
	p.CycleSort()
	if p.SortKey() != "desc" || src.sortKey != "desc" {
		t.Errorf("after cycle: pane=%q src=%q, want desc/desc", p.SortKey(), src.sortKey)
	}
	p.CycleSort()
	if p.SortKey() != "asc" {
		t.Errorf("cycle should wrap to asc, got %q", p.SortKey())
	}
	if src.sorts != 2 {
		t.Errorf("SetSort called %d times, want 2", src.sorts)
	}
}

func TestCycleSortNoopOnPlainSource(t *testing.T) {
	p := New("s", &staticSource{seq(3)})
	p.CycleSort() // must not panic
	if p.SortKey() != "" {
		t.Errorf("plain source has no sort key, got %q", p.SortKey())
	}
}

func TestRenderRecordsRectAndFitsHeight(t *testing.T) {
	th, _ := theme.Load()
	mode := th.ActiveMode()
	p := New("Stories", &listSource{items: seq(30)})
	out := p.RenderAt(2, 3, 24, 8, mode)
	if out == "" {
		t.Fatal("render produced no output")
	}
	if got := p.Rect(); got.X != 2 || got.Y != 3 || got.W != 24 || got.H != 8 {
		t.Errorf("rect = %+v, want {2 3 24 8}", got)
	}
	if p.Rect().Contains(1, 3) {
		t.Errorf("rect should not contain a point left of X")
	}
	if !p.Rect().Contains(3, 4) {
		t.Errorf("rect should contain an interior point")
	}
	if lines := strings.Count(out, "\n") + 1; lines != 8 {
		t.Errorf("rendered %d lines, want 8 (box height)", lines)
	}
}

func TestRenderTinyBoxIsSafe(t *testing.T) {
	th, _ := theme.Load()
	p := New("x", &listSource{items: seq(3)})
	if out := p.RenderAt(0, 0, 1, 1, th.ActiveMode()); out != "" {
		t.Errorf("sub-minimal box should render empty, got %q", out)
	}
	if p.VisibleRows() != 0 {
		t.Errorf("tiny box visible rows = %d, want 0", p.VisibleRows())
	}
}
