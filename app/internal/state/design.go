package state

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DesignTree returns an indented listing of docs/design (📁 dirs, 📄 files),
// skipping .gitkeep. Empty when the directory is absent.
func (s *FS) DesignTree() []string {
	base := filepath.Join(s.Root, "docs", "design")
	var out []string
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Name() == ".gitkeep" {
			return nil
		}
		rel, e := filepath.Rel(base, path)
		if e != nil || rel == "." {
			return nil
		}
		indent := strings.Repeat("  ", strings.Count(rel, string(os.PathSeparator)))
		icon := "📄 "
		if d.IsDir() {
			icon = "📁 "
		}
		out = append(out, indent+icon+d.Name())
		return nil
	})
	return out
}
