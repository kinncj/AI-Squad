package theme

import (
	"strings"
	"testing"
)

func TestLoadReturnsDefault(t *testing.T) {
	th, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if th.Name != DefaultName {
		t.Errorf("default theme = %q, want %q", th.Name, DefaultName)
	}
	if _, ok := th.Modes[th.DefaultMode]; !ok {
		t.Errorf("default_mode %q not present", th.DefaultMode)
	}
}

func TestNamesIncludesAllFive(t *testing.T) {
	names := Names()
	if len(names) < 6 {
		t.Fatalf("expected at least 6 themes, got %v", names)
	}
	if names[0] != DefaultName {
		t.Errorf("default should be first, got %v", names)
	}
	for _, want := range []string{"tokyo-night", "catppuccin-mocha", "gruvbox", "nord", "everforest", "maple"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing theme %q in %v", want, names)
		}
	}
}

func TestSwitchKnownAndUnknown(t *testing.T) {
	th, err := Switch("gruvbox")
	if err != nil || th.Name != "gruvbox" {
		t.Errorf("Switch(gruvbox) = %v, %v", th, err)
	}
	if _, err := Switch("does-not-exist"); err == nil {
		t.Error("Switch of unknown theme should error")
	}
}

func TestEveryThemeHasRequiredRolesAndStates(t *testing.T) {
	for _, name := range Names() {
		th, err := Switch(name)
		if err != nil {
			t.Fatalf("Switch(%q): %v", name, err)
		}
		m := th.ActiveMode()
		for _, role := range []string{"base", "border", "border_focus", "title", "faint", "accent", "leaf", "selected"} {
			if _, ok := m.Structure[role]; !ok {
				t.Errorf("theme %q missing role %q", name, role)
			}
		}
		for _, st := range []string{"running", "paused", "rate_limited", "done", "blocked", "pending"} {
			if _, ok := m.States[st]; !ok {
				t.Errorf("theme %q missing state %q", name, st)
			}
		}
	}
}

func TestRoleLookupFallsBackToZero(t *testing.T) {
	m := mustLoad(t).ActiveMode()
	if got := m.Role("nope"); got.FG != "" || got.BG != "" {
		t.Errorf("unknown role should be zero, got %+v", got)
	}
	if got := m.State("nope"); got.FG != "" || got.Label != "" {
		t.Errorf("unknown state should be zero, got %+v", got)
	}
}

func TestRoleStyleAppliesForegroundAndAttrs(t *testing.T) {
	title := mustLoad(t).ActiveMode().Role("title")
	st := title.Style()
	if !st.GetBold() {
		t.Error("title role should be bold")
	}
	if st.GetForeground() == nil {
		t.Error("title role should set a foreground")
	}
}

func TestStateRenderPrependsGlyph(t *testing.T) {
	running := mustLoad(t).ActiveMode().State("running")
	out := running.Render("running")
	if !strings.Contains(out, running.Glyph) || !strings.Contains(out, "running") {
		t.Errorf("Render output %q missing glyph or label", out)
	}
}

func mustLoad(t *testing.T) *Theme {
	t.Helper()
	th, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return th
}
