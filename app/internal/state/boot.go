package state

import (
	"os"
	"path/filepath"
)

// ProjectConfigExists reports whether project.config.yaml is present at the root.
func (s *FS) ProjectConfigExists() bool {
	return exists(filepath.Join(s.Root, "project.config.yaml"))
}

// ClaudeDirExists reports whether the .claude directory is present at the root.
func (s *FS) ClaudeDirExists() bool {
	info, err := os.Stat(filepath.Join(s.Root, ".claude"))
	return err == nil && info.IsDir()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
