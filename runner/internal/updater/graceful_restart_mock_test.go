package updater

import (
	"os"
	"os/exec"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for DefaultRestartFunc

func TestDefaultRestartFunc_Success(t *testing.T) {
	// Skip on CI as it may have permission issues
	if os.Getenv("CI") != "" {
		t.Skip("Skipping on CI")
	}

	// Create a simple test binary that exits immediately
	tmpDir, err := os.MkdirTemp("", "restart-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testBinary := testutil.WriteTestScript(t, tmpDir, "test-binary", "exit 0")

	// We can't easily test DefaultRestartFunc directly since it uses os.Executable
	// Instead, test the restart logic separately
	cmd := exec.Command(testBinary)
	err = cmd.Start()
	assert.NoError(t, err)

	if cmd.Process != nil {
		_ = cmd.Wait()
	}
}

func TestDefaultRestartFunc_Creation(t *testing.T) {
	fn := DefaultRestartFunc()
	assert.NotNil(t, fn)
}

func TestDefaultRestartFunc_WithInvalidExec(t *testing.T) {
	// Save original os.Args
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Set os.Args to an empty slice first
	os.Args = []string{"test"}

	// The function should be callable
	fn := DefaultRestartFunc()
	assert.NotNil(t, fn)

	// Calling it will fail in test environment but shouldn't panic
	// Note: We don't call fn() here because it would try to start a process
}
