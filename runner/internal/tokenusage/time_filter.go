package tokenusage

import (
	"os"
	"time"
)

// IsModifiedAfter returns true if the file's modification time is at or after the given time.
func IsModifiedAfter(path string, after time.Time) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.ModTime().Before(after)
}
