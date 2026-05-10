// Package testsupport provides testing-only fixture helpers for the claude
// agent. Separated from the production claude library to avoid linking the
// testing package and fixture bytes into shipped runner binaries.
package testsupport

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/agents/claude"
)

//go:embed testdata/session.jsonl
var fixtureSession []byte

// BuildFixtureSandbox plants the embedded fixture under a temporary HOME
// using claude.ProjectDirName so claudeParser resolves into it.
func BuildFixtureSandbox(t *testing.T) string {
	t.Helper()
	sandbox := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	resolved, err := filepath.EvalSymlinks(sandbox)
	if err != nil {
		t.Fatalf("claude fixture: eval symlinks: %v", err)
	}
	hash := claude.ProjectDirName(resolved)
	dir := filepath.Join(home, ".claude", "projects", hash)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("claude fixture: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), fixtureSession, 0o644); err != nil {
		t.Fatalf("claude fixture: write: %v", err)
	}
	return sandbox
}
