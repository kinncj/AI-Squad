// Package scaffold copies the embedded MAPLE template into a project directory —
// the mechanics behind `maple init`. It takes an fs.FS so it can be unit-tested with
// a synthetic filesystem, independent of the go:embed that supplies the real
// template to the binary.
package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Run copies every file in tpl into cwd. Existing files are skipped unless force is
// true. project.config.yaml is generated from projectConfigYAML (never overwritten).
// It returns the repo-relative paths written.
func Run(tpl fs.FS, cwd string, force bool, now string) ([]string, error) {
	var written []string
	err := fs.WalkDir(tpl, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		target := filepath.Join(cwd, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if p == "project.config.yaml" {
			return nil // generated below, not copied verbatim
		}
		if !force {
			if _, err := os.Stat(target); err == nil {
				return nil // keep the user's file
			}
		}
		data, err := fs.ReadFile(tpl, p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		written = append(written, p)
		return nil
	})
	if err != nil {
		return written, err
	}

	cfg := filepath.Join(cwd, "project.config.yaml")
	if _, err := os.Stat(cfg); os.IsNotExist(err) {
		name := filepath.Base(cwd)
		if err := os.WriteFile(cfg, []byte(ProjectConfigYAML(name, now)), 0o644); err != nil {
			return written, err
		}
		written = append(written, "project.config.yaml")
	}
	return written, nil
}

// ProjectConfigYAML renders the generated project.config.yaml. Ported verbatim from
// tui/init.go so a fresh config stays schema-valid (github block matches the schema).
func ProjectConfigYAML(name, createdAt string) string {
	return fmt.Sprintf(`project:
  name: "%s"
  created_at: "%s"

sdlc:
  mode: standard          # standard | spike
  require_adr_for:
    - new_dependency
    - cross_boundary_change
    - data_model_change
    - auth_change
    - mcp_adoption
    - visual_identity_change

qa:
  bdd: cucumber           # cucumber | behave | none

design:
  target: web             # web | tui — UI medium the design phase targets
  ui_library: none        # mantine | tailwind | shadcn | none

github:
  project_number: null    # null = ask once; 0 = declined; N = configured board (maple project bootstrap)
  project_node_id: null    # Set by: maple project bootstrap
  status_field_id: null    # Status single-select field id; cached by gh-projects / maple project bootstrap
  milestone_granularity: null  # null = ask once; none = declined; minor = major+minor; patch = also per-patch
  # Issue label taxonomy (type:bug | type:feature | type:docs | type:refactor | type:chore)
  # lives in the gh-labels-milestones skill, not here. See "Version & Issue Tracking" in CLAUDE.md.
`, name, createdAt)
}
