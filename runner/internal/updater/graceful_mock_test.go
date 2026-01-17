package updater

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests using mock detector for GracefulUpdater - ScheduleUpdate

func TestGracefulUpdater_ScheduleUpdate_WithMock_NoUpdate(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))
	g := NewGracefulUpdater(u, nil)

	err := g.ScheduleUpdate(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ScheduleUpdate_WithMock_HasUpdate_NoPods(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-mock-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version:      "v2.0.0",
			ReleaseNotes: "New version",
		},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)
	podCounter := func() int { return 0 }
	g := NewGracefulUpdater(u, podCounter)

	err = g.ScheduleUpdate(context.Background())
	assert.NoError(t, err)
}

func TestGracefulUpdater_ScheduleUpdate_WithMock_DownloadFails(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v2.0.0",
		},
		VersionReleases: map[string]*ReleaseInfo{}, // Version not found
	}

	u := New("1.0.0", WithReleaseDetector(mock))
	g := NewGracefulUpdater(u, nil)

	err := g.ScheduleUpdate(context.Background())
	assert.Error(t, err)
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ScheduleUpdate_WithMock_WaitsForPods(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-mock-*")
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

	var podCount int32 = 2
	podCounter := func() int { return int(atomic.LoadInt32(&podCount)) }

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(200*time.Millisecond),
		WithPollInterval(20*time.Millisecond),
	)

	// Simulate pods finishing
	go func() {
		time.Sleep(50 * time.Millisecond)
		atomic.StoreInt32(&podCount, 0)
	}()

	err = g.ScheduleUpdate(context.Background())
	assert.NoError(t, err)
}

func TestGracefulUpdater_StatusCallback_DuringUpdate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-mock-*")
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

	g := NewGracefulUpdater(u, func() int { return 0 }, WithStatusCallback(cb))

	err = g.ScheduleUpdate(context.Background())
	assert.NoError(t, err)

	// Verify state transitions
	assert.Contains(t, states, StateChecking)
	assert.Contains(t, states, StateDownloading)
	assert.Contains(t, states, StateDraining)
	assert.Contains(t, states, StateApplying)
}
