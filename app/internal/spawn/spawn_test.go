package spawn

import "testing"

// envFrom returns a getenv func backed by a map.
func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDetectByCategory(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		kind string
	}{
		{"zellij", map[string]string{"ZELLIJ": "0"}, "zellij"},
		{"tmux", map[string]string{"TMUX": "/tmp/tmux-501/default,1,0"}, "tmux"},
		{"screen", map[string]string{"STY": "1234.pts-0"}, "screen"},
		{"wezterm", map[string]string{"WEZTERM_PANE": "0"}, "wezterm"},
		{"kitty pid", map[string]string{"KITTY_PID": "999"}, "kitty"},
		{"kitty window", map[string]string{"KITTY_WINDOW_ID": "1"}, "kitty"},
		{"ghostty", map[string]string{"TERM_PROGRAM": "ghostty"}, "ghostty"},
		{"iterm term_program", map[string]string{"TERM_PROGRAM": "iTerm.app"}, "iterm2"},
		{"apple terminal", map[string]string{"TERM_PROGRAM": "Apple_Terminal"}, "apple-terminal"},
		{"warp", map[string]string{"TERM_PROGRAM": "WarpTerminal"}, "warp"},
		{"hyper", map[string]string{"TERM_PROGRAM": "Hyper"}, "hyper"},
		{"iterm session", map[string]string{"ITERM_SESSION_ID": "w0t0p0"}, "iterm2"},
		{"alacritty term", map[string]string{"TERM": "alacritty"}, "alacritty"},
		{"alacritty socket", map[string]string{"ALACRITTY_SOCKET": "/tmp/a.sock"}, "alacritty"},
		{"gnome", map[string]string{"GNOME_TERMINAL_SCREEN": "/org/gnome/x"}, "gnome-terminal"},
		{"konsole", map[string]string{"KONSOLE_VERSION": "220400"}, "konsole"},
		{"tilix", map[string]string{"TILIX_ID": "abc"}, "tilix"},
		{"terminator", map[string]string{"TERMINATOR_UUID": "urn:uuid:x"}, "terminator"},
		{"windows terminal", map[string]string{"WT_SESSION": "guid"}, "windows-terminal"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := Detect(envFrom(c.env))
			if !ok {
				t.Fatalf("Detect(%v) not ok", c.env)
			}
			if got.Kind != c.kind {
				t.Errorf("Detect kind = %q, want %q", got.Kind, c.kind)
			}
		})
	}
}

func TestDetectNoneWhenEmpty(t *testing.T) {
	if _, ok := Detect(envFrom(map[string]string{"TERM": "xterm-256color"})); ok {
		t.Error("a bare TERM should not identify a spawnable terminal")
	}
}

func TestDetectMultiplexerBeatsHostTerminal(t *testing.T) {
	// Inside tmux running under iTerm, a new tab should go to tmux, not iTerm.
	env := envFrom(map[string]string{"TMUX": "/tmp/x,1,0", "TERM_PROGRAM": "iTerm.app"})
	got, ok := Detect(env)
	if !ok || got.Kind != "tmux" {
		t.Errorf("multiplexer should win, got %q ok=%v", got.Kind, ok)
	}
}

func TestDetectDirectFlag(t *testing.T) {
	// Multiplexers and IPC terminals spawn directly; script terminals do not.
	direct, _ := Detect(envFrom(map[string]string{"KITTY_PID": "1"}))
	if !direct.Direct {
		t.Error("kitty should be a direct-spawn terminal")
	}
	script, _ := Detect(envFrom(map[string]string{"TERM_PROGRAM": "ghostty"}))
	if script.Direct {
		t.Error("ghostty should spawn via a launch script")
	}
}
