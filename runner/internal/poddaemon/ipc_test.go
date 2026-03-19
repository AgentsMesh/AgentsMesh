//go:build !windows

package poddaemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPCPath(t *testing.T) {
	got := IPCPath("/tmp/test-sockets", "my-pod")
	assert.Equal(t, filepath.Join("/tmp/test-sockets", "my-pod.sock"), got)
}

func TestIPCPathEmptyName(t *testing.T) {
	got := IPCPath("/tmp/test-sockets", "")
	assert.Equal(t, filepath.Join("/tmp/test-sockets", ".sock"), got)
}

func TestEnsureSocketDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "sockets")
	require.NoError(t, EnsureSocketDir(dir))

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// shortDir creates a short temp dir to avoid macOS 104-byte Unix socket path limit.
func shortDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "pd-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestListenAndDial(t *testing.T) {
	dir := shortDir(t)
	ipcPath := IPCPath(dir, "t")

	listener, err := Listen(ipcPath)
	require.NoError(t, err)
	defer listener.Close()

	connCh := make(chan struct{}, 1)
	go func() {
		conn, err := Dial(ipcPath)
		if err == nil {
			conn.Close()
		}
		connCh <- struct{}{}
	}()

	conn, err := listener.Accept()
	require.NoError(t, err)
	conn.Close()

	<-connCh
}

func TestListenInvalidPath(t *testing.T) {
	_, err := Listen("/nonexistent/dir/test.sock")
	assert.Error(t, err)
}

func TestDialNonexistentSocket(t *testing.T) {
	_, err := Dial("/nonexistent/path.sock")
	assert.Error(t, err)
}

// TestListenCleansUpStaleSocket verifies that Listen() removes a pre-existing
// stale socket file (left by a crashed daemon) before creating a new listener
// (P2 fix: stale socket cleanup).
func TestListenCleansUpStaleSocket(t *testing.T) {
	dir := shortDir(t)
	ipcPath := IPCPath(dir, "stale")

	// Create a stale socket file (simulates a crashed daemon)
	err := os.WriteFile(ipcPath, []byte("stale"), 0600)
	require.NoError(t, err)

	// Listen should succeed despite the stale file
	listener, err := Listen(ipcPath)
	require.NoError(t, err, "Listen should succeed even with stale socket file")
	defer listener.Close()

	// Verify the listener actually works
	connCh := make(chan struct{}, 1)
	go func() {
		conn, err := Dial(ipcPath)
		if err == nil {
			conn.Close()
		}
		connCh <- struct{}{}
	}()

	conn, err := listener.Accept()
	require.NoError(t, err)
	conn.Close()
	<-connCh
}

// TestListenTwiceOnSamePath verifies that a second Listen() on the same path
// succeeds after the first listener is closed (also depends on stale cleanup).
func TestListenTwiceOnSamePath(t *testing.T) {
	dir := shortDir(t)
	ipcPath := IPCPath(dir, "twice")

	listener1, err := Listen(ipcPath)
	require.NoError(t, err)
	listener1.Close()

	// Second listen should clean up the socket left by the first
	listener2, err := Listen(ipcPath)
	require.NoError(t, err, "second Listen should succeed after first is closed")
	listener2.Close()
}

func TestListenAndDialBidirectional(t *testing.T) {
	dir := shortDir(t)
	ipcPath := IPCPath(dir, "b")

	listener, err := Listen(ipcPath)
	require.NoError(t, err)
	defer listener.Close()

	serverReady := make(chan struct{})
	serverDone := make(chan string, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		close(serverReady)

		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		if err == nil {
			serverDone <- string(buf[:n])
		}
	}()

	clientConn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer clientConn.Close()

	<-serverReady

	_, err = clientConn.Write([]byte("hello"))
	require.NoError(t, err)

	got := <-serverDone
	assert.Equal(t, "hello", got)
}
