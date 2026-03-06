package updater

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithExecPathFunc(t *testing.T) {
	customPath := "/custom/path/runner"
	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return customPath, nil
	}))

	path, err := u.execPathFunc()
	assert.NoError(t, err)
	assert.Equal(t, customPath, path)
}

func TestWithExecPathFunc_Error(t *testing.T) {
	expectedErr := errors.New("path error")
	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return "", expectedErr
	}))

	_, err := u.execPathFunc()
	assert.Equal(t, expectedErr, err)
}

func TestUpdater_Apply_WithCustomExecPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apply-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "new-binary")
	dstPath := filepath.Join(tmpDir, "runner")

	err = os.WriteFile(srcPath, []byte("new binary"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(dstPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return dstPath, nil
	}))

	err = u.Apply(srcPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "new binary", string(content))
}

func TestUpdater_Apply_ExecPathError(t *testing.T) {
	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return "", errors.New("cannot get path")
	}))

	err := u.Apply("/tmp/some-file")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get executable path")
}

func TestUpdater_Rollback_WithCustomExecPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	backupPath := execPath + ".bak"

	err = os.WriteFile(execPath, []byte("new binary"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(backupPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return execPath, nil
	}))

	err = u.Rollback()
	assert.NoError(t, err)

	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "old binary", string(content))
}

func TestUpdater_Rollback_ExecPathError(t *testing.T) {
	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return "", errors.New("cannot get path")
	}))

	err := u.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get executable path")
}

func TestUpdater_CreateBackup_WithCustomExecPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "backup-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("original binary"), 0755)
	require.NoError(t, err)

	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return execPath, nil
	}))

	backupPath, err := u.CreateBackup()
	assert.NoError(t, err)
	assert.Equal(t, execPath+".bak", backupPath)

	content, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "original binary", string(content))
}

func TestUpdater_CreateBackup_ExecPathError(t *testing.T) {
	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return "", errors.New("cannot get path")
	}))

	_, err := u.CreateBackup()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get executable path")
}

func TestUpdater_Rollback_AtomicReplaceError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping: Windows allows rename of file over directory")
	}
	tmpDir, err := os.MkdirTemp("", "rollback-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	backupPath := execPath + ".bak"

	// Create backup file but make exec path a directory (causes rename error)
	err = os.WriteFile(backupPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	err = os.Mkdir(execPath, 0755)
	require.NoError(t, err)

	u := New("1.0.0", WithExecPathFunc(func() (string, error) {
		return execPath, nil
	}))

	err = u.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restore backup")
}
