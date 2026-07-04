// Package selfupdate implements `maple self-update`: it fetches the latest GitHub
// release and replaces the running binary in place. Ported from tui/main.go.
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	mapleRepo      = "kinncj/maple"
	installTimeout = 300 * time.Second
)

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

// Run downloads the latest release and replaces the running binary. version is the
// current build version.
func Run(version string) error {
	latest, err := latestRelease()
	if err != nil {
		return fmt.Errorf("could not fetch latest release: %w", err)
	}
	if strings.TrimPrefix(latest, "v") == strings.TrimPrefix(version, "v") {
		fmt.Println("maple " + version + " is already up to date.")
		return nil
	}

	archive := fmt.Sprintf("maple-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
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

	newBin := exe + ".new"
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
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
