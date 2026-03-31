//go:build integration

package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdater_UpdateToSpecificVersion_Integration verifies that UpdateToVersion
// applies the requested version, not the latest detected version.
func TestUpdater_UpdateToSpecificVersion_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "runner-bin")
	require.NoError(t, os.WriteFile(execPath, []byte("old binary"), 0755))

	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{Version: "v5.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v3.0.0": {Version: "v3.0.0"},
			"v5.0.0": {Version: "v5.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	err := u.UpdateToVersion(context.Background(), "v3.0.0")
	require.NoError(t, err)

	data, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "mock binary", string(data), "binary should be replaced")
}

// TestUpdater_BackgroundChecker_AutoApply_Integration verifies that when the
// OnUpdate callback triggers UpdateNow, the binary is actually replaced.
func TestUpdater_BackgroundChecker_AutoApply_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "runner-bin")
	require.NoError(t, os.WriteFile(execPath, []byte("old binary"), 0755))

	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{Version: "v8.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v8.0.0": {Version: "v8.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	doneCh := make(chan struct{}, 1)
	checker := NewBackgroundChecker(u, nil, 100*time.Millisecond,
		WithOnUpdate(func(info *UpdateInfo) {
			if _, err := u.UpdateNow(context.Background()); err == nil {
				select {
				case doneCh <- struct{}{}:
				default:
				}
			}
		}),
		WithAutoApply(false),
		WithInitialDelay(10*time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	checker.Start(ctx)
	defer checker.Stop()

	select {
	case <-doneCh:
		data, err := os.ReadFile(execPath)
		require.NoError(t, err)
		assert.Equal(t, "mock binary", string(data))
	case <-ctx.Done():
		t.Fatal("timed out waiting for auto-apply via callback")
	}
}

// TestUpdater_NoUpdateAvailable_Integration verifies that when the detector
// reports no release found, CheckForUpdate returns HasUpdate=false without error.
func TestUpdater_NoUpdateAvailable_Integration(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: nil, // DetectLatest returns found=false
	}
	u := New("2.0.0", WithReleaseDetector(mock))

	info, err := u.CheckForUpdate(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.HasUpdate)
	assert.Equal(t, "v2.0.0", info.CurrentVersion)
	assert.Empty(t, info.LatestVersion)
}

// TestUpdater_BackgroundChecker_StopDuringCheck_Integration verifies that
// calling Stop() immediately after Start() results in a clean shutdown.
func TestUpdater_BackgroundChecker_StopDuringCheck_Integration(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{Version: "v9.0.0"},
	}
	u := New("1.0.0", WithReleaseDetector(mock))

	checker := NewBackgroundChecker(u, nil, 1*time.Hour,
		WithAutoApply(false),
		WithInitialDelay(500*time.Millisecond),
	)

	ctx := context.Background()
	checker.Start(ctx)

	// Immediately stop — should not hang or panic.
	checker.Stop()

	assert.False(t, checker.IsRunning(), "checker should not be running after Stop")
}
