package spawn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Command returns the argv that opens args in a new tab/window of the detected
// terminal. Direct terminals (multiplexers, wezterm, kitty) receive args verbatim;
// script terminals receive the pre-written launch script path. It is a pure function
// of getenv/args/script so it can be unit-tested. Warp has no external-launch CLI, so
// it returns ErrNoTerminal.
func Command(getenv func(string) string, args []string, script string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("spawn: empty command")
	}
	t, ok := Detect(getenv)
	if !ok {
		return nil, ErrNoTerminal
	}
	switch t.Kind {
	case "zellij":
		return append([]string{"zellij", "action", "new-tab", "--"}, args...), nil
	case "tmux":
		return append([]string{"tmux", "new-window", "--"}, args...), nil
	case "screen":
		return append([]string{"screen", "-S", getenv("STY"), "-X", "screen", "-t", args[0]}, args...), nil
	case "wezterm":
		return append([]string{"wezterm", "cli", "spawn", "--"}, args...), nil
	case "kitty":
		return append([]string{"kitty", "@", "launch", "--type=tab", "--"}, args...), nil
	case "ghostty":
		return []string{"open", "-na", "Ghostty", "--args", "-e", script}, nil
	case "iterm2":
		return []string{"open", "-na", "iTerm", script}, nil
	case "apple-terminal":
		return []string{"open", "-a", "Terminal", script}, nil
	case "alacritty":
		return []string{"alacritty", "-e", script}, nil
	case "gnome-terminal":
		return []string{"gnome-terminal", "--tab", "--", script}, nil
	case "konsole":
		return []string{"konsole", "--new-tab", "-e", script}, nil
	case "tilix":
		return []string{"tilix", "--action=app-new-session", "-e", script}, nil
	case "terminator":
		return []string{"terminator", "-e", script}, nil
	case "windows-terminal":
		return []string{"wt", "-w", "0", "nt", script}, nil
	default: // warp and anything without an external-launch mechanism
		return nil, ErrNoTerminal
	}
}

// Spawn opens args in a new tab/window of the current terminal, writing a launch
// script first for script-based terminals. Returns ErrNoTerminal when the terminal
// can't be driven from outside — callers should then show a manual-launch hint.
func Spawn(args []string) error {
	t, ok := Detected()
	if !ok {
		return ErrNoTerminal
	}
	script := ""
	if !t.Direct {
		s, err := writeLaunchScript(args)
		if err != nil {
			return ErrNoTerminal
		}
		script = s
	}
	argv, err := Command(os.Getenv, args, script)
	if err != nil {
		return err
	}
	return exec.Command(argv[0], argv[1:]...).Start()
}

// writeLaunchScript writes a temp script that runs args, resolving the binary to a
// full path so it works in minimal-PATH shells. Returns the script path.
func writeLaunchScript(args []string) (string, error) {
	if runtime.GOOS == "windows" {
		f, err := os.CreateTemp("", "maple-launch-*.bat")
		if err != nil {
			return "", err
		}
		defer f.Close()
		var parts []string
		for _, a := range args {
			if strings.ContainsAny(a, ` "`) {
				parts = append(parts, `"`+strings.ReplaceAll(a, `"`, `""`)+`"`)
			} else {
				parts = append(parts, a)
			}
		}
		fmt.Fprintf(f, "@echo off\r\n%s\r\npause\r\n", strings.Join(parts, " "))
		return f.Name(), nil
	}

	bin := args[0]
	if full, err := exec.LookPath(bin); err == nil {
		bin = full
	}
	f, err := os.CreateTemp("", "maple-launch-*.sh")
	if err != nil {
		return "", err
	}
	defer f.Close()
	rest := ""
	if len(args) > 1 {
		rest = " " + shellQuote(args[1:])
	}
	fmt.Fprintf(f, "#!/bin/sh\nexec %s%s\n", shellQuote([]string{bin}), rest)
	if err := os.Chmod(f.Name(), 0o755); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// shellQuote single-quotes each arg for /bin/sh.
func shellQuote(args []string) string {
	q := make([]string, len(args))
	for i, a := range args {
		q[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(q, " ")
}
