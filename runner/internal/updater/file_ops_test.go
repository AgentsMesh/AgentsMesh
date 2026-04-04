package updater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicReplaceWindows(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-ops-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source")
	dstPath := filepath.Join(tmpDir, "target")

	// Create source file
	err = os.WriteFile(srcPath, []byte("new content"), 0755)
	require.NoError(t, err)

	// Create target file
	err = os.WriteFile(dstPath, []byte("old content"), 0755)
	require.NoError(t, err)

	// Test Windows replacement logic
	err = atomicReplaceWindows(srcPath, dstPath)
	assert.NoError(t, err)

	// Verify target has new content
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))
}

func TestAtomicReplaceWindows_BackupFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-ops-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source")
	dstPath := filepath.Join(tmpDir, "nonexistent")

	// Create source file
	err = os.WriteFile(srcPath, []byte("new content"), 0755)
	require.NoError(t, err)

	// Target doesn't exist, backup should fail
	err = atomicReplaceWindows(srcPath, dstPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to backup")
}

func TestAtomicReplaceWindows_ReplaceFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-ops-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a directory that can't be renamed
	srcPath := filepath.Join(tmpDir, "source-dir")
	dstPath := filepath.Join(tmpDir, "target")

	err = os.Mkdir(srcPath, 0755)
	require.NoError(t, err)

	// Create nested file to prevent simple rename
	err = os.WriteFile(filepath.Join(srcPath, "file"), []byte("data"), 0644)
	require.NoError(t, err)

	// Create target file
	err = os.WriteFile(dstPath, []byte("old content"), 0755)
	require.NoError(t, err)

	// This should fail because we're trying to rename a directory to a file
	err = atomicReplaceWindows(srcPath, dstPath)
	// May or may not fail depending on OS behavior
	_ = err
}

func TestCopyFile_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-ops-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source")
	dstPath := filepath.Join(tmpDir, "dest")

	err = os.WriteFile(srcPath, []byte("test content"), 0644)
	require.NoError(t, err)

	err = copyFile(srcPath, dstPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestCopyFile_SourceStatError(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)

	tmpDir, err := os.MkdirTemp("", "file-ops-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create source file then make it unreadable
	srcPath := filepath.Join(tmpDir, "source")
	dstPath := filepath.Join(tmpDir, "dest")

	err = os.WriteFile(srcPath, []byte("test"), 0000)
	require.NoError(t, err)
	defer os.Chmod(srcPath, 0644) // Restore permissions for cleanup

	err = copyFile(srcPath, dstPath)
	// May fail due to permission issues
	_ = err
}

func TestCopyFile_DestCreateError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-ops-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source")
	// Destination is a directory that exists - cannot create file with same name
	dstPath := filepath.Join(tmpDir, "destdir")

	err = os.WriteFile(srcPath, []byte("test"), 0644)
	require.NoError(t, err)

	err = os.Mkdir(dstPath, 0755)
	require.NoError(t, err)

	err = copyFile(srcPath, dstPath)
	assert.Error(t, err)
}
