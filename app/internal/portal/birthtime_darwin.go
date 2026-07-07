package portal

import (
	"os"
	"syscall"
)

// birthTime returns the file creation time (macOS exposes birthtime).
func birthTime(fi os.FileInfo) int64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return st.Birthtimespec.Sec
	}
	return 0
}
