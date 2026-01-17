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

// Tests for UpdateToVersion using mock

func TestUpdater_UpdateToVersion_WithMock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "update-version-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	err = u.UpdateToVersion(context.Background(), "2.0.0", nil)
	assert.NoError(t, err)

	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "mock binary", string(content))
}

func TestUpdater_UpdateToVersion_ApplyError(t *testing.T) {
	mock := &MockReleaseDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return "", errors.New("path error") }),
	)

	err := u.UpdateToVersion(context.Background(), "2.0.0", nil)
	assert.Error(t, err)
}

func TestUpdater_UpdateNow_ApplyError(t *testing.T) {
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
		WithExecPathFunc(func() (string, error) { return "", errors.New("path error") }),
	)

	version, err := u.UpdateNow(context.Background(), nil)
	assert.Error(t, err)
	assert.Empty(t, version)
}

func TestUpdater_UpdateNow_WithMock_DownloadError(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v2.0.0",
		},
		VersionReleases: map[string]*ReleaseInfo{},
		// Version not in VersionReleases -> download fails
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	version, err := u.UpdateNow(context.Background(), nil)
	assert.Error(t, err)
	assert.Empty(t, version)
}

func TestUpdater_UpdateNow_WithMock_CheckError(t *testing.T) {
	mock := &MockReleaseDetector{
		DetectError: errors.New("check failed"),
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	version, err := u.UpdateNow(context.Background(), nil)
	assert.Error(t, err)
	assert.Empty(t, version)
}
