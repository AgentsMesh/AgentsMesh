package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for Apply and atomicReplace functionality

func TestAtomicReplace_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atomic-test-*")
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

	// Perform atomic replace
	err = atomicReplace(srcPath, dstPath)
	assert.NoError(t, err)

	// Verify target has new content
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))

	// Source should be gone (renamed to target)
	_, err = os.Stat(srcPath)
	assert.True(t, os.IsNotExist(err))
}

func TestAtomicReplace_TargetNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atomic-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source")
	dstPath := filepath.Join(tmpDir, "nonexistent")

	// Create source file
	err = os.WriteFile(srcPath, []byte("new content"), 0755)
	require.NoError(t, err)

	if runtime.GOOS == "windows" {
		// On Windows, target must exist for rename
		err = atomicReplace(srcPath, dstPath)
		assert.Error(t, err)
	} else {
		// On Unix, rename works even if target doesn't exist
		err = atomicReplace(srcPath, dstPath)
		assert.NoError(t, err)

		content, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(content))
	}
}

func TestAtomicReplace_SourceNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atomic-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "nonexistent")
	dstPath := filepath.Join(tmpDir, "target")

	// Create target file
	err = os.WriteFile(dstPath, []byte("old content"), 0755)
	require.NoError(t, err)

	err = atomicReplace(srcPath, dstPath)
	assert.Error(t, err)

	// Target should still have old content
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "old content", string(content))
}

func TestAtomicReplace_InvalidPath(t *testing.T) {
	err := atomicReplace("/nonexistent/path/source", "/nonexistent/path/target")
	assert.Error(t, err)
}

func TestUpdater_Apply_InvalidPath(t *testing.T) {
	u := New("1.0.0")
	err := u.Apply("/nonexistent/path/file")
	assert.Error(t, err)
}

func TestUpdater_CreateBackup_AndRollback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "backup-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a fake executable
	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("original binary"), 0755)
	require.NoError(t, err)

	// Note: Cannot fully test CreateBackup without mocking os.Executable
	// but we can test the underlying copyFile function

	backupPath := execPath + ".bak"
	err = copyFile(execPath, backupPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "original binary", string(content))
}
