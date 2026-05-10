// Package testsupport provides testing-only fixture helpers for the aider
// agent. Separated from the production aider library to avoid linking the
// testing package and fixture bytes into shipped runner binaries.
package testsupport

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/session.md
var fixtureSession []byte

// BuildFixtureSandbox plants the embedded chat history at the location the
// aider parser scans (<sandbox>/workspace/.aider.chat.history.md).
func BuildFixtureSandbox(t *testing.T) string {
	t.Helper()
	sandbox := t.TempDir()
	workspace := filepath.Join(sandbox, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("aider fixture: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".aider.chat.history.md"), fixtureSession, 0o644); err != nil {
		t.Fatalf("aider fixture: write: %v", err)
	}
	return sandbox
}
