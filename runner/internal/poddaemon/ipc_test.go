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
	got := IPCPath("/tmp/sandbox", "my-pod")
	assert.Equal(t, filepath.Join("/tmp/sandbox", "my-pod.sock"), got)
}

func TestIPCPathEmptyName(t *testing.T) {
	got := IPCPath("/tmp/sandbox", "")
	assert.Equal(t, filepath.Join("/tmp/sandbox", ".sock"), got)
}

func TestIPCPathLongPathFallback(t *testing.T) {
	// Create a path that would exceed the Unix socket limit
	longDir := "/var/folders/fd/s_43n3d57433mxqq58bgxxth0000gn/T/agentsmesh-workspace/sandboxes/1-standalone-567ec13b"
	longName := "1-standalone-567ec13b"
	candidate := filepath.Join(longDir, longName+".sock")
	require.Greater(t, len(candidate), maxSocketPath(), "test path should exceed socket limit")

	got := IPCPath(longDir, longName)
	assert.Less(t, len(got), maxSocketPath(), "fallback path should be within limit")
	assert.Contains(t, got, "am-")
	assert.True(t, filepath.IsAbs(got))

	// Deterministic: same input produces same output
	got2 := IPCPath(longDir, longName)
	assert.Equal(t, got, got2)
}

func TestIPCPathLongPathDifferentInputs(t *testing.T) {
	longDir := "/var/folders/xx/very_long_directory_name_that_exceeds_limit/T/agentsmesh-workspace/sandboxes"
	p1 := IPCPath(longDir, "pod-aaaa-bbbb-cccc-dddd")
	p2 := IPCPath(longDir, "pod-eeee-ffff-gggg-hhhh")
	assert.NotEqual(t, p1, p2, "different inputs should produce different paths")
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
