package updater

import (
	"context"
	"os"
)

// MockReleaseDetector implements ReleaseDetector for testing.
type MockReleaseDetector struct {
	LatestRelease   *ReleaseInfo
	VersionReleases map[string]*ReleaseInfo
	UpdateError     error
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

func (m *MockReleaseDetector) UpdateBinary(ctx context.Context, release *ReleaseInfo, execPath string) error {
	if m.UpdateError != nil {
		return m.UpdateError
	}
	// Write a dummy file directly to the exec path
	return os.WriteFile(execPath, []byte("mock binary"), 0755)
}
