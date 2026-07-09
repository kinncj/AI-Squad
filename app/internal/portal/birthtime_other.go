//go:build !darwin

package portal

import "os"

// birthTime falls back to 0 (callers use mtime) where the OS lacks a birthtime.
func birthTime(_ os.FileInfo) int64 { return 0 }
