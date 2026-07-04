package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// pipelineFields are the maple.json keys shown in the pipeline overlay, in order.
var pipelineFields = []string{
	"pipeline", "taffy", "stage", "status", "state",
	"awaiting_approval", "started_at", "updated_at", "ts",
}

// PipelineLines returns the pipeline state from .claude/state/maple.json as aligned
// "key value" lines for the pipeline overlay.
func (s *FS) PipelineLines() []string {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", "maple.json"))
	if err != nil {
		return []string{"(no active pipeline)"}
	}
	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return []string{"(unreadable maple.json)"}
	}
	var out []string
	for _, k := range pipelineFields {
		if v, ok := m[k]; ok && v != nil && v != "" {
			out = append(out, fmt.Sprintf("%-18s %v", k, v))
		}
	}
	if len(out) == 0 {
		out = append(out, "(no pipeline fields)")
	}
	return out
}

// Skills lists the skill directories under .claude/skills, sorted.
func (s *FS) Skills() []string {
	entries, err := os.ReadDir(filepath.Join(s.Root, ".claude", "skills"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}
