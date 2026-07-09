package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChangeKind describes how an update would affect a file.
type ChangeKind int

const (
	Added   ChangeKind = iota // file does not exist yet — will be created
	Changed                   // managed file differs — will be overwritten
	Patched                   // Makefile: only the MAPLE-managed section is spliced in
)

func (k ChangeKind) String() string {
	switch k {
	case Added:
		return "add"
	case Patched:
		return "patch"
	default:
		return "replace"
	}
}

// Change is one file the update would write.
type Change struct {
	Path string
	Kind ChangeKind
	Old  []byte // current on-disk content ("" for Added)
	New  []byte // content that would be written
}

// Plan is the set of changes an update would make. project.config.yaml is never
// touched, and files whose content already matches the template are omitted.
type Plan struct{ Changes []Change }

// Empty reports whether the project is already up to date.
func (p *Plan) Empty() bool { return len(p.Changes) == 0 }

const (
	makefileBegin = "# ─── BEGIN MAPLE MANAGED"
	makefileEnd   = "# ─── END MAPLE MANAGED"
)

// PlanUpdate compares the template against the project at cwd and returns what an
// update would change, without writing anything. The Makefile is patched (managed
// section only) so user targets survive; everything else is replace-if-different.
func PlanUpdate(tpl fs.FS, cwd string) (*Plan, error) {
	var changes []Change
	err := fs.WalkDir(tpl, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == "." || d.IsDir() || p == "project.config.yaml" {
			return err
		}
		tplData, rerr := fs.ReadFile(tpl, p)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(cwd, filepath.FromSlash(p))
		cur, statErr := os.ReadFile(target)
		exists := statErr == nil

		if p == "Makefile" {
			newData := patchMakefile(tplData, cur, exists)
			switch {
			case !exists:
				changes = append(changes, Change{Path: p, Kind: Added, New: newData})
			case !bytes.Equal(cur, newData):
				changes = append(changes, Change{Path: p, Kind: Patched, Old: cur, New: newData})
			}
			return nil
		}

		switch {
		case !exists:
			changes = append(changes, Change{Path: p, Kind: Added, New: tplData})
		case !bytes.Equal(cur, tplData):
			changes = append(changes, Change{Path: p, Kind: Changed, Old: cur, New: tplData})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return &Plan{Changes: changes}, nil
}

// Apply writes every change under cwd and returns the repo-relative paths written.
func (p *Plan) Apply(cwd string) ([]string, error) {
	var written []string
	for _, c := range p.Changes {
		target := filepath.Join(cwd, filepath.FromSlash(c.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(target, c.New, 0o644); err != nil {
			return written, err
		}
		written = append(written, c.Path)
	}
	return written, nil
}

// patchMakefile returns the Makefile content after splicing the template's managed
// section into the current file. When the file doesn't exist yet it returns the full
// template. When markers are absent from the current file the section is appended.
func patchMakefile(tplData, cur []byte, exists bool) []byte {
	tmpl := string(tplData)
	ts := strings.Index(tmpl, makefileBegin)
	te := strings.Index(tmpl, makefileEnd)
	if ts < 0 || te < 0 {
		return tplData // template has no markers — treat as a plain replace
	}
	section := tmpl[ts : te+len(makefileEnd)]

	if !exists {
		return tplData
	}
	c := string(cur)
	es := strings.Index(c, makefileBegin)
	ee := strings.Index(c, makefileEnd)
	if es >= 0 && ee >= 0 {
		c = c[:es] + section + c[ee+len(makefileEnd):]
	} else {
		c = strings.TrimRight(c, "\n") + "\n\n" + section + "\n"
	}
	return []byte(c)
}

// Summary returns per-kind counts for a plan.
func (p *Plan) Summary() (added, changed, patched int) {
	for _, c := range p.Changes {
		switch c.Kind {
		case Added:
			added++
		case Patched:
			patched++
		default:
			changed++
		}
	}
	return
}

// fmtChange renders a one-line plan entry (kept out of the CLI so it can be tested).
func fmtChange(c Change) string {
	return fmt.Sprintf("  %-8s %s", c.Kind.String(), c.Path)
}
