package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const rtkRepo = "rtk-ai/rtk"

// installRTK downloads and installs the rtk binary next to the maple binary.
// Returns the path to the installed binary, or an error.
// If rtk is already on PATH, returns the existing path immediately.
//
// Fallback chain:
//  1. GitHub release tarball (every platform)
//  2. Upstream install.sh via curl|sh (linux/macos)
//  3. cargo install --git (any platform where cargo is on PATH)
func installRTK() (string, error) {
	if p, err := exec.LookPath("rtk"); err == nil {
		return p, nil
	}

	var errs []string

	if p, err := installRTKFromRelease(); err == nil {
		return p, nil
	} else {
		errs = append(errs, "release: "+err.Error())
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if p, err := installRTKFromUpstreamScript(); err == nil {
			return p, nil
		} else {
			errs = append(errs, "upstream-script: "+err.Error())
		}
	}

	if p, err := installRTKFromCargo(); err == nil {
		return p, nil
	} else {
		errs = append(errs, "cargo: "+err.Error())
	}

	return "", fmt.Errorf("all rtk install paths failed — %s", strings.Join(errs, "; "))
}

// installRTKFromRelease downloads the platform tarball from rtk-ai's GitHub
// releases and copies the binary next to the maple binary.
func installRTKFromRelease() (string, error) {
	version, err := latestRTKVersion()
	if err != nil {
		return "", fmt.Errorf("could not resolve latest rtk version: %w", err)
	}

	triple, ext, err := rtkPlatformTriple()
	if err != nil {
		return "", err
	}

	archiveName := fmt.Sprintf("rtk-%s.%s", triple, ext)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", rtkRepo, version, archiveName)

	tmp, err := os.CreateTemp("", "rtk-download-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed: HTTP %d for %s", resp.StatusCode, url)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}
	tmp.Close()

	binData, err := extractRTKBinary(tmp.Name(), ext)
	if err != nil {
		return "", fmt.Errorf("extract failed: %w", err)
	}

	dest := rtkInstallPath()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, binData, 0o755); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}
	return dest, nil
}

// rtkUpstreamInstallURL is the canonical linux/macos install script from rtk-ai.
const rtkUpstreamInstallURL = "https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh"

// installRTKFromUpstreamScript pipes the rtk-ai upstream install.sh to sh.
// After the script exits, searches PATH and common install locations for rtk.
func installRTKFromUpstreamScript() (string, error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return "", fmt.Errorf("upstream install script is linux/macos only")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutInstall)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "curl -fsSL "+rtkUpstreamInstallURL+" | sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("upstream install script timed out after %s", timeoutInstall)
		}
		return "", fmt.Errorf("upstream install script failed: %w", err)
	}
	return findInstalledRTK()
}

// installRTKFromCargo runs `cargo install --git https://github.com/rtk-ai/rtk`.
// Works on any platform where cargo is on PATH — rtk-ai's officially documented
// Windows install path, plus a universal fallback when binary release download
// is blocked (proxies, corp firewalls, missing platform triple).
func installRTKFromCargo() (string, error) {
	cargo, err := exec.LookPath("cargo")
	if err != nil {
		return "", fmt.Errorf("cargo not found on PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInstall)
	defer cancel()
	cmd := exec.CommandContext(ctx, cargo, "install", "--git", "https://github.com/rtk-ai/rtk", "--quiet")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("cargo install timed out after %s", timeoutInstall)
		}
		return "", fmt.Errorf("cargo install failed: %w", err)
	}
	return findInstalledRTK()
}

// findInstalledRTK searches PATH and well-known install locations for the rtk binary.
func findInstalledRTK() (string, error) {
	if p, err := exec.LookPath("rtk"); err == nil {
		return p, nil
	}
	name := rtkBinaryName()
	var candidates []string
	if runtime.GOOS == "windows" {
		if up := os.Getenv("USERPROFILE"); up != "" {
			candidates = append(candidates,
				filepath.Join(up, ".cargo", "bin", name),
				filepath.Join(up, ".tools", "maple", "bin", name),
			)
		}
	} else {
		home := os.Getenv("HOME")
		candidates = append(candidates,
			filepath.Join(home, ".cargo", "bin", name),
			filepath.Join(home, ".local", "bin", name),
			"/usr/local/bin/"+name,
		)
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("rtk binary not found on PATH or in well-known install locations — you may need to start a new shell")
}

// latestRTKVersion fetches the highest semver tag from the RTK releases API.
func latestRTKVersion() (string, error) {
	url := "https://api.github.com/repos/" + rtkRepo + "/releases?per_page=100"
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}
	var tags []string
	for _, r := range releases {
		if r.TagName != "" {
			tags = append(tags, r.TagName)
		}
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("no releases found")
	}
	sort.Slice(tags, func(i, j int) bool {
		return compareSemver(tags[i], tags[j]) < 0
	})
	return tags[len(tags)-1], nil
}

// rtkPlatformTriple returns the release asset triple and archive extension for the current OS/arch.
func rtkPlatformTriple() (triple, ext string, err error) {
	os_ := runtime.GOOS
	arch := runtime.GOARCH
	switch os_ + "/" + arch {
	case "darwin/arm64":
		return "aarch64-apple-darwin", "tar.gz", nil
	case "darwin/amd64":
		return "x86_64-apple-darwin", "tar.gz", nil
	case "linux/arm64":
		return "aarch64-unknown-linux-gnu", "tar.gz", nil
	case "linux/amd64":
		return "x86_64-unknown-linux-gnu", "tar.gz", nil
	case "windows/amd64":
		return "x86_64-pc-windows-msvc", "zip", nil
	default:
		return "", "", fmt.Errorf("rtk: unsupported platform %s/%s", os_, arch)
	}
}

// rtkInstallPath returns where to install the rtk binary — same directory as maple.
func rtkInstallPath() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join(os.Getenv("HOME"), ".tools", "maple", "bin", rtkBinaryName())
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Join(filepath.Dir(exe), rtkBinaryName())
}

func rtkBinaryName() string {
	if runtime.GOOS == "windows" {
		return "rtk.exe"
	}
	return "rtk"
}

// extractRTKBinary finds and returns the raw bytes of the rtk binary inside the archive.
func extractRTKBinary(archivePath, ext string) ([]byte, error) {
	switch ext {
	case "tar.gz":
		return extractFromTarGz(archivePath)
	case "zip":
		return extractFromZip(archivePath)
	default:
		return nil, fmt.Errorf("unknown archive extension: %s", ext)
	}
}

func extractFromTarGz(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		base := filepath.Base(hdr.Name)
		if base == "rtk" || base == "rtk.exe" || isRTKBinaryName(hdr.Name) {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("rtk binary not found in archive")
}

func extractFromZip(path string) ([]byte, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base == "rtk" || base == "rtk.exe" || isRTKBinaryName(f.Name) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("rtk binary not found in archive")
}

// isRTKBinaryName matches binaries named after platform triples (e.g. rtk-x86_64-unknown-linux-gnu).
func isRTKBinaryName(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return strings.HasPrefix(base, "rtk") && !strings.Contains(base, ".")
}

// compareSemver compares two version strings lexicographically after stripping "v" prefix.
// Returns negative if a < b, zero if equal, positive if a > b.
func compareSemver(a, b string) int {
	normalize := func(s string) []int {
		s = strings.TrimPrefix(s, "v")
		parts := strings.Split(s, ".")
		nums := make([]int, 3)
		for i, p := range parts {
			if i >= 3 {
				break
			}
			n := 0
			for _, c := range p {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				} else {
					break
				}
			}
			nums[i] = n
		}
		return nums
	}
	av, bv := normalize(a), normalize(b)
	for i := range av {
		if av[i] != bv[i] {
			if av[i] < bv[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}
