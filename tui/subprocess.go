package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Default timeouts by call class.
const (
	timeoutAI      = 120 * time.Second // AI harness calls (req)
	timeoutNetwork = 60 * time.Second  // gh / npx network calls
	timeoutInstall = 300 * time.Second // curl|sh, cargo install, tar extract
	timeoutAuth    = 10 * time.Second  // gh auth status at boot — fail fast
	timeoutLocal   = 30 * time.Second  // local helpers (lefthook, rtk init, portal)
)

// runWithTimeout runs name+args with a deadline, returning combined output.
// On timeout it kills the child and returns an error callers can surface as
// "timed out — check auth / network".
func runWithTimeout(d time.Duration, env []string, name string, args ...string) ([]byte, error) {
	return runWithTimeoutStdin(d, env, "", name, args...)
}

// runWithTimeoutStdin is runWithTimeout with a string fed to the child's stdin.
func runWithTimeoutStdin(d time.Duration, env []string, stdin, name string, args ...string) ([]byte, error) {
	return runCtx(context.Background(), d, env, stdin, name, args...)
}

// runCtx runs name+args under parent+deadline. Cancelling parent kills the child;
// a deadline hit is reported as a timeout error. Either path frees the process.
func runCtx(parent context.Context, d time.Duration, env []string, stdin, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, d)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	if env != nil {
		cmd.Env = env
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("%s timed out after %s — check auth / network", name, d)
	}
	return out, err
}
