package state

import "testing"

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

func TestPhaseFromLabels(t *testing.T) {
	cases := map[string]string{
		"phase:implement":          "implement",
		"[phase:architect]":        "architect",
		"type:bug, phase:validate": "validate",
		"type:feature":             "discover", // no phase label -> default
		"":                         "discover",
	}
	for in, want := range cases {
		if got := phaseFromLabels(in); got != want {
			t.Errorf("phaseFromLabels(%q) = %q, want %q", in, got, want)
		}
	}
}
