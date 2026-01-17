package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for BackgroundChecker.doCheck using MockReleaseDetector

func TestBackgroundChecker_DoCheck_WithMock_HasUpdate(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version:      "v2.0.0",
			ReleaseNotes: "New features",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	var updateCalled atomic.Bool
	var receivedInfo *UpdateInfo

	c := NewBackgroundChecker(u, nil, time.Hour,
		WithOnUpdate(func(info *UpdateInfo) {
			updateCalled.Store(true)
			receivedInfo = info
		}),
	)

	ctx := context.Background()
	info, err := c.CheckNow(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.True(t, info.HasUpdate)
	assert.True(t, updateCalled.Load())
	assert.Equal(t, "v2.0.0", receivedInfo.LatestVersion)
}

func TestBackgroundChecker_DoCheck_WithMock_NoUpdate(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	var updateCalled atomic.Bool
	c := NewBackgroundChecker(u, nil, time.Hour,
		WithOnUpdate(func(info *UpdateInfo) {
			updateCalled.Store(true)
		}),
	)

	ctx := context.Background()
	info, err := c.CheckNow(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.False(t, info.HasUpdate)
	assert.False(t, updateCalled.Load())
}

func TestBackgroundChecker_DoCheck_WithMock_Error(t *testing.T) {
	mock := &MockReleaseDetector{
		DetectError: errors.New("network error"),
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	var errorCalled atomic.Bool
	c := NewBackgroundChecker(u, nil, time.Hour,
		WithOnError(func(err error) {
			errorCalled.Store(true)
		}),
	)

	ctx := context.Background()
	info, err := c.CheckNow(ctx)

	assert.Error(t, err)
	assert.Nil(t, info)
	assert.True(t, errorCalled.Load())
}

func TestBackgroundChecker_DoCheck_WithMock_AutoApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker-mock-*")
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
	g := NewGracefulUpdater(u, func() int { return 0 })

	c := NewBackgroundChecker(u, g, time.Hour, WithAutoApply(true))

	ctx := context.Background()
	info, err := c.CheckNow(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.True(t, info.HasUpdate)

	// Wait for auto-apply goroutine
	time.Sleep(200 * time.Millisecond)
}
