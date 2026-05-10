// Package testsupport provides testing-only fixture helpers for the
// opencode agent. Separated from the production opencode library to avoid
// linking the testing package and fixture bytes into shipped runner
// binaries.
package testsupport

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/msg.json
var fixtureMsg []byte

// BuildFixtureSandbox plants the embedded message file under a temporary
// HOME at the path opencode parser globs.
func BuildFixtureSandbox(t *testing.T) string {
	t.Helper()
	sandbox := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	dir := filepath.Join(home, ".local", "share", "opencode", "storage", "message", "session_fixture")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("opencode fixture: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "msg_fixture.json"), fixtureMsg, 0o644); err != nil {
		t.Fatalf("opencode fixture: write: %v", err)
	}
	return sandbox
}
