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

// Tests for UpdateNow and UpdateToVersion using MockReleaseDetector

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

	version, err := u.UpdateNow(context.Background())
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

	version, err := u.UpdateNow(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, version)
}

func TestUpdater_UpdateToVersion_WithMock_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
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

	err = u.UpdateToVersion(context.Background(), "v2.0.0")
	assert.NoError(t, err)

	content, err := os.ReadFile(execPath)
	assert.NoError(t, err)
	assert.Equal(t, "mock binary", string(content))
}

func TestUpdater_UpdateBinary_WithMock_UpdateError(t *testing.T) {
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
		UpdateError: errors.New("update failed"),
	}

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	version, err := u.UpdateNow(context.Background())
	assert.Error(t, err)
	assert.Empty(t, version)
	assert.Contains(t, err.Error(), "update failed")
}

func TestUpdater_UpdateBinary_WithMock_DetectError(t *testing.T) {
	mock := &MockReleaseDetector{
		DetectError: errors.New("version detect error"),
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	version, err := u.UpdateNow(context.Background())
	assert.Error(t, err)
	assert.Empty(t, version)
	assert.Contains(t, err.Error(), "detect")
}

func TestUpdater_UpdateBinary_WithMock_ExecPathError(t *testing.T) {
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
		WithExecPathFunc(func() (string, error) { return "", errors.New("no exec path") }),
	)

	version, err := u.UpdateNow(context.Background())
	assert.Error(t, err)
	assert.Empty(t, version)
	assert.Contains(t, err.Error(), "failed to get executable path")
}
