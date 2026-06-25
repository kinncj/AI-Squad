package main

import "testing"

func TestStatusGlyph(t *testing.T) {
	cases := []struct {
		x, y byte
		want string
	}{
		{' ', 'M', "M"}, {'A', ' ', "A"}, {' ', 'D', "D"},
		{'?', '?', "??"}, {'M', ' ', "M"}, {'R', ' ', "R"},
	}
	for _, c := range cases {
		if got := statusGlyph(c.x, c.y); got != c.want {
			t.Errorf("statusGlyph(%q,%q)=%q want %q", c.x, c.y, got, c.want)
		}
	}
}

func TestParsePorcelain(t *testing.T) {
	out := " M tui/dashboard.go\n?? docs/new.md\nA  tui/git_changes.go\n D old.go\nR  a.go -> b.go\n"
	files := parsePorcelain(out)
	if len(files) != 5 {
		t.Fatalf("got %d files, want 5", len(files))
	}
	if files[0].Status != "M" || files[0].Path != "tui/dashboard.go" || files[0].Staged {
		t.Errorf("file 0: %+v", files[0])
	}
	if files[1].Status != "??" || files[1].Path != "docs/new.md" || files[1].Staged {
		t.Errorf("file 1: %+v", files[1])
	}
	if files[2].Status != "A" || files[2].Path != "tui/git_changes.go" || !files[2].Staged {
		t.Errorf("file 2: %+v", files[2])
	}
	if files[3].Status != "D" || files[3].Path != "old.go" {
		t.Errorf("file 3: %+v", files[3])
	}
	if files[4].Status != "R" || files[4].Path != "b.go" {
		t.Errorf("file 4 (rename): %+v", files[4])
	}
}

func TestParsePorcelainEmpty(t *testing.T) {
	if got := parsePorcelain(""); len(got) != 0 {
		t.Errorf("empty porcelain -> %d files", len(got))
	}
}

func TestDiffLineKind(t *testing.T) {
	cases := map[string]string{
		"+added line":   "add",
		"-removed line": "del",
		"+++ b/file.go":  "meta",
		"--- a/file.go":  "meta",
		"@@ -1,2 +1,3 @@": "hunk",
		" context":       "ctx",
		"diff --git ...":  "ctx",
	}
	for line, want := range cases {
		if got := diffLineKind(line); got != want {
			t.Errorf("diffLineKind(%q)=%q want %q", line, got, want)
		}
	}
}

func TestStagedCount(t *testing.T) {
	c := gitChanges{Files: []gitFile{
		{Status: "A", Staged: true}, {Status: "M", Staged: false}, {Status: "M", Staged: true},
	}}
	if c.stagedCount() != 2 {
		t.Errorf("stagedCount=%d want 2", c.stagedCount())
	}
}
