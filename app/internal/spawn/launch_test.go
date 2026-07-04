package spawn

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestCommandDirectTerminals(t *testing.T) {
	cases := []struct {
		env    map[string]string
		prefix []string
	}{
		{map[string]string{"ZELLIJ": "0"}, []string{"zellij", "action", "new-tab", "--"}},
		{map[string]string{"TMUX": "x"}, []string{"tmux", "new-window", "--"}},
		{map[string]string{"WEZTERM_PANE": "0"}, []string{"wezterm", "cli", "spawn", "--"}},
		{map[string]string{"KITTY_PID": "1"}, []string{"kitty", "@", "launch", "--type=tab", "--"}},
	}
	for _, c := range cases {
		argv, err := Command(envFrom(c.env), []string{"claude", "--resume"}, "")
		if err != nil {
			t.Fatalf("Command(%v): %v", c.env, err)
		}
		for i, p := range c.prefix {
			if argv[i] != p {
				t.Errorf("argv[%d] = %q, want %q (env %v)", i, argv[i], p, c.env)
			}
		}
		// the real command trails the prefix
		if argv[len(argv)-2] != "claude" || argv[len(argv)-1] != "--resume" {
			t.Errorf("direct argv should end with the command, got %v", argv)
		}
	}
}

func TestCommandScreenUsesSTY(t *testing.T) {
	argv, err := Command(envFrom(map[string]string{"STY": "1234.pts-0"}), []string{"claude"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if argv[0] != "screen" || argv[2] != "1234.pts-0" {
		t.Errorf("screen argv should carry the STY session, got %v", argv)
	}
}

func TestCommandScriptTerminals(t *testing.T) {
	script := "/tmp/maple-launch-x.sh"
	cases := []struct {
		env      map[string]string
		launcher string
	}{
		{map[string]string{"TERM_PROGRAM": "ghostty"}, "open"},
		{map[string]string{"TERM_PROGRAM": "Apple_Terminal"}, "open"},
		{map[string]string{"ALACRITTY_SOCKET": "/tmp/a"}, "alacritty"},
		{map[string]string{"KONSOLE_VERSION": "1"}, "konsole"},
	}
	for _, c := range cases {
		argv, err := Command(envFrom(c.env), []string{"claude"}, script)
		if err != nil {
			t.Fatalf("Command(%v): %v", c.env, err)
		}
		if argv[0] != c.launcher {
			t.Errorf("launcher = %q, want %q (env %v)", argv[0], c.launcher, c.env)
		}
		if !containsStr(argv, script) {
			t.Errorf("script terminal argv should include the script path, got %v", argv)
		}
	}
}

func TestCommandWarpUnsupported(t *testing.T) {
	if _, err := Command(envFrom(map[string]string{"TERM_PROGRAM": "WarpTerminal"}), []string{"claude"}, "s"); !errors.Is(err, ErrNoTerminal) {
		t.Errorf("warp should return ErrNoTerminal, got %v", err)
	}
}

func TestCommandNoTerminal(t *testing.T) {
	if _, err := Command(envFrom(map[string]string{}), []string{"claude"}, ""); !errors.Is(err, ErrNoTerminal) {
		t.Errorf("no terminal should return ErrNoTerminal, got %v", err)
	}
}

func TestCommandEmptyArgs(t *testing.T) {
	if _, err := Command(envFrom(map[string]string{"TMUX": "x"}), nil, ""); err == nil {
		t.Error("empty args should error")
	}
}

func TestWriteLaunchScriptRunnable(t *testing.T) {
	path, err := writeLaunchScript([]string{"echo", "hello world"})
	if err != nil {
		t.Fatalf("writeLaunchScript: %v", err)
	}
	data := mustRead(t, path)
	if !strings.Contains(data, "hello world") {
		t.Errorf("script should embed the args, got:\n%s", data)
	}
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
