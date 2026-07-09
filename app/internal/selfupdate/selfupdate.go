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

// latestRelease fetches the highest-version release tag from GitHub. It reads the full
// release list (not /releases/latest, which ignores prereleases) and picks the highest
// semver. Prereleases (…-rc1) are included only when includePre is true.
func latestRelease(includePre bool) (string, error) {
	apiURL := "https://api.github.com/repos/" + mapleRepo + "/releases?per_page=100"
	resp, err := http.Get(apiURL) //nolint:noctx
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var releases []struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}
	best := ""
	for _, r := range releases {
		if r.Draft || (r.Prerelease && !includePre) {
			continue
		}
		if !isSemverTag(r.TagName) {
			continue
		}
		if best == "" || semverLess(best, r.TagName) {
			best = r.TagName
		}
	}
	if best == "" {
		return "", fmt.Errorf("no suitable release found (try `maple upgrade --pre` for prereleases)")
	}
	return best, nil
}

// isSemverTag reports whether tag looks like vX.Y.Z or vX.Y.Z-pre.
func isSemverTag(tag string) bool {
	core, _, _ := strings.Cut(strings.TrimPrefix(tag, "v"), "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// semverLess reports whether tag a is a lower version than b. Core numbers compare
// numerically; for equal cores a release (no prerelease) outranks a prerelease, and two
// prereleases compare lexically (rc1 < rc2).
func semverLess(a, b string) bool {
	ac, ap, _ := strings.Cut(strings.TrimPrefix(a, "v"), "-")
	bc, bp, _ := strings.Cut(strings.TrimPrefix(b, "v"), "-")
	an, bn := parseCore(ac), parseCore(bc)
	for i := 0; i < 3; i++ {
		if an[i] != bn[i] {
			return an[i] < bn[i]
		}
	}
	if ap == bp {
		return false
	}
	if ap == "" { // a is a full release, b is a prerelease → a is greater
		return false
	}
	if bp == "" {
		return true
	}
	return ap < bp
}

func parseCore(core string) [3]int {
	var out [3]int
	for i, p := range strings.Split(core, ".") {
		if i > 2 {
			break
		}
		n := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		out[i] = n
	}
	return out
}

// Run downloads the latest release and replaces the running binary. version is the current
// build version; includePre opts into prereleases (also auto-on when the running build is
// itself a prerelease, so an RC user tracks RCs).
func Run(version string, includePre bool) error {
	if strings.Contains(version, "-") {
		includePre = true
	}
	latest, err := latestRelease(includePre)
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
