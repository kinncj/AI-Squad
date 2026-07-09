package state

import "testing"

func TestDiscoverTestsTagsFrameworksAndSkipsVendored(t *testing.T) {
	got := discoverTests("testdata/qa")
	want := map[string]string{
		"app_test.go":    "go",
		"button.test.ts": "js",
		"login.feature":  "bdd",
		"test_auth.py":   "py",
		"user_spec.rb":   "rb",
	}
	if len(got) != len(want) {
		t.Fatalf("discovered %d tests, want %d: %+v", len(got), len(want), got)
	}
	for _, tst := range got {
		if want[tst.Path] != tst.Framework {
			t.Errorf("%s tagged %q, want %q", tst.Path, tst.Framework, want[tst.Path])
		}
	}
}

func TestDiscoverTestsSorted(t *testing.T) {
	got := discoverTests("testdata/qa")
	for i := 1; i < len(got); i++ {
		if got[i-1].Path > got[i].Path {
			t.Errorf("results not sorted: %q before %q", got[i-1].Path, got[i].Path)
		}
	}
}

func TestFrameworkFor(t *testing.T) {
	cases := map[string]string{
		"x_test.go":  "go",
		"a.feature":  "bdd",
		"b.spec.tsx": "js",
		"c.test.js":  "js",
		"test_d.py":  "py",
		"e_test.py":  "py",
		"f_spec.rb":  "rb",
		"main.go":    "",
		"README.md":  "",
		"notes.txt":  "",
	}
	for name, want := range cases {
		if got := frameworkFor(name); got != want {
			t.Errorf("frameworkFor(%q) = %q, want %q", name, got, want)
		}
	}
}
