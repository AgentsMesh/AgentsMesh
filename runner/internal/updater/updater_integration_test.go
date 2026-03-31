//go:build integration

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

func TestUpdater_CheckForUpdate_NewVersionAvailable_Integration(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version:      "v99.0.0",
			ReleaseNotes: "Major release",
			AssetURL:     "https://example.com/v99.0.0.tar.gz",
			AssetName:    "runner_v99.0.0.tar.gz",
		},
	}
	u := New("1.0.0", WithReleaseDetector(mock))

	info, err := u.CheckForUpdate(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.True(t, info.HasUpdate)
	assert.Equal(t, "v99.0.0", info.LatestVersion)
	assert.Equal(t, "v1.0.0", info.CurrentVersion)
	assert.Equal(t, "Major release", info.ReleaseNotes)
	assert.Equal(t, "https://example.com/v99.0.0.tar.gz", info.AssetURL)
}

func TestUpdater_CheckForUpdate_AlreadyLatest_Integration(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
	}{
		{"same version", "2.0.0", "v2.0.0"},
		{"older remote", "3.0.0", "v2.0.0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockReleaseDetector{
				LatestRelease: &ReleaseInfo{Version: tc.latestVersion},
			}
			u := New(tc.currentVersion, WithReleaseDetector(mock))

			info, err := u.CheckForUpdate(context.Background())
			require.NoError(t, err)
			require.NotNil(t, info)

			assert.False(t, info.HasUpdate)
		})
	}
}

func TestUpdater_BackgroundChecker_DetectsUpdate_Integration(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version:      "v5.0.0",
			ReleaseNotes: "Background update",
		},
	}
	u := New("1.0.0", WithReleaseDetector(mock))

	updateCh := make(chan *UpdateInfo, 1)
	checker := NewBackgroundChecker(u, nil, 100*time.Millisecond,
		WithOnUpdate(func(info *UpdateInfo) {
			select {
			case updateCh <- info:
			default:
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
	case info := <-updateCh:
		assert.True(t, info.HasUpdate)
		assert.Equal(t, "v5.0.0", info.LatestVersion)
		assert.Equal(t, "Background update", info.ReleaseNotes)
	case <-ctx.Done():
		t.Fatal("timed out waiting for update callback")
	}
}

func TestUpdater_BackgroundChecker_ErrorCallback_Integration(t *testing.T) {
	expectedErr := errors.New("simulated network failure")
	mock := &MockReleaseDetector{DetectError: expectedErr}
	u := New("1.0.0", WithReleaseDetector(mock))

	var receivedErr atomic.Value
	errCh := make(chan struct{}, 1)

	checker := NewBackgroundChecker(u, nil, 100*time.Millisecond,
		WithOnError(func(err error) {
			receivedErr.Store(err)
			select {
			case errCh <- struct{}{}:
			default:
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
	case <-errCh:
		stored := receivedErr.Load()
		require.NotNil(t, stored)
		assert.ErrorContains(t, stored.(error), "simulated network failure")
	case <-ctx.Done():
		t.Fatal("timed out waiting for error callback")
	}
}

func TestUpdater_UpdateNow_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "runner-bin")
	require.NoError(t, os.WriteFile(execPath, []byte("old binary"), 0755))

	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{Version: "v10.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v10.0.0": {Version: "v10.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	newVersion, err := u.UpdateNow(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v10.0.0", newVersion)

	// Verify the binary was replaced by MockReleaseDetector.UpdateBinary
	data, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "mock binary", string(data))
}
