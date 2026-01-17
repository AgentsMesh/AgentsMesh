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

// Tests for Download and UpdateNow using MockReleaseDetector

func TestUpdater_Download_WithMock_Success(t *testing.T) {
	mock := &MockReleaseDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
	defer os.Remove(path)

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "mock binary", string(content))
}

func TestUpdater_Download_WithMock_VersionNotFound(t *testing.T) {
	mock := &MockReleaseDetector{
		VersionReleases: map[string]*ReleaseInfo{},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdater_Download_WithMock_DownloadError(t *testing.T) {
	mock := &MockReleaseDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		DownloadError: errors.New("download failed"),
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "download failed")
}

func TestUpdater_UpdateNow_WithMock_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
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

	version, err := u.UpdateNow(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, "v2.0.0", version)

	content, err := os.ReadFile(execPath)
	assert.NoError(t, err)
	assert.Equal(t, "mock binary", string(content))
}

func TestUpdater_UpdateNow_WithMock_NoUpdate(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	version, err := u.UpdateNow(context.Background(), nil)
	assert.NoError(t, err)
	assert.Empty(t, version)
}

func TestUpdater_Download_WithMock_DownloadToError(t *testing.T) {
	mock := &MockReleaseDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		DownloadError: errors.New("download failed"),
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "failed to download update")
}

func TestUpdater_Download_WithMock_DetectVersionError(t *testing.T) {
	mock := &MockReleaseDetector{
		DetectError: errors.New("version detect error"),
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "failed to find version")
}
