//go:build !windows

package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteTestScript writes a shell script to the given directory and returns its path.
// The script is made executable (chmod 0755).
// body should contain the script logic without the shebang line — it will be added automatically.
func WriteTestScript(t *testing.T, dir, name, body string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	content := "#!/bin/sh\n" + body + "\n"

	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("WriteTestScript: failed to write %s: %v", path, err)
	}

	return path
}
