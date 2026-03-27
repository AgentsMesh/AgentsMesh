package runner

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDirSelective_SkipsSessions(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create sessions/ directory (should be skipped)
	sessionsDir := filepath.Join(src, "sessions", "2026", "03")
	require.NoError(t, os.MkdirAll(sessionsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sessionsDir, "log.jsonl"), []byte("data"), 0644))

	// Create config.toml (should be copied)
	require.NoError(t, os.WriteFile(filepath.Join(src, "config.toml"), []byte("key = \"val\""), 0644))

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// config.toml should exist
	assert.FileExists(t, filepath.Join(dst, "config.toml"))
	// sessions/ should NOT exist
	_, err = os.Stat(filepath.Join(dst, "sessions"))
	assert.True(t, os.IsNotExist(err), "sessions/ should be skipped")
}

func TestCopyDirSelective_SymlinkToDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create a real directory with a file inside
	realDir := filepath.Join(src, "real-skill")
	require.NoError(t, os.MkdirAll(realDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(realDir, "init.js"), []byte("export default {}"), 0644))

	// Create a symlink to a directory (reproduces the original bug)
	symlinkDir := filepath.Join(src, "linked-skill")
	require.NoError(t, os.Symlink(realDir, symlinkDir))

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// Symlink should be preserved as a symlink
	destLink := filepath.Join(dst, "linked-skill")
	info, err := os.Lstat(destLink)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be a symlink")

	// Symlink target should be preserved
	target, err := os.Readlink(destLink)
	require.NoError(t, err)
	assert.Equal(t, realDir, target)

	// Real directory should be copied normally
	assert.FileExists(t, filepath.Join(dst, "real-skill", "init.js"))
}

func TestCopyDirSelective_SymlinkToFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create a real file and a symlink to it
	realFile := filepath.Join(src, "real-config.toml")
	require.NoError(t, os.WriteFile(realFile, []byte("key = \"val\""), 0644))
	require.NoError(t, os.Symlink(realFile, filepath.Join(src, "linked-config.toml")))

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// Symlink should be preserved
	destLink := filepath.Join(dst, "linked-config.toml")
	info, err := os.Lstat(destLink)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be a symlink")
}

func TestCopyDirSelective_DanglingSymlink(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create a dangling symlink (target does not exist)
	// Use filepath.FromSlash for cross-platform compatibility
	danglingTarget := filepath.FromSlash("/nonexistent/target")
	require.NoError(t, os.Symlink(danglingTarget, filepath.Join(src, "broken-link")))
	// Also create a real file so we can verify the copy still works
	require.NoError(t, os.WriteFile(filepath.Join(src, "good-file.txt"), []byte("ok"), 0644))

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// Dangling symlink should be recreated (preserving the link itself)
	destLink := filepath.Join(dst, "broken-link")
	target, err := os.Readlink(destLink)
	require.NoError(t, err)
	assert.Equal(t, danglingTarget, target)

	// Regular file should still be copied
	assert.FileExists(t, filepath.Join(dst, "good-file.txt"))
}

func TestCopyDirSelective_SkipsSocketFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix domain sockets not reliably supported on Windows")
	}

	// Use a short base path to avoid macOS 104-char Unix socket limit
	baseDir, err := os.MkdirTemp("/tmp", "cp-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(baseDir) })

	src := filepath.Join(baseDir, "src")
	dst := filepath.Join(baseDir, "dst")
	require.NoError(t, os.MkdirAll(src, 0755))
	require.NoError(t, os.MkdirAll(dst, 0755))

	// Create a Unix socket and keep the file after close
	socketPath := filepath.Join(src, "mcp.sock")
	addr := &net.UnixAddr{Name: socketPath, Net: "unix"}
	l, err := net.ListenUnix("unix", addr)
	require.NoError(t, err)
	l.SetUnlinkOnClose(false)
	l.Close()

	// Verify it's actually a socket
	info, err := os.Lstat(socketPath)
	require.NoError(t, err)
	require.True(t, info.Mode()&os.ModeSocket != 0, "test setup: should be a socket")

	// Also create a regular file
	require.NoError(t, os.WriteFile(filepath.Join(src, "config.toml"), []byte("ok"), 0644))

	err = copyDirSelective(src, dst)
	require.NoError(t, err)

	// Socket should NOT be copied
	_, err = os.Lstat(filepath.Join(dst, "mcp.sock"))
	assert.True(t, os.IsNotExist(err), "socket should be skipped")

	// Regular file should still be copied
	assert.FileExists(t, filepath.Join(dst, "config.toml"))
}

func TestCopyDirSelective_SkipsUnreadableFile(t *testing.T) {
	if os.Getuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("file permission tests not reliable on root or Windows")
	}

	src := t.TempDir()
	dst := t.TempDir()

	// Create an unreadable file
	unreadable := filepath.Join(src, "secret.key")
	require.NoError(t, os.WriteFile(unreadable, []byte("secret"), 0000))
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0644) })

	// Also create a readable file
	require.NoError(t, os.WriteFile(filepath.Join(src, "config.toml"), []byte("ok"), 0644))

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// Unreadable file should be skipped (not abort the copy)
	_, err = os.Stat(filepath.Join(dst, "secret.key"))
	assert.True(t, os.IsNotExist(err), "unreadable file should be skipped")

	// Readable file should still be copied
	assert.FileExists(t, filepath.Join(dst, "config.toml"))
}

func TestCopyDirSelective_SymlinkErrorDoesNotAbort(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create a symlink, then make the destination read-only so copySymlink fails on MkdirAll/Symlink
	require.NoError(t, os.Symlink("/some/target", filepath.Join(src, "link")))

	// Create a regular file that should still be copied
	require.NoError(t, os.WriteFile(filepath.Join(src, "config.toml"), []byte("ok"), 0644))

	// Make dst read-only for the symlink's parent creation to fail
	// We do this by pre-creating a *file* at the symlink's destination path
	// so os.Symlink fails with "file exists" (not a symlink)
	require.NoError(t, os.WriteFile(filepath.Join(dst, "link"), []byte("blocker"), 0644))

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// config.toml should still be copied despite symlink error
	assert.FileExists(t, filepath.Join(dst, "config.toml"))
}

func TestCopyDirSelective_MkdirErrorDoesNotAbort(t *testing.T) {
	if os.Getuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("file permission tests not reliable on root or Windows")
	}

	src := t.TempDir()
	dst := t.TempDir()

	// Create a subdirectory in source
	require.NoError(t, os.MkdirAll(filepath.Join(src, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "subdir", "nested.txt"), []byte("nested"), 0644))

	// Create a top-level file
	require.NoError(t, os.WriteFile(filepath.Join(src, "config.toml"), []byte("ok"), 0644))

	// Block subdir creation by placing a read-only file at the same path
	blocker := filepath.Join(dst, "subdir")
	require.NoError(t, os.WriteFile(blocker, []byte("blocker"), 0444))
	t.Cleanup(func() { _ = os.Chmod(blocker, 0644) })

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// config.toml should still be copied despite subdir error
	assert.FileExists(t, filepath.Join(dst, "config.toml"))
}

func TestCopyDirSelective_SkipsOversizedFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create a file just over 10 MiB
	oversized := make([]byte, 10<<20+1)
	require.NoError(t, os.WriteFile(filepath.Join(src, "huge.db"), oversized, 0644))

	// Also create a small file
	require.NoError(t, os.WriteFile(filepath.Join(src, "config.toml"), []byte("ok"), 0644))

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// Oversized file should be skipped
	_, err = os.Stat(filepath.Join(dst, "huge.db"))
	assert.True(t, os.IsNotExist(err), "oversized file should be skipped")

	// Small file should be copied
	assert.FileExists(t, filepath.Join(dst, "config.toml"))
}
