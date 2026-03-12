package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimulatedDetector mimics go-selfupdate's real UpdateTo behavior inside DownloadTo:
// 1. Write new content to ".target.new"
// 2. Rename target → ".target.old"      ← this is where the real bug was
// 3. Rename ".target.new" → target
// 4. Remove ".target.old"
//
// This catches issues that MockReleaseDetector (which just calls os.WriteFile) misses.
type SimulatedDetector struct {
	LatestRelease   *ReleaseInfo
	VersionReleases map[string]*ReleaseInfo
	BinaryContent   []byte // content to write as the "new binary"
	DetectError     error
	DownloadError   error
}

func (s *SimulatedDetector) DetectLatest(_ context.Context) (*ReleaseInfo, bool, error) {
	if s.DetectError != nil {
		return nil, false, s.DetectError
	}
	if s.LatestRelease == nil {
		return nil, false, nil
	}
	return s.LatestRelease, true, nil
}

func (s *SimulatedDetector) DetectVersion(_ context.Context, version string) (*ReleaseInfo, bool, error) {
	if s.DetectError != nil {
		return nil, false, s.DetectError
	}
	r, ok := s.VersionReleases[version]
	return r, ok, nil
}

// DownloadTo simulates the real go-selfupdate UpdateTo → update.Apply sequence:
//
//	new → ".target.new" → rename target → ".target.old" → rename ".target.new" → target
//
// This is the sequence that broke in production when target did not exist.
func (s *SimulatedDetector) DownloadTo(_ context.Context, _ *ReleaseInfo, path string) error {
	if s.DownloadError != nil {
		return s.DownloadError
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	newPath := filepath.Join(dir, "."+base+".new")
	oldPath := filepath.Join(dir, "."+base+".old")

	content := s.BinaryContent
	if content == nil {
		content = []byte("simulated binary")
	}

	// Step 1: write new content to .new file
	if err := os.WriteFile(newPath, content, 0755); err != nil {
		return fmt.Errorf("failed to create .new file: %w", err)
	}

	// Step 2: remove leftover .old (may not exist)
	os.Remove(oldPath)

	// Step 3: rename target → .old  (THIS IS THE CRITICAL STEP)
	if err := os.Rename(path, oldPath); err != nil {
		// Cleanup .new on failure
		os.Remove(newPath)
		return fmt.Errorf("rename %s → %s: %w", path, oldPath, err)
	}

	// Step 4: rename .new → target
	if err := os.Rename(newPath, path); err != nil {
		// Rollback: restore .old → target
		_ = os.Rename(oldPath, path)
		return fmt.Errorf("rename .new → target: %w", err)
	}

	// Step 5: remove .old
	os.Remove(oldPath)

	return nil
}

// TestE2E_Download_TargetFileDoesNotExist reproduces the exact production failure:
// go-selfupdate's UpdateTo internally renames the target file to .old, but
// when Download creates a fresh tmpDir the target doesn't exist yet.
// This test ensures our DownloadTo creates the necessary placeholder.
func TestE2E_Download_TargetFileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	// Write a fake "current binary" so Download can determine the exec dir
	err := os.WriteFile(execPath, []byte("old"), 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		BinaryContent: []byte("new binary v2"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	require.NoError(t, err, "Download should succeed even though target file does not pre-exist in tmpDir")
	defer os.RemoveAll(filepath.Dir(path))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new binary v2", string(content))
}

// TestE2E_UpdateNow_FullCycle tests the complete update cycle:
// check → download → apply → verify new binary at exec path.
func TestE2E_UpdateNow_FullCycle(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	err := os.WriteFile(execPath, []byte("old binary v1"), 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		LatestRelease: &ReleaseInfo{Version: "v2.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		BinaryContent: []byte("new binary v2"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	version, err := u.UpdateNow(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", version)

	// Verify the exec path now has the new content
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "new binary v2", string(content))
}

// TestE2E_UpdateToVersion_FullCycle tests updating to a specific version.
func TestE2E_UpdateToVersion_FullCycle(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	err := os.WriteFile(execPath, []byte("old binary v1"), 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v3.0.0": {Version: "v3.0.0"},
		},
		BinaryContent: []byte("new binary v3"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	err = u.UpdateToVersion(context.Background(), "3.0.0", nil)
	require.NoError(t, err)

	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "new binary v3", string(content))
}

// TestE2E_Download_TmpDirCleanedOnFailure ensures the temp directory is
// cleaned up when DownloadTo fails.
func TestE2E_Download_TmpDirCleanedOnFailure(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	err := os.WriteFile(execPath, []byte("old"), 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		DownloadError: fmt.Errorf("network timeout"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "network timeout")

	// No leftover runner-update-* dirs should remain
	matches, _ := filepath.Glob(filepath.Join(tmpDir, "runner-update-*"))
	assert.Empty(t, matches, "temp directory should be cleaned up on failure")
}

// TestE2E_Download_BinaryNameMatchesPlatform ensures the temp file is always
// named "agentsmesh-runner" (or .exe on Windows) so go-selfupdate's
// DecompressCommand can locate it inside the tar archive.
func TestE2E_Download_BinaryNameMatchesPlatform(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	err := os.WriteFile(execPath, []byte("old"), 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		BinaryContent: []byte("binary"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	require.NoError(t, err)
	defer os.RemoveAll(filepath.Dir(path))

	expectedName := "agentsmesh-runner"
	if runtime.GOOS == "windows" {
		expectedName += ".exe"
	}
	assert.Equal(t, expectedName, filepath.Base(path),
		"downloaded binary must be named %q for tar archive extraction", expectedName)
}

// TestE2E_Download_ExecPathError tests that Download returns an error when
// the exec path function fails.
func TestE2E_Download_ExecPathError(t *testing.T) {
	sim := &SimulatedDetector{
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return "", fmt.Errorf("no exec path") }),
	)

	path, err := u.Download(context.Background(), "v2.0.0", nil)
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "failed to get executable path")
}

// TestE2E_BackupAndRollback_FullCycle tests backup → update → rollback.
func TestE2E_BackupAndRollback_FullCycle(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	originalContent := []byte("original binary v1")
	err := os.WriteFile(execPath, originalContent, 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		LatestRelease: &ReleaseInfo{Version: "v2.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		BinaryContent: []byte("new binary v2"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	// Step 1: Create backup
	backupPath, err := u.CreateBackup()
	require.NoError(t, err)
	assert.Equal(t, execPath+".bak", backupPath)

	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, backupContent)

	// Step 2: Update
	version, err := u.UpdateNow(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", version)

	updatedContent, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "new binary v2", string(updatedContent))

	// Step 3: Rollback
	err = u.Rollback()
	require.NoError(t, err)

	rolledBackContent, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, rolledBackContent)
}

// TestE2E_Apply_CleansTmpDir verifies that UpdateNow cleans up the
// temporary download directory after a successful apply.
func TestE2E_Apply_CleansTmpDir(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	err := os.WriteFile(execPath, []byte("old"), 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		LatestRelease: &ReleaseInfo{Version: "v2.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		BinaryContent: []byte("new"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	_, err = u.UpdateNow(context.Background(), nil)
	require.NoError(t, err)

	// No leftover runner-update-* dirs should remain
	matches, _ := filepath.Glob(filepath.Join(tmpDir, "runner-update-*"))
	assert.Empty(t, matches, "temp directory should be cleaned up after successful update")
}

// TestE2E_VersionNormalization verifies v-prefix handling through the
// full update path (regression for #44).
func TestE2E_VersionNormalization(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	err := os.WriteFile(execPath, []byte("old"), 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		VersionReleases: map[string]*ReleaseInfo{
			// Stored with v-prefix as the real GitHub API returns
			"v1.2.3": {Version: "v1.2.3"},
		},
		BinaryContent: []byte("v1.2.3 binary"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	// Pass version WITHOUT v-prefix — normalizeVersion should add it
	err = u.UpdateToVersion(context.Background(), "1.2.3", nil)
	require.NoError(t, err)

	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3 binary", string(content))
}
