package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func updTemplate() fstest.MapFS {
	makefile := "# user top\n" +
		makefileBegin + " ─────\nMAPLE-V2:\n\techo v2\n" + makefileEnd + "\n"
	return fstest.MapFS{
		"CLAUDE.md":                 {Data: []byte("# rules v2\n")},
		".claude/agents/x.md":       {Data: []byte("agent v2\n")},
		"Makefile":                  {Data: []byte(makefile)},
		"project.config.yaml":       {Data: []byte("stale\n")},
	}
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPlanUpdateClassifiesChanges(t *testing.T) {
	cwd := t.TempDir()
	// CLAUDE.md exists but differs → Changed. agents/x.md missing → Added.
	write(t, cwd, "CLAUDE.md", "# my edited rules\n")
	// Makefile: user has custom targets + an old managed section.
	write(t, cwd, "Makefile", "# user top\nMY-TARGET:\n\techo mine\n"+
		makefileBegin+" old\nMAPLE-V1:\n\techo v1\n"+makefileEnd+"\n")

	plan, err := PlanUpdate(updTemplate(), cwd)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]Change{}
	for _, c := range plan.Changes {
		byPath[c.Path] = c
	}
	if _, ok := byPath["project.config.yaml"]; ok {
		t.Error("project.config.yaml must never be in the plan")
	}
	if byPath["CLAUDE.md"].Kind != Changed {
		t.Errorf("CLAUDE.md should be Changed, got %v", byPath["CLAUDE.md"].Kind)
	}
	if byPath[".claude/agents/x.md"].Kind != Added {
		t.Errorf("missing agent should be Added, got %v", byPath[".claude/agents/x.md"].Kind)
	}
	mk, ok := byPath["Makefile"]
	if !ok || mk.Kind != Patched {
		t.Fatalf("Makefile should be Patched, got %+v", mk)
	}
	// The patch keeps the user's target and swaps the managed section to v2.
	if !strings.Contains(string(mk.New), "MY-TARGET:") {
		t.Error("Makefile patch must preserve the user's custom target")
	}
	if strings.Contains(string(mk.New), "MAPLE-V1") || !strings.Contains(string(mk.New), "MAPLE-V2") {
		t.Error("Makefile patch must swap the managed section to the template version")
	}
}

func TestPlanUpdateEmptyWhenIdentical(t *testing.T) {
	cwd := t.TempDir()
	tpl := updTemplate()
	// Materialise the template exactly (except config), so nothing should change.
	write(t, cwd, "CLAUDE.md", "# rules v2\n")
	write(t, cwd, ".claude/agents/x.md", "agent v2\n")
	write(t, cwd, "Makefile", string(tpl["Makefile"].Data))
	plan, err := PlanUpdate(tpl, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Empty() {
		t.Errorf("identical project should yield an empty plan, got %d changes", len(plan.Changes))
	}
}

func TestApplyWritesChanges(t *testing.T) {
	cwd := t.TempDir()
	write(t, cwd, "CLAUDE.md", "old\n")
	plan, err := PlanUpdate(updTemplate(), cwd)
	if err != nil {
		t.Fatal(err)
	}
	written, err := plan.Apply(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) == 0 {
		t.Fatal("apply should write files")
	}
	got, _ := os.ReadFile(filepath.Join(cwd, "CLAUDE.md"))
	if string(got) != "# rules v2\n" {
		t.Errorf("CLAUDE.md not updated, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".claude/agents/x.md")); err != nil {
		t.Error("added file should be written")
	}
}
