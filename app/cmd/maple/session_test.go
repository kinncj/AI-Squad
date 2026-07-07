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

func TestEnsureHerdrKittyGraphics(t *testing.T) {
	set := func(p string) { os.Setenv("HERDR_CONFIG_PATH", p); t.Cleanup(func() { os.Unsetenv("HERDR_CONFIG_PATH") }) }

	// 1. no file → created with the setting
	p := filepath.Join(t.TempDir(), "config.toml")
	set(p)
	if !ensureHerdrKittyGraphics() {
		t.Fatal("should create config and report a change")
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), "kitty_graphics = true") {
		t.Errorf("created config missing setting:\n%s", b)
	}

	// 2. already has an explicit setting → untouched
	p2 := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(p2, []byte("[experimental]\nkitty_graphics = false\n"), 0o644)
	os.Setenv("HERDR_CONFIG_PATH", p2)
	if ensureHerdrKittyGraphics() {
		t.Error("must not override an explicit user setting")
	}
	if b, _ := os.ReadFile(p2); strings.Contains(string(b), "= true") {
		t.Error("must not flip a user's false to true")
	}

	// 3. existing [experimental] without the key → inserted under it
	p3 := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(p3, []byte("[experimental]\nallow_nested = true\n"), 0o644)
	os.Setenv("HERDR_CONFIG_PATH", p3)
	if !ensureHerdrKittyGraphics() {
		t.Error("should add the key")
	}
	b, _ := os.ReadFile(p3)
	if !strings.Contains(string(b), "kitty_graphics = true") || strings.Count(string(b), "[experimental]") != 1 {
		t.Errorf("should insert under the existing section without duplicating it:\n%s", b)
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
