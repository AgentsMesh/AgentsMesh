//go:build windows

package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WriteTestScript writes a batch script to the given directory and returns its path.
// The .bat extension is appended automatically if not already present.
// body should contain the script logic — @echo off is prepended automatically.
func WriteTestScript(t *testing.T, dir, name, body string) string {
	t.Helper()

	// Ensure .bat extension
	if !strings.HasSuffix(name, ".bat") && !strings.HasSuffix(name, ".cmd") {
		name = name + ".bat"
	}

	path := filepath.Join(dir, name)
	content := "@echo off\r\n" + body + "\r\n"

	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("WriteTestScript: failed to write %s: %v", path, err)
	}

	return path
}
