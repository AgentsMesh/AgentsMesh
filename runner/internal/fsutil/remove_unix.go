//go:build !windows

package fsutil

import "os"

// RemoveAll removes path and any children it contains.
// On Unix, this is a direct delegation to os.RemoveAll.
func RemoveAll(path string) error {
	return os.RemoveAll(path)
}
