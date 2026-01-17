package updater

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty version", "", "v0.0.0-dev"},
		{"dev version", "dev", "v0.0.0-dev"},
		{"version without v prefix", "1.2.3", "v1.2.3"},
		{"version with v prefix", "v1.2.3", "v1.2.3"},
		{"version with whitespace", "  v1.2.3  ", "v1.2.3"},
		{"prerelease version", "1.2.3-beta.1", "v1.2.3-beta.1"},
		{"only v", "v", "v"},
		{"v with number", "v1", "v1"},
		{"alpha version", "v1.0.0-alpha", "v1.0.0-alpha"},
		{"build metadata", "v1.0.0+build123", "v1.0.0+build123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdater_New(t *testing.T) {
	t.Run("creates updater with default options", func(t *testing.T) {
		u := New("1.0.0")
		assert.NotNil(t, u)
		assert.Equal(t, "v1.0.0", u.currentVersion)
		assert.False(t, u.allowPrerelease)
	})

	t.Run("creates updater with prerelease option", func(t *testing.T) {
		u := New("1.0.0", WithPrerelease(true))
		assert.NotNil(t, u)
		assert.True(t, u.allowPrerelease)
	})
}

func TestUpdater_CurrentVersion(t *testing.T) {
	u := New("v2.1.0")
	assert.Equal(t, "v2.1.0", u.CurrentVersion())
}

func TestRepoConstants(t *testing.T) {
	assert.Equal(t, "AgentsMesh", RepoOwner)
	assert.Equal(t, "AgentsMeshRunner", RepoName)
}

func TestUpdater_GetSource(t *testing.T) {
	u := New("1.0.0")

	source, err := u.getSource()
	assert.NoError(t, err)
	assert.NotNil(t, source)
}

func TestUpdater_GetDetector(t *testing.T) {
	u := New("1.0.0")

	detector, err := u.getDetector()
	assert.NoError(t, err)
	assert.NotNil(t, detector)

	// Same detector should be returned on subsequent calls
	detector2, err := u.getDetector()
	assert.NoError(t, err)
	assert.Same(t, detector, detector2)
}

func TestUpdater_GetUpdater(t *testing.T) {
	u := New("1.0.0")

	updater, err := u.getUpdater()
	assert.NoError(t, err)
	assert.NotNil(t, updater)
}

func TestWithPrerelease(t *testing.T) {
	u := New("1.0.0", WithPrerelease(true))
	assert.True(t, u.allowPrerelease)

	u = New("1.0.0", WithPrerelease(false))
	assert.False(t, u.allowPrerelease)
}

func TestUpdateInfo(t *testing.T) {
	info := &UpdateInfo{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v2.0.0",
		ReleaseNotes:   "New features",
		PublishedAt:    time.Now(),
		HasUpdate:      true,
		AssetURL:       "https://example.com/release.tar.gz",
		AssetName:      "runner-linux-amd64.tar.gz",
	}

	assert.Equal(t, "v1.0.0", info.CurrentVersion)
	assert.Equal(t, "v2.0.0", info.LatestVersion)
	assert.Equal(t, "New features", info.ReleaseNotes)
	assert.True(t, info.HasUpdate)
}

func TestUpdater_WithInjectedDetector(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{Version: "v2.0.0"},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	detector, err := u.getDetector()
	assert.NoError(t, err)
	assert.Same(t, mock, detector)
}

func TestUpdater_Download_ContextCanceled(t *testing.T) {
	u := New("1.0.0")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := u.Download(ctx, "v1.0.0", nil)
	assert.Error(t, err)
}

func TestUpdater_CheckForUpdate_InvalidVersion(t *testing.T) {
	u := New("invalid-version")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = u.CheckForUpdate(ctx)
	// Just verify no panic
}

func TestUpdater_CheckForUpdate_WithMockSource(t *testing.T) {
	u := New("1.0.0")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = u.CheckForUpdate(ctx)
	// Just verify no panic
}

func TestUpdater_UpdateNow_NoUpdate(t *testing.T) {
	u := New("999.0.0")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := u.UpdateNow(ctx, nil)
	if err == nil {
		assert.Empty(t, version)
	}
}

func TestUpdater_UpdateToVersion(t *testing.T) {
	u := New("1.0.0")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := u.UpdateToVersion(ctx, "v0.0.1", nil)
	assert.Error(t, err)
}
