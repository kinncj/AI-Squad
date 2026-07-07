package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeSession(t *testing.T) {
	cases := map[string]string{
		"lolz":           "lolz",
		"My Project":     "my-project",
		"foo_bar.baz":    "foo-barbaz",
		"  trailing--":   "trailing",
		"/":              "project",
		"UPPER":          "upper",
		"a!!!b":          "ab",
	}
	for in, want := range cases {
		if got := sanitizeSession(in); got != want {
			t.Errorf("sanitizeSession(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPaneNumOrdering(t *testing.T) {
	if paneNum("w5:p2") != 2 {
		t.Errorf("paneNum(w5:p2) = %d, want 2", paneNum("w5:p2"))
	}
	if !paneLess("w5:p1", "w5:p2") {
		t.Error("p1 should order before p2")
	}
	if paneLess("wA:p10", "wA:p2") {
		t.Error("p2 should order before p10 (numeric, not lexical)")
	}
}

func TestConfigSessionReadAndPersist(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	// no config yet → empty, and persist is a no-op (nothing to append to)
	if configSession() != "" {
		t.Error("no config should yield empty session")
	}
	persistConfigSession("maple-x")
	if _, err := os.Stat("project.config.yaml"); err == nil {
		t.Error("persist must not create a config in an uninitialised dir")
	}

	// with a config lacking the key → persist appends it, then read returns it
	os.WriteFile("project.config.yaml", []byte("project:\n  name: x\n"), 0o644)
	persistConfigSession("maple-lolz")
	if got := configSession(); got != "maple-lolz" {
		t.Errorf("after persist, configSession() = %q, want maple-lolz", got)
	}
	data, _ := os.ReadFile("project.config.yaml")
	if strings.Count(string(data), "session:") != 1 {
		t.Errorf("session key should be written exactly once:\n%s", data)
	}

	// idempotent: a second persist with a different name does not overwrite
	persistConfigSession("maple-other")
	if got := configSession(); got != "maple-lolz" {
		t.Errorf("persist must not overwrite an existing session, got %q", got)
	}
}

func TestMapleSessionNameDerivesFromDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cool Repo")
	os.MkdirAll(dir, 0o755)
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)
	if got := mapleSessionName(dir); got != "maple-cool-repo" {
		t.Errorf("mapleSessionName = %q, want maple-cool-repo", got)
	}
}
