package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

var version = "dev" // overridden at build time: -ldflags "-X main.version=vX.Y.Z"

func main() {
	args := os.Args[1:]
	tplFS, _ := resolveTemplateFS()

	noAnimate := contains(args, "--no-animate")
	// strip --no-animate so subcommand parsing below is unaffected
	var cleanArgs []string
	for _, a := range args {
		if a != "--no-animate" {
			cleanArgs = append(cleanArgs, a)
		}
	}
	args = cleanArgs

	// No args → interactive TUI (boot check + dashboard if initialized, menu otherwise)
	if len(args) == 0 {
		runInteractive(tplFS, noAnimate)
		return
	}

	tools := Detect()

	// Subcommand mode
	switch args[0] {
	case "--version", "-v", "version":
		fmt.Println("maple " + version)

	case "--help", "-h", "help":
		printHelp()

	case "init":
		force := contains(args[1:], "--force")
		if err := runInit(tools, tplFS, force); err != nil {
			fatalf("init: %v", err)
		}
		// Auto-open dashboard after successful init
		if isStdinTTY() && hasTTY() {
			runInteractive(tplFS, noAnimate)
		}

	case "req":
		if err := runReq(tools); err != nil {
			fatalf("req: %v", err)
		}

	case "labels":
		if err := runLabels(tools.GH); err != nil {
			fatalf("labels: %v", err)
		}

	case "project":
		if err := runProject(tools.GH); err != nil {
			fatalf("project: %v", err)
		}

	case "update":
		if err := runInit(tools, tplFS, true); err != nil {
			fatalf("update: %v", err)
		}

	case "self-update", "upgrade":
		if err := selfUpdate(); err != nil {
			fatalf("self-update: %v", err)
		}

	case "resume-session", "resume":
		harness := ""
		if len(args) > 1 {
			harness = args[1]
		}
		if err := runResumeSession(harness); err != nil {
			fatalf("resume-session: %v", err)
		}

	case "rtk-audit":
		rtkPath, err := exec.LookPath("rtk")
		if err != nil {
			fatalf("rtk not found — install with: maple init")
		}
		// Show hook wiring status.
		showCmd := exec.Command(rtkPath, "init", "--show")
		showCmd.Stdout = os.Stdout
		showCmd.Stderr = os.Stderr
		_ = showCmd.Run()
		fmt.Println()
		// Show token savings (works without RTK_HOOK_AUDIT).
		gainCmd := exec.Command(rtkPath, "gain")
		gainCmd.Stdout = os.Stdout
		gainCmd.Stderr = os.Stderr
		_ = gainCmd.Run()
		fmt.Println()
		// Show audit log if it exists (requires RTK_HOOK_AUDIT=1 in a prior session).
		auditCmd := exec.Command(rtkPath, "hook-audit")
		auditCmd.Stdout = os.Stdout
		auditCmd.Stderr = os.Stderr
		_ = auditCmd.Run()

	default:
		fmt.Fprintf(os.Stderr, "maple: unknown command %q\n\n", args[0])
		printHelpStatic()
		os.Exit(1)
	}
}

func runInteractive(fsys fs.FS, noAnimate bool) {
	if !isStdinTTY() {
		printHelpStatic()
		return
	}

	// If project is initialized, run boot check then launch dashboard.
	if _, err := os.Stat("project.config.yaml"); err == nil {
		t, ok := runBoot()
		if !ok {
			return
		}
		runDashboardLoop(t, noAnimate, Detect(), fsys)
		return
	}

	// Not yet initialized — show the setup menu.
	for {
		tools := Detect()
		result := runMenu(tools, fsys)
		switch result.action {
		case menuQuit:
			return
		case menuInit:
			if err := runInitFromMenu(tools, fsys, false); err != nil {
				fmt.Fprintf(os.Stderr, "init: %v\n", err)
			}
			// Jump straight to dashboard if init succeeded
			if _, err := os.Stat("project.config.yaml"); err == nil {
				t, ok := runBoot()
				if ok {
					runDashboardLoop(t, noAnimate, tools, fsys)
				}
				return
			}
		case menuUpdate:
			if err := runInitFromMenu(tools, fsys, true); err != nil {
				fmt.Fprintf(os.Stderr, "update: %v\n", err)
			}
		case menuReq:
			if err := runReq(tools); err != nil {
				fmt.Fprintf(os.Stderr, "req: %v\n", err)
			}
		case menuLabels:
			if err := runLabels(tools.GH); err != nil {
				fmt.Fprintf(os.Stderr, "labels: %v\n", err)
			}
		case menuProject:
			if err := runProject(tools.GH); err != nil {
				fmt.Fprintf(os.Stderr, "project: %v\n", err)
			}
		case menuHelp:
			// handled inline in menu
		}
	}
}

func runDashboardLoop(t Theme, noAnimate bool, tools Tools, fsys fs.FS) {
	port := findFreePort()

	// Start the design portal once for the lifetime of this session.
	if port > 0 {
		if _, err := os.Stat("scripts/design-review-portal.sh"); err == nil {
			_ = exec.Command("bash", "scripts/design-review-portal.sh", "start", fmt.Sprintf("%d", port)).Run()
		}
	}

	// Stop portal on clean exit or SIGTERM/SIGINT.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		stopDesignPortal()
		os.Exit(0)
	}()
	defer func() {
		signal.Stop(sigCh)
		stopDesignPortal()
	}()

	for {
		action, openTarget, err := runDashboard(t, noAnimate, port)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dashboard: %v\n", err)
			return
		}
		switch action {
		case dashActionOpenAgent, dashActionLaunch:
			if len(openTarget) > 0 {
				if err := spawnInNewTerminal(openTarget); err != nil {
					// fallback: suspend maple, run in foreground, resume when done
					fmt.Printf("\n  → launching %s  (maple resumes when it exits)\n\n", openTarget[0])
					cmd := exec.Command(openTarget[0], openTarget[1:]...)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					_ = cmd.Run()
					fmt.Println("\n  ← back to maple")
				}
				tools = Detect()
			}
			// loop back in both cases — maple dashboard restarts
		case dashActionReq:
			if err := runReq(tools); err != nil {
				fmt.Fprintf(os.Stderr, "req: %v\n", err)
			}
		case dashActionUpdate:
			if err := runInitFromMenu(tools, fsys, true); err != nil {
				fmt.Fprintf(os.Stderr, "update: %v\n", err)
			}
		case dashActionLabels:
			if err := runLabels(tools.GH); err != nil {
				fmt.Fprintf(os.Stderr, "labels: %v\n", err)
			}
		case dashActionProject:
			if err := runProject(tools.GH); err != nil {
				fmt.Fprintf(os.Stderr, "project: %v\n", err)
			}
		default:
			return
		}
		tools = Detect()
	}
}

func printHelp() {
	printLogoAnimated()
	printHelpText()
}

func printHelpStatic() {
	printLogoStatic()
	printHelpText()
}

func printHelpText() {
	fmt.Printf(`maple %s — MAPLE initialiser and project helper

Usage:
  maple                   Launch interactive menu
  maple init              Set up MAPLE in the current directory
  maple update            Re-sync template files (preserves Makefile edits)
  maple req               Write requirements → generate Gherkin story
  maple labels            Bootstrap GitHub label set
  maple project           Create GitHub Project v2
  maple self-update       Upgrade maple to the latest release
  maple resume-session    Resume the pinned session for the project
  maple resume-session claude   Resume specifically the pinned claude session
  maple rtk-audit         Run rtk hook-audit (verify hook wiring + show savings)

  maple --no-animate      Skip logo animations (SSH / slow terminals)
  maple --version         Print version
  maple --help            Show this help
`, version)
}

const mapleRepo = "kinncj/maple"

// latestRelease fetches the latest release tag from GitHub.
func latestRelease() (string, error) {
	apiURL := "https://api.github.com/repos/" + mapleRepo + "/releases/latest"
	resp, err := http.Get(apiURL) //nolint:noctx
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.TagName, nil
}

// selfUpdate downloads and replaces the running binary with the latest release.
func selfUpdate() error {
	latest, err := latestRelease()
	if err != nil {
		return fmt.Errorf("could not fetch latest release: %w", err)
	}
	if strings.TrimPrefix(latest, "v") == strings.TrimPrefix(version, "v") {
		fmt.Println("maple " + version + " is already up to date.")
		return nil
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	archive := fmt.Sprintf("maple-%s-%s.tar.gz", goos, goarch)
	dlURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", mapleRepo, latest, archive)

	fmt.Printf("Updating maple %s → %s\n", version, latest)

	resp, err := http.Get(dlURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d for %s", resp.StatusCode, dlURL)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("could not resolve symlinks: %w", err)
	}

	tmp, err := os.CreateTemp("", "maple-update-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	tmp.Close()

	// Extract the maple binary from the tar and write alongside current exe
	newBin := exe + ".new"
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInstall)
	defer cancel()
	extractCmd := exec.CommandContext(ctx, "tar", "-xzf", tmp.Name(), "-O", "maple")
	newFile, err := os.OpenFile(newBin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	extractCmd.Stdout = newFile
	if err := extractCmd.Run(); err != nil {
		newFile.Close()
		os.Remove(newBin)
		return fmt.Errorf("extract failed: %w", err)
	}
	newFile.Close()

	if err := os.Rename(newBin, exe); err != nil {
		os.Remove(newBin)
		return fmt.Errorf("could not replace binary (try with sudo): %w", err)
	}
	fmt.Printf("✓ maple updated to %s\n", latest)
	return nil
}

// resolveTemplateFS resolves the template source as an fs.FS.
// Resolution order:
//  1. MAPLE_TEMPLATE env var (resolved to absolute path, uses OS filesystem)
//  2. <binary_dir>/template/ if it exists on disk (dev checkout)
//  3. ./template/ in cwd if exists (running from repo root in dev)
//  4. Embedded FS (always works for released binaries)
func resolveTemplateFS() (fs.FS, string) {
	if v := os.Getenv("MAPLE_TEMPLATE"); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			v = abs
		}
		return os.DirFS(v), v
	}
	exe, _ := os.Executable()
	for _, c := range []string{
		filepath.Join(filepath.Dir(exe), "template"),
		"template",
	} {
		if stat, err := os.Stat(c); err == nil && stat.IsDir() {
			abs, _ := filepath.Abs(c)
			return os.DirFS(abs), abs
		}
	}
	sub, _ := fs.Sub(embeddedTemplate, "template")
	return sub, "(embedded)"
}

// runResumeSession reads .claude/state/sessions.json and launches the pinned
// session for the given harness. If harness is "", it prefers detected harness order.
func runResumeSession(harness string) error {
	data, err := os.ReadFile(".claude/state/sessions.json")
	if err != nil {
		return fmt.Errorf("no sessions file — use the TUI [o] key to pin a session first\n  (expected .claude/state/sessions.json)")
	}
	var sessions map[string]string
	if err := json.Unmarshal(data, &sessions); err != nil {
		return fmt.Errorf("corrupt sessions file: %w", err)
	}
	if len(sessions) == 0 {
		return fmt.Errorf("sessions.json is empty — navigate to the Agents pane and press [o] or [p]")
	}

	if harness == "" {
		for _, pref := range []string{"claude", "copilot", "opencode", "cursor"} {
			if sessions[pref] != "" {
				harness = pref
				break
			}
		}
	}
	if harness == "" {
		for k := range sessions {
			harness = k
			break
		}
	}

	id := sessions[harness]
	if id == "" {
		var available []string
		for k, v := range sessions {
			if v != "" {
				available = append(available, k)
			}
		}
		return fmt.Errorf("no pinned session for %q\n  available: %s", harness, strings.Join(available, ", "))
	}

	var args []string
	switch harness {
	case "claude":
		args = []string{"claude", "--resume", id}
	case "opencode":
		args = []string{"opencode", "--session", id}
	case "copilot":
		args = []string{"copilot", "--resume=" + id}
	case "cursor":
		cursorBin := "cursor-agent"
		if _, err := exec.LookPath(cursorBin); err != nil {
			cursorBin = "cursor"
		}
		args = []string{cursorBin}
	default:
		return fmt.Errorf("unknown harness %q — supported: claude, copilot, opencode, cursor", harness)
	}

	short := id
	if len(short) > 8 {
		short = short[:8] + "…"
	}
	fmt.Printf("resuming %s session %s\n", harness, short)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if rtkPath, err := exec.LookPath("rtk"); err == nil && rtkPath != "" {
		cmd.Env = append(cmd.Env, "RTK_HOOK_AUDIT=1")
	}
	return cmd.Run()
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "maple: "+format+"\n", args...)
	if runtime.GOOS == "windows" {
		os.Exit(1)
	}
	os.Exit(1)
}
