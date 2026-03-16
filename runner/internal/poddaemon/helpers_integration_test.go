//go:build integration

package poddaemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// findModuleRoot walks up from the current directory to find go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root (go.mod)")
		}
		dir = parent
	}
}

// buildTestRunner compiles the runner binary into a temp directory.
// Returns the path to the compiled binary.
func buildTestRunner(t *testing.T) string {
	t.Helper()

	binDir := t.TempDir()
	binName := "runner-test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(binDir, binName)

	modRoot := findModuleRoot(t)
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/runner")
	cmd.Dir = modRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	require.NoError(t, cmd.Run(), "failed to build runner binary")
	return binPath
}

// shortWorkspace creates a short temp dir to avoid macOS 104-byte
// Unix socket path limit. On Windows, t.TempDir() is fine.
// Returns (workspace, sandbox) paths.
func shortWorkspace(t *testing.T, name string) (string, string) {
	t.Helper()

	var workspace string
	if runtime.GOOS == "darwin" {
		// macOS has a 104-byte limit for Unix socket paths.
		var err error
		workspace, err = os.MkdirTemp("/tmp", "pd-")
		require.NoError(t, err)
		t.Cleanup(func() { os.RemoveAll(workspace) })
	} else {
		workspace = t.TempDir()
	}

	sandbox := filepath.Join(workspace, name)
	require.NoError(t, os.MkdirAll(sandbox, 0755))
	return workspace, sandbox
}
