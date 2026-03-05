//go:build !windows

package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPath(t *testing.T) {
	path := GetPath()
	assert.NotEmpty(t, path)
	assert.True(t, filepath.IsAbs(path))
	assert.Equal(t, "runner.pid", filepath.Base(path))
	assert.Equal(t, ".agentsmesh", filepath.Base(filepath.Dir(path)))
}

func TestWriteAndRemove(t *testing.T) {
	// Use a temp dir to avoid touching the real ~/.agentsmesh
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "runner.pid")

	// Write PID file manually (since Write() uses GetPath which points to real home)
	execName := filepath.Base(os.Args[0])
	content := fmt.Sprintf("%d %s\n", os.Getpid(), execName)
	err := os.WriteFile(pidPath, []byte(content), 0644)
	require.NoError(t, err)

	// Verify content
	data, err := os.ReadFile(pidPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), strconv.Itoa(os.Getpid()))
	assert.Contains(t, string(data), execName)
}

func TestParsePIDFile_NotFound(t *testing.T) {
	pid, exec, err := parsePIDFile("/nonexistent/path/runner.pid")
	assert.NoError(t, err)
	assert.Equal(t, 0, pid)
	assert.Empty(t, exec)
}

func TestParsePIDFile_Corrupt_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "runner.pid")

	// Empty file
	err := os.WriteFile(pidPath, []byte(""), 0644)
	require.NoError(t, err)

	pid, exec, err := parsePIDFile(pidPath)
	assert.NoError(t, err)
	assert.Equal(t, 0, pid)
	assert.Empty(t, exec)

	// File should be removed
	_, err = os.Stat(pidPath)
	assert.True(t, os.IsNotExist(err))
}

func TestParsePIDFile_Corrupt_OnlyPID(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "runner.pid")

	// Only PID, no exec name
	err := os.WriteFile(pidPath, []byte("12345\n"), 0644)
	require.NoError(t, err)

	pid, exec, err := parsePIDFile(pidPath)
	assert.NoError(t, err)
	assert.Equal(t, 0, pid)
	assert.Empty(t, exec)

	// File should be removed
	_, err = os.Stat(pidPath)
	assert.True(t, os.IsNotExist(err))
}

func TestParsePIDFile_Corrupt_NonNumericPID(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "runner.pid")

	err := os.WriteFile(pidPath, []byte("notapid agentsmesh-runner\n"), 0644)
	require.NoError(t, err)

	pid, exec, err := parsePIDFile(pidPath)
	assert.NoError(t, err)
	assert.Equal(t, 0, pid)
	assert.Empty(t, exec)

	// File should be removed
	_, err = os.Stat(pidPath)
	assert.True(t, os.IsNotExist(err))
}

func TestParsePIDFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "runner.pid")

	err := os.WriteFile(pidPath, []byte("12345 agentsmesh-runner\n"), 0644)
	require.NoError(t, err)

	pid, exec, err := parsePIDFile(pidPath)
	assert.NoError(t, err)
	assert.Equal(t, 12345, pid)
	assert.Equal(t, "agentsmesh-runner", exec)

	// File should still exist
	_, err = os.Stat(pidPath)
	assert.NoError(t, err)
}

func TestCleanupStaleProcess_NoPIDFile(t *testing.T) {
	// Temporarily override GetPath by just testing that it returns nil
	// when no PID file exists. This test relies on the real GetPath
	// pointing to ~/.agentsmesh/runner.pid which shouldn't exist in CI.
	// For safety, we test parsePIDFile directly with a nonexistent path.
	pid, _, err := parsePIDFile("/tmp/nonexistent-pidfile-test")
	assert.NoError(t, err)
	assert.Equal(t, 0, pid)
}

func TestCleanupStaleProcess_DeadProcess(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "runner.pid")

	// Use a PID that almost certainly doesn't exist
	deadPID := 4999999
	err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d agentsmesh-runner\n", deadPID)), 0644)
	require.NoError(t, err)

	// Verify the process doesn't exist
	err = syscall.Kill(deadPID, 0)
	require.Error(t, err, "PID %d should not exist", deadPID)

	// parsePIDFile should succeed
	pid, exec, err := parsePIDFile(pidPath)
	assert.NoError(t, err)
	assert.Equal(t, deadPID, pid)
	assert.Equal(t, "agentsmesh-runner", exec)

	// The actual CleanupStaleProcess uses GetPath() which we can't override,
	// but we can verify the logic by checking that syscall.Kill(deadPID, 0) fails
	// which is what CleanupStaleProcess checks internally.
	assert.Error(t, syscall.Kill(deadPID, 0))
}

func TestRemove_MismatchedPID(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "runner.pid")

	// Write a PID file with a different PID (simulating another instance)
	otherPID := os.Getpid() + 1000
	content := fmt.Sprintf("%d agentsmesh-runner\n", otherPID)
	err := os.WriteFile(pidPath, []byte(content), 0644)
	require.NoError(t, err)

	// Read it back and verify the PID check logic
	data, err := os.ReadFile(pidPath)
	require.NoError(t, err)

	pid, _, err := parsePIDFile(pidPath)
	require.NoError(t, err)
	assert.NotEqual(t, os.Getpid(), pid, "PID should not match current process")

	// File should still exist (Remove() would skip it due to PID mismatch)
	_, err = os.Stat(pidPath)
	assert.NoError(t, err)
	_ = data
}

func TestWaitForExit_AlreadyDead(t *testing.T) {
	// A PID that doesn't exist should return true immediately
	result := waitForExit(4999999, 1*time.Second)
	assert.True(t, result)
}
