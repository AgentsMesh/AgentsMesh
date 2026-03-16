//go:build windows

package poddaemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPCPathWindows(t *testing.T) {
	path := IPCPath("", "test-pod")
	assert.Equal(t, `\\.\pipe\agentsmesh-test-pod`, path)
}

func TestIPCPathWindowsIgnoresDir(t *testing.T) {
	// The dir parameter is ignored on Windows.
	path1 := IPCPath("/some/dir", "my-pod")
	path2 := IPCPath("", "my-pod")
	assert.Equal(t, path1, path2)
}

func TestListenAndDialWindows(t *testing.T) {
	pipePath := IPCPath("", "test-listen-dial-"+t.Name())

	listener, err := Listen(pipePath)
	require.NoError(t, err)
	defer listener.Close()

	// Accept in a goroutine.
	errCh := make(chan error, 1)
	dataCh := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()

		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		if err != nil {
			errCh <- err
			return
		}
		dataCh <- string(buf[:n])
	}()

	// Dial and send data.
	conn, err := Dial(pipePath)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Write([]byte("hello"))
	require.NoError(t, err)

	select {
	case data := <-dataCh:
		assert.Equal(t, "hello", data)
	case err := <-errCh:
		t.Fatalf("server error: %v", err)
	}
}

func TestDialNonexistentPipe(t *testing.T) {
	_, err := Dial(`\\.\pipe\agentsmesh-nonexistent-` + t.Name())
	assert.Error(t, err)
}
