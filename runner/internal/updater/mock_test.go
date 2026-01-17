package updater

import (
	"context"
	"os"
)

// MockReleaseDetector implements ReleaseDetector for testing.
type MockReleaseDetector struct {
	LatestRelease   *ReleaseInfo
	VersionReleases map[string]*ReleaseInfo
	DownloadError   error
	DetectError     error
}

func (m *MockReleaseDetector) DetectLatest(ctx context.Context) (*ReleaseInfo, bool, error) {
	if m.DetectError != nil {
		return nil, false, m.DetectError
	}
	if m.LatestRelease == nil {
		return nil, false, nil
	}
	return m.LatestRelease, true, nil
}

func (m *MockReleaseDetector) DetectVersion(ctx context.Context, version string) (*ReleaseInfo, bool, error) {
	if m.DetectError != nil {
		return nil, false, m.DetectError
	}
	if m.VersionReleases == nil {
		return nil, false, nil
	}
	release, ok := m.VersionReleases[version]
	return release, ok, nil
}

func (m *MockReleaseDetector) DownloadTo(ctx context.Context, release *ReleaseInfo, path string) error {
	if m.DownloadError != nil {
		return m.DownloadError
	}
	// Write a dummy file
	return os.WriteFile(path, []byte("mock binary"), 0755)
}
