package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGitHubReleaseDetector(t *testing.T) {
	detector, err := NewGitHubReleaseDetector()
	assert.NoError(t, err)
	assert.NotNil(t, detector)
	assert.NotNil(t, detector.updater)
}

func TestReleaseInfo(t *testing.T) {
	info := &ReleaseInfo{
		Version:      "v1.0.0",
		ReleaseNotes: "Initial release",
		AssetURL:     "https://example.com/v1.0.0",
		AssetName:    "runner-v1.0.0.tar.gz",
	}

	assert.Equal(t, "v1.0.0", info.Version)
	assert.Equal(t, "Initial release", info.ReleaseNotes)
	assert.Equal(t, "https://example.com/v1.0.0", info.AssetURL)
	assert.Equal(t, "runner-v1.0.0.tar.gz", info.AssetName)
}
