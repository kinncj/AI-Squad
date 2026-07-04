// Package spawn opens a command in a new tab/window of the terminal the user is
// already running in. The terminal is identified from environment variables using
// the same precedence as the OG tui/terminal_spawn.go: multiplexers first, then
// IPC terminals, then TERM_PROGRAM, then secondary per-session env vars. Detection
// is a pure function of the environment so it is unit-testable; the actual process
// launch is a thin exec wrapper.
package spawn

import (
	"errors"
	"os"
)

// ErrNoTerminal is returned when no supported new-terminal mechanism is found, or
// when the identified terminal has no way to open a new window from outside.
var ErrNoTerminal = errors.New("no supported new-terminal mechanism found")

// Terminal identifies the running terminal and how to drive it.
type Terminal struct {
	Kind   string // stable identifier, e.g. "tmux", "ghostty", "kitty"
	Direct bool   // true = spawn the command with args directly; false = via a launch script
}

// Detect identifies the running terminal from getenv, or ok=false when none match.
// Order matters: multiplexers win over the host terminal so a new tab lands in the
// multiplexer the user is actually looking at.
func Detect(getenv func(string) string) (Terminal, bool) {
	e := getenv
	// 1. multiplexers — tab support, no display needed.
	switch {
	case e("ZELLIJ") != "":
		return Terminal{"zellij", true}, true
	case e("TMUX") != "":
		return Terminal{"tmux", true}, true
	case e("STY") != "":
		return Terminal{"screen", true}, true
	}
	// 2. GPU terminals with native IPC.
	switch {
	case e("WEZTERM_PANE") != "":
		return Terminal{"wezterm", true}, true
	case e("KITTY_PID") != "" || e("KITTY_WINDOW_ID") != "":
		return Terminal{"kitty", true}, true
	}
	// 3. TERM_PROGRAM — canonical per-terminal identity.
	switch e("TERM_PROGRAM") {
	case "ghostty":
		return Terminal{"ghostty", false}, true
	case "iTerm.app":
		return Terminal{"iterm2", false}, true
	case "Apple_Terminal":
		return Terminal{"apple-terminal", false}, true
	case "WarpTerminal":
		return Terminal{"warp", false}, true
	case "Hyper":
		return Terminal{"hyper", false}, true
	}
	// 4. Secondary per-session env vars.
	switch {
	case e("ITERM_SESSION_ID") != "":
		return Terminal{"iterm2", false}, true
	case e("TERM") == "alacritty" || e("ALACRITTY_SOCKET") != "":
		return Terminal{"alacritty", false}, true
	case e("GNOME_TERMINAL_SCREEN") != "" || e("GNOME_TERMINAL_SERVICE") != "":
		return Terminal{"gnome-terminal", false}, true
	case e("KONSOLE_VERSION") != "" || e("KONSOLE_DBUS_SESSION") != "" || e("KONSOLE_DBUS_WINDOW") != "":
		return Terminal{"konsole", false}, true
	case e("TILIX_ID") != "":
		return Terminal{"tilix", false}, true
	case e("TERMINATOR_UUID") != "":
		return Terminal{"terminator", false}, true
	case e("WT_SESSION") != "":
		return Terminal{"windows-terminal", false}, true
	}
	return Terminal{}, false
}

// Detected reports the current terminal from the real environment.
func Detected() (Terminal, bool) { return Detect(os.Getenv) }
