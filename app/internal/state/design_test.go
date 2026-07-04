package state

import (
	"strings"
	"testing"
)

func TestDesignTreeListsDirsAndFilesSkippingGitkeep(t *testing.T) {
	got := NewFS("testdata").DesignTree()
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, ".gitkeep") {
		t.Error(".gitkeep should be skipped")
	}
	// Expect the two subdirs and their files, with folder/file icons.
	for _, want := range []string{"📁 wireframes", "📄 home.md", "📁 identity", "📄 tokens.json"} {
		if !strings.Contains(joined, want) {
			t.Errorf("design tree missing %q; got:\n%s", want, joined)
		}
	}
	// Nested entries are indented.
	for _, line := range got {
		if strings.Contains(line, "home.md") && !strings.HasPrefix(line, "  ") {
			t.Errorf("nested file should be indented: %q", line)
		}
	}
}

func TestDesignTreeEmptyWhenAbsent(t *testing.T) {
	if got := NewFS("testdata/nope").DesignTree(); len(got) != 0 {
		t.Errorf("absent design dir should yield no entries, got %d", len(got))
	}
}

func TestFormatLogLines(t *testing.T) {
	data := `{"ts":"12:00","agent":"qa","skill":"tdd","op":"run"}
{"ts":"12:01","agent":"dev","error":"boom"}
plain non-json line`
	got := formatLogLines(data, 0)
	if len(got) != 3 {
		t.Fatalf("got %d lines, want 3", len(got))
	}
	if got[0] != "ts=12:00  agent=qa  skill=tdd  op=run" {
		t.Errorf("line 0 = %q", got[0])
	}
	if got[1] != "ts=12:01  agent=dev  error=boom" {
		t.Errorf("line 1 = %q", got[1])
	}
	if got[2] != "plain non-json line" {
		t.Errorf("non-json line should pass through, got %q", got[2])
	}
}

func TestLogLinesTail(t *testing.T) {
	// n limits to the last n lines.
	got := formatLogLines("a\nb\nc\nd\ne", 2)
	if len(got) != 2 || got[0] != "d" || got[1] != "e" {
		t.Errorf("tail(2) = %v, want [d e]", got)
	}
}

func TestLogLinesFromFixture(t *testing.T) {
	got := NewFS("testdata").LogLines(10)
	if len(got) != 3 {
		t.Errorf("fixture log has 3 lines, got %d: %v", len(got), got)
	}
}
