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

// SimulatedDetector mimics go-selfupdate's real UpdateTo behavior inside UpdateBinary:
// 1. Write new content to ".target.new"
// 2. Rename target → ".target.old"
// 3. Rename ".target.new" → target
// 4. Remove ".target.old"
//
// This catches issues that MockReleaseDetector (which just calls os.WriteFile) misses.
type SimulatedDetector struct {
	LatestRelease   *ReleaseInfo
	VersionReleases map[string]*ReleaseInfo
	BinaryContent   []byte // content to write as the "new binary"
	DetectError     error
	UpdateError     error
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

// UpdateBinary simulates the real go-selfupdate UpdateTo → update.Apply sequence:
//
//	new → ".target.new" → rename target → ".target.old" → rename ".target.new" → target
//
// Since we now operate on the real exec path (which exists), the rename succeeds.
func (s *SimulatedDetector) UpdateBinary(_ context.Context, _ *ReleaseInfo, execPath string) error {
	if s.UpdateError != nil {
		return s.UpdateError
	}

	dir := filepath.Dir(execPath)
	base := filepath.Base(execPath)
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

	// Step 3: rename target → .old  (works because execPath exists)
	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("rename %s → %s: %w", execPath, oldPath, err)
	}

	// Step 4: rename .new → target
	if err := os.Rename(newPath, execPath); err != nil {
		_ = os.Rename(oldPath, execPath)
		return fmt.Errorf("rename .new → target: %w", err)
	}

	// Step 5: remove .old
	os.Remove(oldPath)

	return nil
}

// TestE2E_UpdateNow_FullCycle tests the complete update cycle:
// check → update binary in-place → verify new binary at exec path.
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

	version, err := u.UpdateNow(context.Background())
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

	err = u.UpdateToVersion(context.Background(), "3.0.0")
	require.NoError(t, err)

	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "new binary v3", string(content))
}

// TestE2E_UpdateBinary_Error ensures errors from UpdateBinary are propagated.
func TestE2E_UpdateBinary_Error(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	err := os.WriteFile(execPath, []byte("old"), 0755)
	require.NoError(t, err)

	sim := &SimulatedDetector{
		LatestRelease: &ReleaseInfo{Version: "v2.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
		UpdateError: fmt.Errorf("network timeout"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	version, err := u.UpdateNow(context.Background())
	assert.Error(t, err)
	assert.Empty(t, version)
	assert.Contains(t, err.Error(), "network timeout")

	// Original binary should be unchanged
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "old", string(content))
}

// TestE2E_UpdateBinary_ExecPathError tests that UpdateNow returns an error when
// the exec path function fails.
func TestE2E_UpdateBinary_ExecPathError(t *testing.T) {
	sim := &SimulatedDetector{
		LatestRelease: &ReleaseInfo{Version: "v2.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
		},
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return "", fmt.Errorf("no exec path") }),
	)

	version, err := u.UpdateNow(context.Background())
	assert.Error(t, err)
	assert.Empty(t, version)
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
	version, err := u.UpdateNow(context.Background())
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

// TestE2E_ConsecutiveUpgrades verifies that a second upgrade succeeds even
// after the first upgrade deleted the .old file. This is a regression test for
// the bug where go-selfupdate deletes .old after a successful swap, causing
// /proc/self/exe to resolve to a deleted path. With a pinned execPathFunc the
// Updater always operates on the canonical binary path, not the stale one.
func TestE2E_ConsecutiveUpgrades(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "agentsmesh-runner")
	err := os.WriteFile(execPath, []byte("v1 binary"), 0755)
	require.NoError(t, err)

	// Pin the exec path — simulates resolving it at startup before any upgrade.
	pinnedExecPath := execPath

	sim := &SimulatedDetector{
		LatestRelease: &ReleaseInfo{Version: "v2.0.0"},
		VersionReleases: map[string]*ReleaseInfo{
			"v2.0.0": {Version: "v2.0.0"},
			"v3.0.0": {Version: "v3.0.0"},
		},
		BinaryContent: []byte("v2 binary"),
	}

	u := New("1.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return pinnedExecPath, nil }),
	)

	// First upgrade: v1 → v2
	version, err := u.UpdateNow(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", version)

	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "v2 binary", string(content))

	// Simulate what happens in the real runner: .old is deleted by go-selfupdate,
	// /proc/self/exe would now return a stale path. But because we pinned
	// execPathFunc at startup, the Updater still uses the canonical path.

	// Second upgrade: v2 → v3
	sim.LatestRelease = &ReleaseInfo{Version: "v3.0.0"}
	sim.BinaryContent = []byte("v3 binary")
	u2 := New("2.0.0",
		WithReleaseDetector(sim),
		WithExecPathFunc(func() (string, error) { return pinnedExecPath, nil }),
	)

	version, err = u2.UpdateNow(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v3.0.0", version)

	content, err = os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "v3 binary", string(content))
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
	err = u.UpdateToVersion(context.Background(), "1.2.3")
	require.NoError(t, err)

	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3 binary", string(content))
}
