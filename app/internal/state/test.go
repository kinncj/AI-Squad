package state

import (
	"context"
	"io/fs"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Test is a discovered test file surfaced in the QA pane.
type Test struct {
	Path      string
	Framework string   // "go", "bdd", "js", "py", "rb"
	RunCmd    []string // argv to run just this test, or nil if unknown
}

// testRunCmd builds the argv to run a single test by framework, relative to the repo root.
func testRunCmd(framework, rel string) []string {
	switch framework {
	case "go":
		return []string{"go", "test", "-v", "./" + filepath.ToSlash(filepath.Dir(rel))}
	case "py":
		return []string{"python3", "-m", "pytest", "-v", rel}
	case "js":
		return []string{"npx", "jest", "--no-coverage", rel}
	case "bdd":
		return []string{"npx", "cucumber-js", rel}
	case "rb":
		return []string{"bundle", "exec", "rspec", rel}
	}
	return nil
}

// RunTest executes a discovered test and returns its output as lines, relative to Root.
func (s *FS) RunTest(t Test) []string { return runTest(s.Root, t.RunCmd) }

// runTest executes a test's run command from root and returns its combined output as lines.
func runTest(root string, cmd []string) []string {
	if len(cmd) == 0 {
		return []string{"(no run command available for this test)"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Dir = root
	out, err := c.CombinedOutput()
	lines := []string{"$ " + strings.Join(cmd, " "), ""}
	lines = append(lines, strings.Split(strings.TrimRight(string(out), "\n"), "\n")...)
	if ctx.Err() == context.DeadlineExceeded {
		lines = append(lines, "", "(timed out after 5m)")
	} else if err != nil {
		lines = append(lines, "", "✗ exit: "+err.Error())
	} else {
		lines = append(lines, "", "✓ passed")
	}
	return lines
}

// skipDirs are never descended into during test discovery.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"bin": true, ".claude": true, ".opencode": true, ".cursor": true,
}

// Tests discovers test files under Root, tagged by framework, sorted by path.
func (s *FS) Tests() []Test { return discoverTests(s.Root) }

// maxWalkEntries bounds test discovery so a wrong root (e.g. $HOME) can never hang the
// dashboard walking millions of files.
const maxWalkEntries = 50000

func discoverTests(root string) []Test {
	var out []Test
	count := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if count++; count > maxWalkEntries {
			return filepath.SkipAll
		}
		if d.IsDir() {
			// Skip vendored/VCS dirs and any hidden dir (tests never live in dotdirs).
			if skipDirs[d.Name()] || (path != root && strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if fw := frameworkFor(d.Name()); fw != "" {
			rel, e := filepath.Rel(root, path)
			if e != nil {
				rel = path
			}
			out = append(out, Test{Path: rel, Framework: fw, RunCmd: testRunCmd(fw, rel)})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// frameworkFor classifies a filename into a test framework tag, or "" if it is not
// a recognised test file.
func frameworkFor(base string) string {
	switch {
	case strings.HasSuffix(base, "_test.go"):
		return "go"
	case strings.HasSuffix(base, ".feature"):
		return "bdd"
	case hasAnySuffix(base, ".test.ts", ".test.js", ".test.tsx", ".test.jsx",
		".spec.ts", ".spec.js", ".spec.tsx", ".spec.jsx"):
		return "js"
	case strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py"),
		strings.HasSuffix(base, "_test.py"):
		return "py"
	case strings.HasSuffix(base, "_spec.rb"), strings.HasSuffix(base, "_test.rb"):
		return "rb"
	default:
		return ""
	}
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}
