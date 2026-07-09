package render

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 1, "…"},
		{"hello", 0, ""},
		{"hello", -3, ""},
	}
	for _, c := range cases {
		if got := Truncate(c.s, c.max); got != c.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", c.s, c.max, got, c.want)
		}
	}
}

func TestTruncateNeverExceedsWidth(t *testing.T) {
	for max := 1; max <= 8; max++ {
		got := Truncate("abcdefghij", max)
		if Width(got) > max {
			t.Errorf("Truncate width %d exceeds max %d (%q)", Width(got), max, got)
		}
	}
}

func TestPadRight(t *testing.T) {
	if got := PadRight("hi", 5); got != "hi   " {
		t.Errorf("PadRight = %q, want %q", got, "hi   ")
	}
	if got := PadRight("hi", 2); got != "hi" {
		t.Errorf("PadRight exact = %q", got)
	}
	if got := PadRight("hello", 3); Width(got) != 3 {
		t.Errorf("PadRight truncates over-wide to width 3, got width %d (%q)", Width(got), got)
	}
	if got := PadRight("x", 0); got != "" {
		t.Errorf("PadRight width 0 = %q, want empty", got)
	}
}

func TestScrollHintNoScrollWhenAllVisible(t *testing.T) {
	top, bottom := ScrollHint(0, 10, 8)
	if top != "" || bottom != "" {
		t.Errorf("all visible should have no hints, got top=%q bottom=%q", top, bottom)
	}
}

func TestScrollHintTopAndBottom(t *testing.T) {
	// 100 rows, showing rows [20,30): 20 hidden above, 70 hidden below.
	top, bottom := ScrollHint(20, 10, 100)
	if top != "▲ 20 more" {
		t.Errorf("top = %q, want %q", top, "▲ 20 more")
	}
	if bottom != "▼ 70 more · 30/100" {
		t.Errorf("bottom = %q, want %q", bottom, "▼ 70 more · 30/100")
	}
}

func TestScrollHintBottomOnlyShowsPosition(t *testing.T) {
	// At the top: nothing hidden above, some below.
	top, bottom := ScrollHint(0, 10, 25)
	if top != "" {
		t.Errorf("top should be empty at offset 0, got %q", top)
	}
	if bottom != "▼ 15 more · 10/25" {
		t.Errorf("bottom = %q, want %q", bottom, "▼ 15 more · 10/25")
	}
}

func TestScrollHintAtBottomShowsPositionOnly(t *testing.T) {
	// Scrolled to the end: hidden above, nothing below.
	top, bottom := ScrollHint(15, 10, 25)
	if top != "▲ 15 more" {
		t.Errorf("top = %q, want %q", top, "▲ 15 more")
	}
	if bottom != "· 25/25" {
		t.Errorf("bottom = %q, want %q", bottom, "· 25/25")
	}
}

func TestScrollHintEmptyContent(t *testing.T) {
	top, bottom := ScrollHint(0, 10, 0)
	if top != "" || bottom != "" {
		t.Errorf("empty content should have no hints, got top=%q bottom=%q", top, bottom)
	}
}
