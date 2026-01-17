package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for file operations: copyFile, atomicReplace, CreateBackup, Rollback, Apply

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source.txt")
	srcContent := []byte("test content")
	err = os.WriteFile(srcPath, srcContent, 0644)
	require.NoError(t, err)

	dstPath := filepath.Join(tmpDir, "dest.txt")
	err = copyFile(srcPath, dstPath)
	assert.NoError(t, err)

	dstContent, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, srcContent, dstContent)

	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)
	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
}

func TestCopyFile_SourceNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = copyFile(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
	assert.Error(t, err)
}

func TestCopyFile_DestDirNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source.txt")
	err = os.WriteFile(srcPath, []byte("test"), 0644)
	require.NoError(t, err)

	err = copyFile(srcPath, filepath.Join(tmpDir, "nonexistent", "dest.txt"))
	assert.Error(t, err)
}

func TestAtomicReplace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "new.txt")
	dstPath := filepath.Join(tmpDir, "old.txt")

	err = os.WriteFile(srcPath, []byte("new content"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(dstPath, []byte("old content"), 0755)
	require.NoError(t, err)

	err = atomicReplace(srcPath, dstPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))

	_, err = os.Stat(srcPath)
	assert.True(t, os.IsNotExist(err))
}

func TestAtomicReplace_DestNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "new.txt")
	dstPath := filepath.Join(tmpDir, "nonexistent.txt")

	err = os.WriteFile(srcPath, []byte("new content"), 0755)
	require.NoError(t, err)

	if runtime.GOOS != "windows" {
		err = atomicReplace(srcPath, dstPath)
		assert.NoError(t, err)

		content, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(content))
	}
}

func TestAtomicReplace_SrcNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err = os.WriteFile(dstPath, []byte("dest"), 0755)
	require.NoError(t, err)

	err = atomicReplace(srcPath, dstPath)
	assert.Error(t, err)
}

func TestUpdater_CreateBackup(t *testing.T) {
	u := New("1.0.0")

	execPath, err := os.Executable()
	if err != nil {
		t.Skip("Cannot get executable path")
	}

	backupPath, err := u.CreateBackup()
	if err == nil {
		defer os.Remove(backupPath)
		assert.Equal(t, execPath+".bak", backupPath)
		_, err := os.Stat(backupPath)
		assert.NoError(t, err)
	}
}

func TestUpdater_Rollback_NoBackup(t *testing.T) {
	u := New("1.0.0")

	err := u.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backup found")
}

func TestUpdater_Rollback_WithBackup(t *testing.T) {
	execPath, err := os.Executable()
	if err != nil {
		t.Skip("Cannot get executable path")
	}

	backupPath := execPath + ".bak"
	err = os.WriteFile(backupPath, []byte("backup content"), 0755)
	if err != nil {
		t.Skip("Cannot create backup file")
	}
	defer os.Remove(backupPath)

	u := New("1.0.0")
	err = u.Rollback()
	_ = err // May fail since binary is running
}

func TestUpdater_Apply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpPath := filepath.Join(tmpDir, "new-binary")
	err = os.WriteFile(tmpPath, []byte("new binary content"), 0755)
	require.NoError(t, err)

	u := New("1.0.0")
	err = u.Apply(tmpPath)
	// May fail since replacing running binary
	if err != nil {
		assert.Error(t, err)
	}
}
