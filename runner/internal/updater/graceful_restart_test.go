package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for DefaultRestartFunc

func TestDefaultRestartFunc_Returns(t *testing.T) {
	// DefaultRestartFunc returns a function
	restartFunc := DefaultRestartFunc()
	assert.NotNil(t, restartFunc)

	// Note: We cannot actually call restartFunc() in tests as it would
	// start a new process. We can only verify it returns a function.
}

func TestGracefulUpdater_WithRestartFunc_Custom(t *testing.T) {
	u := New("1.0.0")

	called := false
	customRestart := func() (int, error) {
		called = true
		return 12345, nil
	}

	g := NewGracefulUpdater(u, nil, WithRestartFunc(customRestart))
	assert.NotNil(t, g.restartFunc)

	pid, err := g.restartFunc()
	assert.NoError(t, err)
	assert.Equal(t, 12345, pid)
	assert.True(t, called)
}

func TestGracefulUpdater_WithRestartFunc_Nil(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	// Default has no restart function
	assert.Nil(t, g.restartFunc)
}

func TestGracefulUpdater_ApplyUpdate_RestartNotCalledOnFailure(t *testing.T) {
	// When updateBinary fails, restart should not be called
	mock := &MockReleaseDetector{
		UpdateError: fmt.Errorf("update failed"),
	}

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	called := false
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		called = true
		return 12345, nil
	}))

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.Error(t, err)

	// Restart should not be called because updateBinary failed
	assert.False(t, called)
}
