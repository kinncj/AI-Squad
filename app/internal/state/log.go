package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// logFields are the skills.jsonl keys rendered, in order, for each log line.
var logFields = []string{"ts", "agent", "skill", "op", "file", "duration", "error"}

// LogLines returns the last n formatted lines of .claude/logs/skills.jsonl. Each
// JSON record is flattened to "k=v  k=v"; non-JSON lines pass through verbatim.
// Empty when the log is absent.
func (s *FS) LogLines(n int) []string {
	data, err := os.ReadFile(filepath.Join(s.Root, ".claude", "logs", "skills.jsonl"))
	if err != nil {
		return nil
	}
	return formatLogLines(string(data), n)
}

func formatLogLines(data string, n int) []string {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(l), &m) != nil {
			out = append(out, l)
			continue
		}
		var parts []string
		for _, k := range logFields {
			if v, ok := m[k]; ok && v != nil && v != "" {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
		}
		out = append(out, strings.Join(parts, "  "))
	}
	return out
}
