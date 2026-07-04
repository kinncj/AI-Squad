package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectName reads project.name from project.config.yaml, or "" if absent.
func (s *FS) ProjectName() string {
	data, err := os.ReadFile(filepath.Join(s.Root, "project.config.yaml"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "name:") {
			v := strings.TrimSpace(strings.TrimPrefix(t, "name:"))
			return strings.Trim(strings.SplitN(v, "#", 2)[0], " \"'")
		}
	}
	return ""
}

// TaffyCount counts the named taffy workflows in .claude/taffy/*.yaml, excluding
// the schema file.
func (s *FS) TaffyCount() int {
	files, _ := filepath.Glob(filepath.Join(s.Root, ".claude", "taffy", "*.yaml"))
	n := 0
	for _, f := range files {
		if filepath.Base(f) == "schema.yaml" {
			continue
		}
		n++
	}
	return n
}

// PipelineStatus reads status from .claude/state/maple.json, or "" if absent.
func (s *FS) PipelineStatus() string {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "maple.json"))
	if err != nil {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return ""
	}
	if v, ok := m["status"].(string); ok {
		return v
	}
	return ""
}

// InFlight reports whether a pipeline status represents active work.
func InFlight(status string) bool {
	switch strings.ToUpper(status) {
	case "RUNNING", "PAUSED", "RATE_LIMITED":
		return true
	}
	return false
}
