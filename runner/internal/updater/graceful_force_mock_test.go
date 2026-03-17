package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for ForceUpdate using MockReleaseDetector

func TestGracefulUpdater_ForceUpdate_WithMock_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-force-mock-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v2.0.0",
		},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)
	g := NewGracefulUpdater(u, nil)

	err = g.ForceUpdate(context.Background())
	assert.NoError(t, err)
}

func TestGracefulUpdater_ForceUpdate_WithMock_NoUpdateAvailable(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))
	g := NewGracefulUpdater(u, nil)

	err := g.ForceUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no update available")
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ForceUpdate_WithMock_CheckError(t *testing.T) {
	mock := &MockReleaseDetector{
		DetectError: errors.New("network error"),
	}

	u := New("1.0.0", WithReleaseDetector(mock))
	g := NewGracefulUpdater(u, nil)

	err := g.ForceUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ForceUpdate_WithMock_UpdateError(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v2.0.0",
		},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		UpdateError: errors.New("update binary failed"),
	}

	tmpDir, err := os.MkdirTemp("", "graceful-force-mock-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)
	g := NewGracefulUpdater(u, nil)

	err = g.ForceUpdate(context.Background())
	assert.Error(t, err)
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ForceUpdate_WithMock_WithPending(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-force-mock-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{}
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)
	g := NewGracefulUpdater(u, nil)

	// Set up pending update (from a previous ScheduleUpdate that was draining)
	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	// ForceUpdate should apply pending
	err = g.ForceUpdate(context.Background())
	assert.NoError(t, err)

	// Verify binary was replaced
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "mock binary", string(content))
}

func TestGracefulUpdater_ForceUpdate_WithMock_InvalidState(t *testing.T) {
	mock := &MockReleaseDetector{}
	u := New("1.0.0", WithReleaseDetector(mock))
	g := NewGracefulUpdater(u, nil)

	// Set state to StateDownloading
	g.mu.Lock()
	g.state = StateDownloading
	g.mu.Unlock()

	err := g.ForceUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot force update in state")
}

func TestGracefulUpdater_ForceUpdate_WithMock_StateTransitions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-force-mock-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v2.0.0",
		},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	var states []State
	cb := func(state State, info *UpdateInfo, activePods int) {
		states = append(states, state)
	}

	g := NewGracefulUpdater(u, nil, WithStatusCallback(cb))

	err = g.ForceUpdate(context.Background())
	assert.NoError(t, err)

	// Should have gone through these states
	assert.Contains(t, states, StateChecking)
	assert.Contains(t, states, StateApplying)
}
