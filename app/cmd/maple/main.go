// Command maple is the rebuilt MAPLE TUI + CLI.
//
// With no arguments it runs the dashboard TUI. Subcommands:
//
//	maple init [--force]   scaffold the MAPLE template into the current project
//	maple version          print the version
//
// Developed on feature/better-ui-ux; see docs/adrs/ADR-002 and the rework spec.
package main

import (
	"bufio"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/kinncj/maple/app/internal/gh"
	"github.com/kinncj/maple/app/internal/harness"
	"github.com/kinncj/maple/app/internal/portal"
	"github.com/kinncj/maple/app/internal/portalsock"
	"github.com/kinncj/maple/app/internal/resume"
	"github.com/kinncj/maple/app/internal/scaffold"
	"github.com/kinncj/maple/app/internal/selfupdate"
	"github.com/kinncj/maple/app/internal/state"
	"github.com/kinncj/maple/app/internal/tui/brand"
	"github.com/kinncj/maple/app/internal/tui/dashboard"
	"github.com/kinncj/maple/app/internal/tui/menu"
	reqtui "github.com/kinncj/maple/app/internal/tui/req"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI()
		return
	}
	switch args[0] {
	case "init":
		runInit(hasFlag(args[1:], "--force"))
	case "req":
		if err := reqtui.Run(); err != nil {
			die(err)
		}
	case "update":
		runUpdate(hasFlag(args[1:], "--yes") || hasFlag(args[1:], "-y"), hasFlag(args[1:], "--diff"))
	case "labels":
		if err := gh.RunLabels(); err != nil {
			die(err)
		}
	case "project":
		if err := gh.RunProject(); err != nil {
			die(err)
		}
	case "self-update", "upgrade":
		if err := selfupdate.Run(version); err != nil {
			die(err)
		}
	case "resume-session", "resume":
		harness := ""
		if len(args) > 1 {
			harness = args[1]
		}
		if err := resume.Session(harness); err != nil {
			die(err)
		}
	case "rtk-audit":
		runRTKAudit()
	case "emit":
		runEmit(args[1:])
	case "portal":
		runPortalServe(args[1:])
	case "version", "--version", "-v":
		fmt.Println("maple", version)
	case "help", "--help", "-h":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "maple: unknown command %q\n\n", args[0])
		usage(os.Stderr)
		os.Exit(2)
	}
}

// runTUI is the no-argument entry point. Without a TTY it prints usage. When the
// project is already set up it runs the dashboard loop; otherwise it shows the
// interactive setup menu (Init / Requirements / Labels / Project / Help) — the old
// binary's behaviour — and enters the dashboard once initialised.
func runTUI() {
	if !isTTY() {
		usage(os.Stdout)
		return
	}
	// Make maple a mini agentic IDE: if we're not already in a multiplexer, relaunch
	// inside one so harnesses open as side panes. Prefer herdr (agent-native) when it's
	// installed; otherwise fall back to a styled tmux session. Opt out per backend with
	// MAPLE_NO_HERDR=1 / MAPLE_NO_TMUX=1 (with both off, harnesses run in-terminal).
	if !inMultiplexer() {
		if os.Getenv("MAPLE_NO_HERDR") == "" && wrapInHerdr() {
			return
		}
		if os.Getenv("MAPLE_NO_TMUX") == "" && wrapInTmux() {
			return
		}
	}
	if initialized() {
		runDashboardLoop()
		return
	}
	runSetupMenu()
}

// inMultiplexer reports whether maple is already running inside a terminal multiplexer
// or a splittable terminal it can drive.
func inMultiplexer() bool {
	return harness.InMultiplexer(os.Getenv)
}

// wrapInHerdr relaunches maple inside an isolated, persistent herdr session named "maple"
// and attaches to it, so harness launches open as herdr side panes. It mirrors wrapInTmux
// but over herdr's socket-API CLI: a fresh headless server has no panes, so it creates a
// workspace, execs maple into that pane, then attaches. Returns false when herdr is absent
// or any bootstrap step fails — the caller then tries tmux. Opt out with MAPLE_NO_HERDR=1.
func wrapInHerdr() bool {
	bin, err := exec.LookPath("herdr")
	if err != nil {
		return false
	}
	self, err := os.Executable()
	if err != nil {
		return false
	}
	const sentinel = "__maple_herdr_ready__"
	const label = "maple"
	cwd, _ := os.Getwd()
	session := mapleSessionName(cwd)

	// herdr defaults kitty_graphics off, so maple's PNG splash (and any image a harness
	// prints) would vanish. Enable it, then reload a running server so it applies live.
	if ensureHerdrKittyGraphics() && herdrRunning(bin, session) {
		_ = exec.Command(bin, "--session", session, "server", "reload-config").Run()
	}

	// herdr sessions are persistent (workspaces survive stop/delete on disk), so the old
	// "stop then create" piled up a new workspace every launch. Instead: ensure the server,
	// then reuse maple's own labelled workspace if it exists (recover — preserving any
	// harness split panes) and close stray workspaces, else create exactly one.
	if !herdrRunning(bin, session) {
		if err := exec.Command(bin, "--session", session, "server").Start(); err != nil {
			return false
		}
		if !waitHerdrReady(bin, session) {
			return false
		}
	}
	pane := reuseOrCreateHerdrWorkspace(bin, session, cwd, label)
	if pane == "" {
		return false
	}

	// A freshly spawned pane's shell may not read input for a beat; echo a sentinel and
	// block until herdr sees it, so the exec that follows lands in a live shell.
	_ = exec.Command(bin, "--session", session, "pane", "run", pane, "echo "+sentinel).Run()
	_ = exec.Command(bin, "--session", session, "wait", "output", pane, "--match", sentinel, "--timeout", "8000").Run()
	if err := exec.Command(bin, "--session", session, "pane", "run", pane, "exec "+shellSingleQuote(self)).Run(); err != nil {
		return false
	}

	fmt.Println(brand.Leaf + " starting maple in herdr — harnesses open in a side pane")
	attach := exec.Command(bin, "--session", session)
	attach.Stdin, attach.Stdout, attach.Stderr = os.Stdin, os.Stdout, os.Stderr
	_ = attach.Run()
	return true
}

// ensureHerdrKittyGraphics turns on inline images in herdr (its kitty_graphics defaults
// off), so the splash and any harness-printed image render instead of vanishing. It creates
// the config when absent and adds the key under [experimental] when missing, but never
// overrides an explicit user setting. Returns true when it changed the config.
func ensureHerdrKittyGraphics() bool {
	path := os.Getenv("HERDR_CONFIG_PATH")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		path = filepath.Join(home, ".config", "herdr", "config.toml")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		return os.WriteFile(path, []byte("[experimental]\nkitty_graphics = true\n"), 0o644) == nil
	}
	body := string(data)
	if strings.Contains(body, "kitty_graphics") {
		return false // respect an explicit user setting
	}
	if strings.Contains(body, "[experimental]") {
		body = strings.Replace(body, "[experimental]", "[experimental]\nkitty_graphics = true", 1)
	} else {
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		body += "\n[experimental]\nkitty_graphics = true\n"
	}
	return os.WriteFile(path, []byte(body), 0o644) == nil
}

// herdrRunning reports whether the named session's server is already up.
func herdrRunning(bin, session string) bool {
	out, _ := exec.Command(bin, "--session", session, "status").Output()
	return strings.Contains(string(out), "status: running")
}

// reuseOrCreateHerdrWorkspace returns the pane maple should run in. It reuses the workspace
// tagged with label (so a relaunch recovers maple's workspace, keeping any harness split
// panes) and closes every other workspace so relaunches don't accumulate. The session is a
// dedicated per-project one, so closing strays never touches the user's own herdr work.
func reuseOrCreateHerdrWorkspace(bin, session, cwd, label string) string {
	mineWS, strays := listHerdrWorkspaces(bin, session, label)
	for _, id := range strays {
		_ = exec.Command(bin, "--session", session, "workspace", "close", id).Run()
	}
	if mineWS != "" {
		if pane := firstHerdrPane(bin, session, mineWS); pane != "" {
			return pane
		}
	}
	args := []string{"--session", session, "workspace", "create", "--focus", "--label", label}
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return ""
	}
	return parseHerdrRootPaneID(out)
}

// listHerdrWorkspaces returns the id of the workspace tagged with label and the ids of all
// other (stray) workspaces in the session.
func listHerdrWorkspaces(bin, session, label string) (mine string, strays []string) {
	out, err := exec.Command(bin, "--session", session, "workspace", "list").Output()
	if err != nil {
		return "", nil
	}
	var resp struct {
		Result struct {
			Workspaces []struct {
				WorkspaceID string `json:"workspace_id"`
				Label       string `json:"label"`
			} `json:"workspaces"`
		} `json:"result"`
	}
	if json.Unmarshal(out, &resp) != nil {
		return "", nil
	}
	for _, w := range resp.Result.Workspaces {
		if mine == "" && w.Label == label {
			mine = w.WorkspaceID
			continue
		}
		strays = append(strays, w.WorkspaceID)
	}
	return mine, strays
}

// firstHerdrPane returns the lowest-numbered pane id in a workspace (maple's root pane —
// harness splits come later as higher pN, so exec-ing maple here never clobbers a harness).
func firstHerdrPane(bin, session, workspaceID string) string {
	out, err := exec.Command(bin, "--session", session, "pane", "list", "--workspace", workspaceID).Output()
	if err != nil {
		return ""
	}
	var resp struct {
		Result struct {
			Panes []struct {
				PaneID string `json:"pane_id"`
			} `json:"panes"`
		} `json:"result"`
	}
	if json.Unmarshal(out, &resp) != nil {
		return ""
	}
	best := ""
	for _, p := range resp.Result.Panes {
		if best == "" || paneLess(p.PaneID, best) {
			best = p.PaneID
		}
	}
	return best
}

// paneLess orders herdr pane ids ("w5:p2") by their numeric pane suffix.
func paneLess(a, b string) bool { return paneNum(a) < paneNum(b) }

func paneNum(id string) int {
	if i := strings.LastIndex(id, ":p"); i >= 0 {
		n, _ := strconv.Atoi(id[i+2:])
		return n
	}
	return 1 << 30
}

// mapleSessionName is the dedicated per-project multiplexer session name: an explicit
// `session:` in project.config.yaml if set, else "maple-<project-dir>" — persisted to the
// config (when one exists) so it is stable and user-editable, and so two projects never
// collide on one shared session.
func mapleSessionName(cwd string) string {
	if s := configSession(); s != "" {
		return s
	}
	base := "project"
	if cwd != "" {
		base = sanitizeSession(filepath.Base(cwd))
	}
	name := "maple-" + base
	persistConfigSession(name)
	return name
}

func sanitizeSession(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == ' ':
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "project"
	}
	return out
}

// configSession reads a top-level `session:` value from project.config.yaml, or "".
func configSession() string {
	data, err := os.ReadFile("project.config.yaml")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "session:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(t, "session:")), `"'`)
		}
	}
	return ""
}

// persistConfigSession appends `session: <name>` to project.config.yaml when the file
// exists and does not already carry the key. No-op in an uninitialised directory.
func persistConfigSession(name string) {
	data, err := os.ReadFile("project.config.yaml")
	if err != nil {
		return
	}
	if configSession() != "" {
		return
	}
	body := string(data)
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	_ = os.WriteFile("project.config.yaml", []byte(body+"session: "+name+"\n"), 0o644)
}

// waitHerdrReady polls until the named session's headless server reports running (~4s cap).
func waitHerdrReady(bin, session string) bool {
	for i := 0; i < 40; i++ {
		out, _ := exec.Command(bin, "--session", session, "status").Output()
		if strings.Contains(string(out), "status: running") {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// parseHerdrRootPaneID pulls the root pane id from `herdr workspace create` JSON.
func parseHerdrRootPaneID(out []byte) string {
	var resp struct {
		Result struct {
			RootPane struct {
				PaneID string `json:"pane_id"`
			} `json:"root_pane"`
		} `json:"result"`
	}
	if json.Unmarshal(out, &resp) == nil {
		return resp.Result.RootPane.PaneID
	}
	return ""
}

// shellSingleQuote wraps a path for `herdr pane run`, which types one line into the pane's
// shell, so a path with spaces survives.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// wrapInTmux relaunches maple inside a fresh, maple-styled tmux session (side-pane
// borders, a status bar with pane-jump hints) and attaches to it. Returns false (so the
// caller proceeds normally) when tmux isn't available or the session can't be created.
func wrapInTmux() bool {
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		return false
	}
	self, err := os.Executable()
	if err != nil {
		return false
	}
	cwd, _ := os.Getwd()
	session := mapleSessionName(cwd) // per-project, so two projects don't share one session
	_ = exec.Command(tmux, "kill-session", "-t", session).Run() // clear a stale one
	// -c pins the session (and maple inside it) to the current project dir, so all its
	// file-based state reads (stories, sessions, gates) resolve correctly.
	args := []string{"new-session", "-d", "-s", session}
	if cwd != "" {
		args = append(args, "-c", cwd)
	}
	if err := exec.Command(tmux, append(args, self)...).Run(); err != nil {
		return false
	}
	configureMapleTmux(tmux, session)
	fmt.Println(brand.Leaf + " starting maple in tmux — harnesses open in a side pane · C-b ←/→ to switch")
	attach := exec.Command(tmux, "attach", "-t", session)
	attach.Stdin, attach.Stdout, attach.Stderr = os.Stdin, os.Stdout, os.Stderr
	_ = attach.Run()
	return true
}

// configureMapleTmux styles the maple session: pane-border titles (so you can see
// which pane is maple vs a harness), mouse on, and a status bar showing how to switch
// panes. Scoped to the maple session — never touches the user's global tmux config.
func configureMapleTmux(tmux, s string) {
	opts := [][]string{
		{"set-option", "-t", s, "mouse", "on"},
		// Close a pane when its process exits instead of leaving a dead-pane corpse —
		// quitting maple or a harness should tear the pane down cleanly. If maple exits
		// while a harness is still running, the harness pane just takes over.
		{"set-option", "-t", s, "-w", "remain-on-exit", "off"},
		{"set-option", "-t", s, "-w", "pane-border-status", "top"},
		{"set-option", "-t", s, "-w", "pane-border-format", " #{?pane_active,#[bold]▸ ,}#{pane_title} "},
		{"set-option", "-t", s, "status-style", "bg=default"},
		{"set-option", "-t", s, "status-left", "#[bold] 🍁 maple #[default]"},
		{"set-option", "-t", s, "status-left-length", "24"},
		{"set-option", "-t", s, "status-right", "#[dim]C-b ←/→ switch · C-b z zoom · C-b d detach "},
		{"set-option", "-t", s, "status-right-length", "60"},
		{"select-pane", "-t", s, "-T", "maple"},
	}
	for _, o := range opts {
		_ = exec.Command(tmux, o...).Run()
	}
}

// runSetupMenu loops the setup menu until the project is initialised (then hands off
// to the dashboard) or the user quits.
func runSetupMenu() {
	for {
		action, err := menu.Run(version)
		if err != nil {
			die(err)
		}
		switch action {
		case menu.ActionQuit:
			return
		case menu.ActionInit:
			runInit(false)
			if initialized() {
				runDashboardLoop()
				return
			}
		case menu.ActionUpdate:
			runUpdate(false, false)
		case menu.ActionReq:
			if err := reqtui.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "req:", err)
			}
		case menu.ActionLabels:
			if err := gh.RunLabels(); err != nil {
				fmt.Fprintln(os.Stderr, "labels:", err)
			}
		case menu.ActionProject:
			if err := gh.RunProject(); err != nil {
				fmt.Fprintln(os.Stderr, "project:", err)
			}
		default:
			return
		}
	}
}

// initialized reports whether the current directory has a MAPLE project config.
func initialized() bool {
	_, err := os.Stat("project.config.yaml")
	return err == nil
}

// startDesignPortal picks a free port and starts the design-review portal for the
// lifetime of the session. It returns the authoritative URL (the port maple assigned,
// so the header always matches the server maple actually started) and a stop func.
// The URL is "" when the portal can't be started (no script, no free port, or the
// script failed — e.g. python3 missing). Honours MAPLE_DESIGN_PORT.
func startDesignPortal() (string, func()) {
	noop := func() {}
	port := findFreePort()
	if port == 0 {
		return "", noop
	}
	root, err := os.Getwd()
	if err != nil {
		return "", noop
	}
	token := portalToken()
	srv := portal.New(root, token)
	// Bind all interfaces (token-gated) so the portal is reachable on the LAN for review.
	go func() { _ = srv.Serve(fmt.Sprintf(":%d", port)) }()

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	_ = os.MkdirAll(".claude/state", 0o755)
	_ = os.WriteFile(".claude/state/design-portal.url", []byte(url+"\n"), 0o644)
	_ = os.WriteFile(".claude/state/design-portal.token", []byte(token), 0o644)
	return url, noop
}

// runPortalServe serves the design portal standalone: `maple portal serve [port]`.
func runPortalServe(args []string) {
	port := 0
	if len(args) >= 2 && args[0] == "serve" {
		port, _ = strconv.Atoi(args[1])
	}
	if port == 0 {
		port = findFreePort()
	}
	root, _ := os.Getwd()
	token := portalToken()
	_ = os.MkdirAll(".claude/state", 0o755)
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	_ = os.WriteFile(".claude/state/design-portal.url", []byte(url+"\n"), 0o644)
	_ = os.WriteFile(".claude/state/design-portal.token", []byte(token), 0o644)
	fmt.Printf("%s design portal: %s\n", brand.Leaf, url)
	if err := portal.New(root, token).Serve(fmt.Sprintf(":%d", port)); err != nil {
		die(err)
	}
}

// portalToken returns a random access token for the design portal.
func portalToken() string {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		return "maple-portal-token"
	}
	return hex.EncodeToString(b)
}

// startHeartbeat writes .claude/state/maple-alive with the current time every 2s and
// removes it when the session ends, so the design portal can tell whether maple is
// still connected (and show "maple offline" if it was closed/quit). Returns a stop
// func that removes the file.
func startHeartbeat() func() {
	const path = ".claude/state/maple-alive"
	beat := func() {
		_ = os.MkdirAll(".claude/state", 0o755)
		_ = os.WriteFile(path, []byte(nowRFC3339()+"\n"), 0o644)
	}
	beat()
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				beat()
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			_ = os.Remove(path)
		})
	}
}

// findFreePort returns a free TCP port in [7800,7900), or the MAPLE_DESIGN_PORT
// override, or 0 if none is available.
func findFreePort() int {
	if v := os.Getenv("MAPLE_DESIGN_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			return p
		}
	}
	for port := 7800; port < 7900; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

// runDashboardLoop runs the dashboard, and when it quits requesting a follow-up
// workflow (req/update/labels/project) runs that in-process and re-enters. Harness
// launches never come through here — they run in the current terminal via ExecProcess.
func runDashboardLoop() {
	portalURL, stopPortal := startDesignPortal()
	stopBeat := startHeartbeat()
	stopHold := portalsock.Hold(".") // live connectivity for the portal
	cleanup := func() { stopHold(); stopBeat(); stopPortal() }
	defer cleanup()
	// Also clean up if the terminal goes away (SIGHUP) or we're terminated, so the
	// portal server isn't orphaned and the portal sees maple go offline immediately.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigCh
		cleanup()
		os.Exit(0)
	}()
	defer signal.Stop(sigCh)
	for {
		model, err := dashboard.New(version, state.NewFS("."), portalURL)
		if err != nil {
			die(err)
		}
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
		final, err := p.Run()
		if err != nil {
			die(err)
		}
		action := dashboard.ExitNone
		if dm, ok := final.(dashboard.Model); ok {
			action = dm.Action()
		}
		switch action {
		case dashboard.ExitReq:
			if err := reqtui.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "req:", err)
			}
		case dashboard.ExitUpdate:
			runUpdate(false, false)
		case dashboard.ExitLabels:
			if err := gh.RunLabels(); err != nil {
				fmt.Fprintln(os.Stderr, "labels:", err)
			}
		case dashboard.ExitProject:
			if err := gh.RunProject(); err != nil {
				fmt.Fprintln(os.Stderr, "project:", err)
			}
		default:
			return
		}
	}
}

// isTTY reports whether stdin is an interactive terminal (not a pipe or /dev/null).
func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// templateFS returns the embedded MAPLE template tree.
func templateFS() fs.FS {
	tpl, err := fs.Sub(embeddedTemplate, "template")
	if err != nil {
		die(err)
	}
	return tpl
}

func runInit(force bool) {
	cwd, err := os.Getwd()
	if err != nil {
		die(err)
	}
	if err := safeInitDir(cwd); err != nil {
		die(err)
	}
	written, err := scaffold.Run(templateFS(), cwd, force, nowRFC3339())
	if err != nil {
		die(err)
	}
	for _, w := range written {
		fmt.Println("  ✓", w)
	}
	fmt.Printf("maple: initialised %d file(s) in %s\n", len(written), cwd)
}

// safeInitDir refuses to scaffold ~600 template files into a directory that is almost
// certainly a mistake — your home directory or a filesystem root — so a wrong cwd (e.g.
// a tmux session that inherited the server's cwd) can never carpet-bomb $HOME.
func safeInitDir(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if home, _ := os.UserHomeDir(); home != "" && abs == filepath.Clean(home) {
		return fmt.Errorf("refusing to scaffold into your home directory (%s) — cd into an empty project directory first", abs)
	}
	if abs == "/" || abs == filepath.VolumeName(abs)+string(filepath.Separator) {
		return fmt.Errorf("refusing to scaffold into the filesystem root (%s)", abs)
	}
	return nil
}

// runUpdate previews the files an update would add/replace/patch, shows a diff on
// request, and applies only after the user confirms. --yes applies without prompting;
// --diff prints every diff up front.
func runUpdate(autoYes, showDiff bool) {
	cwd, err := os.Getwd()
	if err != nil {
		die(err)
	}
	if !initialized() {
		fmt.Println("maple: no project here — run `maple init` first")
		return
	}
	plan, err := scaffold.PlanUpdate(templateFS(), cwd)
	if err != nil {
		die(err)
	}
	if plan.Empty() {
		fmt.Println(brand.Leaf + " maple: already up to date — nothing to change")
		return
	}
	printUpdatePlan(plan)
	if showDiff {
		printUpdateDiffs(plan)
	}

	if autoYes {
		applyUpdate(plan, cwd)
		return
	}
	if !isTTY() {
		fmt.Println("\nnot a TTY — re-run in a terminal, or `maple update --yes` to apply.")
		return
	}
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nApply update? [y]es / [n]o / [d]iff: ")
		line, _ := in.ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
			applyUpdate(plan, cwd)
			return
		case "d", "diff":
			printUpdateDiffs(plan)
		case "n", "no", "":
			fmt.Println("update cancelled — nothing changed")
			return
		}
	}
}

func applyUpdate(plan *scaffold.Plan, cwd string) {
	written, err := plan.Apply(cwd)
	if err != nil {
		die(err)
	}
	fmt.Printf("✓ updated %d file(s)\n", len(written))
}

// printUpdatePlan lists the planned changes grouped by kind, with counts.
func printUpdatePlan(plan *scaffold.Plan) {
	added, changed, patched := plan.Summary()
	fmt.Printf("%s maple update — %d add · %d replace · %d patch\n\n", brand.Leaf, added, changed, patched)
	for _, c := range plan.Changes {
		fmt.Printf("  %-8s %s\n", c.Kind.String(), c.Path)
	}
	fmt.Println("\nproject.config.yaml and your Makefile targets are preserved.")
}

// printUpdateDiffs shows a unified diff per changed/patched file via the system diff.
func printUpdateDiffs(plan *scaffold.Plan) {
	for _, c := range plan.Changes {
		if c.Kind == scaffold.Added {
			fmt.Printf("\n=== %s (new file, %d bytes) ===\n", c.Path, len(c.New))
			continue
		}
		fmt.Printf("\n=== %s (%s) ===\n", c.Path, c.Kind.String())
		fmt.Print(unifiedDiff(c.Path, c.Old, c.New))
	}
}

// unifiedDiff renders a `diff -u` between old and new via the system diff tool, or a
// terse note when diff is unavailable.
func unifiedDiff(path string, oldB, newB []byte) string {
	a, err1 := os.CreateTemp("", "maple-old-*")
	b, err2 := os.CreateTemp("", "maple-new-*")
	if err1 != nil || err2 != nil {
		return "  (diff unavailable)\n"
	}
	defer os.Remove(a.Name())
	defer os.Remove(b.Name())
	a.Write(oldB)
	a.Close()
	b.Write(newB)
	b.Close()
	out, err := exec.Command("diff", "-u", "--label", "current/"+path, "--label", "new/"+path, a.Name(), b.Name()).CombinedOutput()
	if err != nil && len(out) == 0 {
		return "  (diff tool unavailable)\n"
	}
	return string(out)
}

func usage(w *os.File) {
	fmt.Fprint(w, `usage: maple [command]

  (no command)      run the dashboard TUI
  init [--force]    scaffold the MAPLE template into the current project
  req               turn a free-text requirement into Gherkin stories (needs an AI harness)
  update [--yes]    preview + apply template changes (Makefile section-patched)
  labels            bootstrap the canonical MAPLE label set (needs gh)
  project           create a GitHub Project v2 and wire project.config.yaml (needs gh)
  resume [harness]  re-launch a pinned session (claude/copilot/opencode/cursor)
  self-update       replace the binary with the latest GitHub release
  rtk-audit         show RTK hook wiring and token savings
  emit <event…>     push a live event to the design portal (agents: progress/stage)
  version           print the version
`)
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// runEmit pushes a real-time event to the design portal's control socket, so an agent
// can surface progress live. Accepts a raw JSON object, or `<event> [key=value …]`.
func runEmit(args []string) {
	if len(args) == 0 {
		die(fmt.Errorf("usage: maple emit <event> [key=value …]  |  maple emit '<json>'"))
	}
	ev := map[string]any{}
	if strings.HasPrefix(strings.TrimSpace(args[0]), "{") {
		if err := json.Unmarshal([]byte(strings.Join(args, " ")), &ev); err != nil {
			die(fmt.Errorf("emit: invalid JSON: %w", err))
		}
	} else {
		ev["event"] = args[0]
		for _, kv := range args[1:] {
			if k, v, ok := strings.Cut(kv, "="); ok {
				ev[k] = v
			}
		}
	}
	if _, ok := ev["ts"]; !ok {
		ev["ts"] = nowRFC3339()
	}
	// Append to the portal's activity log; the Go portal tails it and pushes over SSE.
	// File-based so it works whether or not the portal is currently running.
	if err := portal.AppendActivity(".", ev); err != nil {
		die(fmt.Errorf("emit: %w", err))
	}
	fmt.Printf("emitted %v\n", ev["event"])
}

// runRTKAudit shows RTK hook wiring, token savings, and the audit log.
func runRTKAudit() {
	rtkPath, err := exec.LookPath("rtk")
	if err != nil {
		die(fmt.Errorf("rtk not found — install with: maple init"))
	}
	for _, sub := range [][]string{{"init", "--show"}, {"gain"}, {"hook-audit"}} {
		cmd := exec.Command(rtkPath, sub...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		fmt.Println()
	}
}

func nowRFC3339() string { return time.Now().Format(time.RFC3339) }

func die(err error) {
	fmt.Fprintln(os.Stderr, "maple:", err)
	os.Exit(1)
}
