package state

import (
	"os"
	"strings"
)

// FileLines reads path and returns its lines (without trailing newline). Returns a
// single "(cannot read …)" line on error so detail overlays always show something.
func FileLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{"(cannot read " + path + ")"}
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}
