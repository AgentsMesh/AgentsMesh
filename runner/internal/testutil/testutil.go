// Package testutil provides cross-platform test helpers.
// This package is only used by test code — it should never be imported by production code.
package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// SkipIfNoChmodSupport skips the test on platforms where chmod has no meaningful effect
// (e.g. Windows, which does not support Unix-style file permission bits).
func SkipIfNoChmodSupport(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("chmod is not supported on Windows")
	}
}

// SkipIfRoot skips the test when running as root/administrator,
// because permission-denied tests behave differently under elevated privileges.
func SkipIfRoot(t *testing.T) {
	t.Helper()
	if isRoot() {
		t.Skip("test requires non-root user")
	}
}

// InvalidDirPath returns a path guaranteed to be invalid for file creation.
//
//   - Unix:    /dev/null/x (cannot create children under /dev/null)
//   - Windows: NUL\x       (NUL is a reserved device name)
func InvalidDirPath() string {
	if runtime.GOOS == "windows" {
		return `NUL\x`
	}
	return "/dev/null/x"
}

// PythonCommand returns the platform-appropriate Python 3 command name.
func PythonCommand() string {
	if runtime.GOOS == "windows" {
		return "python"
	}
	return "python3"
}

// ExeSuffix returns the executable file extension for the current OS.
func ExeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// TempWorkDir creates a temporary directory suitable for use as a working directory in tests.
// It returns the directory path. The directory is automatically cleaned up when the test finishes.
func TempWorkDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Resolve symlinks — macOS /tmp is a symlink to /private/tmp
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return dir
	}
	return resolved
}

// TempDir returns os.TempDir() resolved through symlinks.
func TempDir() string {
	dir := os.TempDir()
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		return resolved
	}
	return dir
}
