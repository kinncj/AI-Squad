package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const rtkHarnessFile = "rtk-harnesses.json"

// RTKHarnesses returns which harnesses have rtk wired, from
// .claude/state/rtk-harnesses.json. Empty map when absent.
func (s *FS) RTKHarnesses() map[string]bool {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "state", rtkHarnessFile))
	if err != nil {
		return map[string]bool{}
	}
	var m map[string]bool
	if json.Unmarshal(data, &m) != nil {
		return map[string]bool{}
	}
	return m
}

// SetRTKHarness records whether a harness has rtk wired and persists the map.
func (s *FS) SetRTKHarness(name string, on bool) error {
	m := s.RTKHarnesses()
	m[name] = on
	dir := filepath.Join(s.Root, ".claude", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, rtkHarnessFile), data, 0644)
}
