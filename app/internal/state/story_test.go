package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoriesReadsSortsAndSkipsPartials(t *testing.T) {
	got := NewFS("testdata").Stories()
	if len(got) != 2 {
		t.Fatalf("Stories() returned %d, want 2 (the _partial.md must be skipped): %+v", len(got), got)
	}
	// Sorted by ID.
	if got[0].ID != "auth-reset-0001" || got[1].ID != "checkout-0002" {
		t.Fatalf("stories not sorted by ID: %q, %q", got[0].ID, got[1].ID)
	}
}

func TestStoryFrontmatterParsed(t *testing.T) {
	got := NewFS("testdata").Stories()
	auth := got[0]
	if auth.Slug != "password-reset" {
		t.Errorf("slug = %q, want password-reset", auth.Slug)
	}
	if auth.Priority != "high" {
		t.Errorf("priority = %q, want high", auth.Priority)
	}
	if auth.Phase != "implement" {
		t.Errorf("phase = %q, want implement (from labels)", auth.Phase)
	}
	if !auth.UI {
		t.Errorf("ui = false, want true")
	}
	if auth.Issue != 42 {
		t.Errorf("issue = %d, want 42", auth.Issue)
	}
}

func TestStoryPhaseDefaultsAndBracketLabels(t *testing.T) {
	got := NewFS("testdata").Stories()
	checkout := got[1]
	if checkout.Phase != "architect" {
		t.Errorf("phase = %q, want architect (from bracketed labels)", checkout.Phase)
	}
	if checkout.UI {
		t.Errorf("checkout ui = true, want false")
	}
	if checkout.Issue != 0 {
		t.Errorf("checkout issue = %d, want 0 (absent)", checkout.Issue)
	}
}

func TestStoriesEmptyWhenNoDir(t *testing.T) {
	if got := NewFS("testdata/nonexistent").Stories(); len(got) != 0 {
		t.Errorf("missing stories dir should yield no stories, got %d", len(got))
	}
}

func TestPhaseLabel(t *testing.T) {
	cases := []struct{ labels, prefix, want string }{
		{"phase:implement", "phase", "implement"},
		{"[phase:architect]", "phase", "architect"},
		{"type:bug,phase:validate", "phase", "validate"},
		{"type:feature", "phase", ""}, // no phase label
		{"type:bug,priority:high", "priority", "high"},
		{"", "phase", ""},
	}
	for _, c := range cases {
		if got := phaseLabel(c.labels, c.prefix); got != c.want {
			t.Errorf("phaseLabel(%q,%q) = %q, want %q", c.labels, c.prefix, got, c.want)
		}
	}
}

func TestParsesMultilineLabelsAndTitle(t *testing.T) {
	dir := t.TempDir()
	story := filepath.Join(dir, "docs", "stories", "reset-1", "Story.md")
	os.MkdirAll(filepath.Dir(story), 0o755)
	// The real saveStory format: title + a multi-line YAML labels list.
	os.WriteFile(story, []byte(`---
id: "reset-0001"
title: "Password reset flow"
priority: "medium"
ui: true
labels:
  - "type:feature"
  - "priority:high"
  - "phase:implement"
issue_number: 42
---

# Password reset flow
`), 0o644)

	stories := NewFS(dir).Stories()
	if len(stories) != 1 {
		t.Fatalf("want 1 story, got %d", len(stories))
	}
	s := stories[0]
	if s.Title != "Password reset flow" {
		t.Errorf("title = %q", s.Title)
	}
	if s.Phase != "implement" {
		t.Errorf("phase from multi-line labels = %q, want implement (not the 'discover' default)", s.Phase)
	}
	if !s.UI || s.Issue != 42 {
		t.Errorf("ui/issue mis-parsed: ui=%v issue=%d", s.UI, s.Issue)
	}
}
