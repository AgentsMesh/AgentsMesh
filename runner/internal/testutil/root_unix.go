//go:build !windows

package testutil

import "os"

// isRoot returns true if the current user is root (uid 0).
func isRoot() bool {
	return os.Getuid() == 0
}
