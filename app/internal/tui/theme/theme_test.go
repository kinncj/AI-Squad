package theme

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedThemeIsWellFormed(t *testing.T) {
	th, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if th.Name != "maple" {
		t.Errorf("Name = %q, want maple", th.Name)
	}
	if th.DefaultMode != "dark" {
		t.Errorf("DefaultMode = %q, want dark", th.DefaultMode)
	}
	for _, mode := range []string{"dark", "light"} {
		if _, ok := th.Modes[mode]; !ok {
			t.Errorf("missing mode %q", mode)
		}
	}
}

func TestActiveModeHasRequiredRolesAndStates(t *testing.T) {
	th, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for name, m := range th.Modes {
		for _, role := range []string{"base", "border", "border_focus", "title", "faint", "accent", "selected"} {
			if _, ok := m.Structure[role]; !ok {
				t.Errorf("mode %q missing structural role %q", name, role)
			}
		}
		// Every MAPLE pipeline state must be themeable.
		for _, st := range []string{"running", "paused", "rate_limited", "done", "blocked", "pending"} {
			if _, ok := m.States[st]; !ok {
				t.Errorf("mode %q missing state %q", name, st)
			}
		}
	}
}

func TestRoleLookupFallsBackToZero(t *testing.T) {
	th, _ := Load()
	m := th.ActiveMode()
	if got := m.Role("does-not-exist"); got.FG != "" || got.BG != "" {
		t.Errorf("unknown role should be zero value, got %+v", got)
	}
	if got := m.State("does-not-exist"); got.FG != "" || got.Label != "" {
		t.Errorf("unknown state should be zero value, got %+v", got)
	}
}

func TestRoleStyleAppliesForegroundAndAttrs(t *testing.T) {
	th, _ := Load()
	title := th.ActiveMode().Role("title")
	st := title.Style()
	if !st.GetBold() {
		t.Errorf("title role should be bold")
	}
	if st.GetForeground() == nil {
		t.Errorf("title role should set a foreground")
	}
}

func TestStateRenderPrependsGlyph(t *testing.T) {
	th, _ := Load()
	running := th.ActiveMode().State("running")
	out := running.Render("running")
	if !strings.Contains(out, running.Glyph) {
		t.Errorf("Render(%q) = %q, want glyph %q present", "running", out, running.Glyph)
	}
	if !strings.Contains(out, "running") {
		t.Errorf("Render output %q missing label text", out)
	}
}

func TestLoadValidatesDefaultModePresent(t *testing.T) {
	// Sanity: the embedded default_mode must resolve. Load already enforces this;
	// this guards against a future JSON edit that points default_mode at a missing key.
	th, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, ok := th.Modes[th.DefaultMode]; !ok {
		t.Fatalf("default_mode %q not in modes", th.DefaultMode)
	}
}
