//go:build windows

package testutil

// isRoot returns true if the current user has administrator privileges.
// On Windows, we conservatively return false — permission-denied tests
// may still pass under elevated shells, but skipping them is safer.
func isRoot() bool {
	return false
}
