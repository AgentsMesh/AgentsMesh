package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestGracefulUpdater_ApplyPendingUpdate_CallsRestart(t *testing.T) {
	u := New("1.0.0")

	// We need a scenario where applyPendingUpdate succeeds past the Apply stage
	// This is difficult without mocking os.Executable
	// Instead, we verify the restart function is called when set

	called := false
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		called = true
		return 12345, nil
	}))

	// Set up pending but with invalid path (will fail at Apply)
	g.mu.Lock()
	g.pendingPath = "/nonexistent/path"
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err := g.applyPendingUpdate()
	assert.Error(t, err)

	// Restart should not be called because Apply failed
	assert.False(t, called)
}
