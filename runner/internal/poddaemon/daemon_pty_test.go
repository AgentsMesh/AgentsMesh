package poddaemon

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDaemonPTY creates a daemonPTY connected to a mock daemon via net.Pipe.
// Note: net.Pipe is synchronous — reads/writes block until the other end acts.
// Tests must use goroutines for concurrent read/write.
func setupDaemonPTY(t *testing.T) (*daemonPTY, net.Conn) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	d := newDaemonPTY(clientConn, 42, 80, 24)
	t.Cleanup(func() {
		d.Close()
		serverConn.Close()
	})
	return d, serverConn
}

func TestDaemonPTYReadOutput(t *testing.T) {
	d, server := setupDaemonPTY(t)

	go func() {
		WriteMessage(server, MsgOutput, []byte("hello"))
	}()

	buf := make([]byte, 64)
	n, err := d.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(buf[:n]))
}

func TestDaemonPTYReadDeadline(t *testing.T) {
	d, _ := setupDaemonPTY(t)

	d.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

	buf := make([]byte, 64)
	_, err := d.Read(buf)
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded)
}

func TestDaemonPTYReadDeadlineExpired(t *testing.T) {
	d, _ := setupDaemonPTY(t)

	d.SetReadDeadline(time.Now().Add(-1 * time.Millisecond))

	buf := make([]byte, 64)
	_, err := d.Read(buf)
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded)
}

func TestDaemonPTYWriteSendsInput(t *testing.T) {
	d, server := setupDaemonPTY(t)

	// net.Pipe is synchronous: read from server concurrently with write from d.
	resultCh := make(chan struct {
		msgType byte
		payload []byte
		err     error
	}, 1)
	go func() {
		mt, p, err := ReadMessage(server)
		resultCh <- struct {
			msgType byte
			payload []byte
			err     error
		}{mt, p, err}
	}()

	_, err := d.Write([]byte("user input"))
	require.NoError(t, err)

	result := <-resultCh
	require.NoError(t, result.err)
	assert.Equal(t, MsgInput, result.msgType)
	assert.Equal(t, []byte("user input"), result.payload)
}

func TestDaemonPTYResizeSendsResize(t *testing.T) {
	d, server := setupDaemonPTY(t)

	resultCh := make(chan struct {
		msgType byte
		payload []byte
		err     error
	}, 1)
	go func() {
		mt, p, err := ReadMessage(server)
		resultCh <- struct {
			msgType byte
			payload []byte
			err     error
		}{mt, p, err}
	}()

	err := d.Resize(120, 40)
	require.NoError(t, err)

	result := <-resultCh
	require.NoError(t, result.err)
	assert.Equal(t, MsgResize, result.msgType)
	assert.Len(t, result.payload, 4)
	assert.Equal(t, uint16(120), binary.BigEndian.Uint16(result.payload[0:2]))
	assert.Equal(t, uint16(40), binary.BigEndian.Uint16(result.payload[2:4]))
}

func TestDaemonPTYCloseSendsDetach(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	d := newDaemonPTY(clientConn, 42, 80, 24)

	resultCh := make(chan struct {
		msgType byte
		err     error
	}, 1)
	go func() {
		mt, _, err := ReadMessage(serverConn)
		resultCh <- struct {
			msgType byte
			err     error
		}{mt, err}
	}()

	err := d.Close()
	require.NoError(t, err)

	result := <-resultCh
	require.NoError(t, result.err)
	assert.Equal(t, MsgDetach, result.msgType)
	serverConn.Close()
}

func TestDaemonPTYExitCode(t *testing.T) {
	d, server := setupDaemonPTY(t)

	exitPayload := make([]byte, 4)
	binary.BigEndian.PutUint32(exitPayload, 42)
	go func() {
		WriteMessage(server, MsgExit, exitPayload)
	}()

	code, err := d.Wait()
	require.NoError(t, err)
	assert.Equal(t, 42, code)
}

func TestDaemonPTYReadEOFAfterExit(t *testing.T) {
	d, server := setupDaemonPTY(t)

	exitPayload := make([]byte, 4)
	binary.BigEndian.PutUint32(exitPayload, 0)
	go func() {
		WriteMessage(server, MsgExit, exitPayload)
	}()

	// Wait for recvLoop to process the exit message
	time.Sleep(100 * time.Millisecond)

	buf := make([]byte, 64)
	_, err := d.Read(buf)
	assert.Equal(t, io.EOF, err)
}

func TestDaemonPTYGracefulStop(t *testing.T) {
	d, server := setupDaemonPTY(t)

	resultCh := make(chan byte, 1)
	go func() {
		mt, _, _ := ReadMessage(server)
		resultCh <- mt
	}()

	err := d.GracefulStop()
	require.NoError(t, err)
	assert.Equal(t, MsgGracefulStop, <-resultCh)
}

func TestDaemonPTYKill(t *testing.T) {
	d, server := setupDaemonPTY(t)

	resultCh := make(chan byte, 1)
	go func() {
		mt, _, _ := ReadMessage(server)
		resultCh <- mt
	}()

	err := d.Kill()
	require.NoError(t, err)
	assert.Equal(t, MsgKill, <-resultCh)
}

func TestDaemonPTYPid(t *testing.T) {
	d, _ := setupDaemonPTY(t)
	assert.Equal(t, 42, d.Pid())
}

func TestDaemonPTYGetSize(t *testing.T) {
	d, _ := setupDaemonPTY(t)
	cols, rows, err := d.GetSize()
	require.NoError(t, err)
	assert.Equal(t, 80, cols)
	assert.Equal(t, 24, rows)
}

// --- Tests targeting specific bug fixes ---

// TestReadDrainsOutputBeforeEOF verifies that when the daemon sends Output
// messages followed by Exit, Read() returns all buffered output before EOF.
// This was a bug where Read() could pick exitCh over outputCh due to Go's
// random select, skipping buffered data.
func TestReadDrainsOutputBeforeEOF(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	d := newDaemonPTY(clientConn, 42, 80, 24)
	defer d.Close()

	// Send 3 output messages then exit — simulates daemon child exiting.
	go func() {
		WriteMessage(serverConn, MsgOutput, []byte("aaa"))
		WriteMessage(serverConn, MsgOutput, []byte("bbb"))
		WriteMessage(serverConn, MsgOutput, []byte("ccc"))

		exitPayload := make([]byte, 4)
		binary.BigEndian.PutUint32(exitPayload, 0)
		WriteMessage(serverConn, MsgExit, exitPayload)
	}()

	// Read all output — must get all 3 chunks before EOF.
	var collected []byte
	buf := make([]byte, 64)
	d.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		n, err := d.Read(buf)
		if n > 0 {
			collected = append(collected, buf[:n]...)
		}
		if err != nil {
			assert.Equal(t, io.EOF, err)
			break
		}
	}
	assert.Equal(t, "aaabbbccc", string(collected), "all output must be delivered before EOF")
}

// TestWaitReturnsExitCodeAfterReadEOF verifies Wait() returns the correct
// exit code even after Read() has already returned EOF.
func TestWaitReturnsExitCodeAfterReadEOF(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	d := newDaemonPTY(clientConn, 42, 80, 24)

	exitPayload := make([]byte, 4)
	binary.BigEndian.PutUint32(exitPayload, 42)
	go func() {
		WriteMessage(serverConn, MsgExit, exitPayload)
	}()

	// Wait for recvLoop to process the exit message
	time.Sleep(100 * time.Millisecond)

	// Read returns EOF (outputCh closed by recvLoop)
	buf := make([]byte, 64)
	_, err := d.Read(buf)
	assert.Equal(t, io.EOF, err)

	// Wait should still get the exit code from exitCh
	code, err := d.Wait()
	require.NoError(t, err)
	assert.Equal(t, 42, code)
}
