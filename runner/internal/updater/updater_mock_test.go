package updater

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for Updater using MockReleaseDetector

func TestUpdater_CheckForUpdate_WithMock_HasUpdate(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version:      "v2.0.0",
			ReleaseNotes: "New features",
			PublishedAt:  time.Now(),
			AssetURL:     "https://example.com/v2.0.0",
			AssetName:    "runner-v2.0.0.tar.gz",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	info, err := u.CheckForUpdate(context.Background())
	assert.NoError(t, err)
	assert.True(t, info.HasUpdate)
	assert.Equal(t, "v2.0.0", info.LatestVersion)
	assert.Equal(t, "New features", info.ReleaseNotes)
}

func TestUpdater_CheckForUpdate_WithMock_NoUpdate(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("2.0.0", WithReleaseDetector(mock))

	info, err := u.CheckForUpdate(context.Background())
	assert.NoError(t, err)
	assert.False(t, info.HasUpdate)
}

func TestUpdater_CheckForUpdate_WithMock_NotFound(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: nil,
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	info, err := u.CheckForUpdate(context.Background())
	assert.NoError(t, err)
	assert.False(t, info.HasUpdate)
}

func TestUpdater_CheckForUpdate_WithMock_Error(t *testing.T) {
	mock := &MockReleaseDetector{
		DetectError: errors.New("network error"),
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	info, err := u.CheckForUpdate(context.Background())
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "network error")
}

func TestUpdater_CheckForUpdate_WithMock_Prerelease(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v2.0.0-beta.1",
		},
	}

	t.Run("prerelease not allowed", func(t *testing.T) {
		u := New("1.0.0", WithReleaseDetector(mock), WithPrerelease(false))
		info, err := u.CheckForUpdate(context.Background())
		assert.NoError(t, err)
		assert.False(t, info.HasUpdate)
	})

	t.Run("prerelease allowed", func(t *testing.T) {
		u := New("1.0.0", WithReleaseDetector(mock), WithPrerelease(true))
		info, err := u.CheckForUpdate(context.Background())
		assert.NoError(t, err)
		assert.True(t, info.HasUpdate)
	})
}

func TestUpdater_CheckForUpdate_WithMock_InvalidLatestVersion(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "invalid-version",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	info, err := u.CheckForUpdate(context.Background())
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "failed to parse latest version")
}

func TestUpdater_CheckForUpdate_WithMock_DevVersion(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v2.0.0",
		},
	}

	// Using "dev" as current version which cannot be parsed
	u := New("dev", WithReleaseDetector(mock))

	info, err := u.CheckForUpdate(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, info)
	// "dev" is treated as v0.0.0, so v2.0.0 should be greater
	assert.True(t, info.HasUpdate)
}
