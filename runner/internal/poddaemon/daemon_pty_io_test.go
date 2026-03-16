package poddaemon

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentReadWaitRace verifies that concurrent Read() and Wait()
// don't cause deadlock. Read() gets EOF via outputCh closure, Wait() gets
// exit code via exitCh — they no longer compete for the same channel.
func TestConcurrentReadWaitRace(t *testing.T) {
	for i := 0; i < 20; i++ {
		func() {
			clientConn, serverConn := net.Pipe()
			defer serverConn.Close()
			d := newDaemonPTY(clientConn, 42, 80, 24)

			// Send exit code
			exitPayload := make([]byte, 4)
			binary.BigEndian.PutUint32(exitPayload, 99)
			go WriteMessage(serverConn, MsgExit, exitPayload)

			var wg sync.WaitGroup
			var readErr error
			var waitCode int
			var waitErr error

			wg.Add(2)
			go func() {
				defer wg.Done()
				buf := make([]byte, 64)
				_, readErr = d.Read(buf)
			}()
			go func() {
				defer wg.Done()
				waitCode, waitErr = d.Wait()
			}()

			wg.Wait()

			// Read gets EOF from outputCh closure, Wait gets exit code from exitCh.
			assert.Equal(t, io.EOF, readErr)
			require.NoError(t, waitErr)
			assert.Equal(t, 99, waitCode)

			d.Close()
		}()
	}
}

// TestOutputChannelSaturation verifies behavior when outputCh buffer (64)
// is full and recvLoop tries to push more data.
func TestOutputChannelSaturation(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	d := newDaemonPTY(clientConn, 42, 80, 24)
	defer serverConn.Close()

	// Fill the output channel (capacity 64) without reading
	done := make(chan struct{})
	go func() {
		for i := 0; i < 80; i++ {
			if err := WriteMessage(serverConn, MsgOutput, []byte("x")); err != nil {
				break
			}
		}
		close(done)
	}()

	// Wait a bit for saturation
	time.Sleep(300 * time.Millisecond)

	// Now drain — should get data without corruption
	buf := make([]byte, 4096)
	count := 0
	d.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, err := d.Read(buf)
		if err != nil {
			break
		}
		count++
	}

	assert.Greater(t, count, 0, "should have read some output before channel closed")
	d.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("writer goroutine didn't finish")
	}
}

// TestReadBufferedDataBeforeDeadline verifies that Read() returns buffered
// data even if a deadline is set, without waiting for the deadline.
func TestReadBufferedDataBeforeDeadline(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	d := newDaemonPTY(clientConn, 42, 80, 24)
	defer func() {
		d.Close()
		serverConn.Close()
	}()

	// Send output to fill the buffer
	go WriteMessage(serverConn, MsgOutput, []byte("buffered-data"))
	time.Sleep(50 * time.Millisecond)

	// First read consumes from outputCh into readBuf, returns partial
	buf := make([]byte, 5)
	d.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := d.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "buffe", string(buf[:n]))

	// Second read should return remaining buffered data immediately,
	// not wait for deadline.
	start := time.Now()
	d.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf2 := make([]byte, 64)
	n, err = d.Read(buf2)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, "red-data", string(buf2[:n]))
	assert.Less(t, elapsed, 100*time.Millisecond,
		"buffered read should be immediate, not wait for deadline")
}

// TestDaemonPTYReadAfterClose verifies Read returns error after Close.
func TestDaemonPTYReadAfterClose(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	d := newDaemonPTY(clientConn, 42, 80, 24)
	serverConn.Close()
	d.Close()

	buf := make([]byte, 64)
	_, err := d.Read(buf)
	assert.Error(t, err)
}

// TestDaemonPTYWriteAfterClose verifies Write returns error after Close.
func TestDaemonPTYWriteAfterClose(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	d := newDaemonPTY(clientConn, 42, 80, 24)
	serverConn.Close()
	d.Close()

	// Give recvLoop time to detect closed connection
	time.Sleep(50 * time.Millisecond)

	_, err := d.Write([]byte("data"))
	assert.Error(t, err)
}

// TestDaemonPTYDoubleClose verifies Close is idempotent.
func TestDaemonPTYDoubleClose(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	d := newDaemonPTY(clientConn, 42, 80, 24)

	// First close sends Detach
	go func() {
		ReadMessage(serverConn) // consume Detach
	}()

	err1 := d.Close()
	err2 := d.Close()
	assert.NoError(t, err1)
	_ = err2 // second close may or may not error, but shouldn't panic
}

// TestDaemonPTYWaitTimeout verifies Wait doesn't hang forever when
// no exit message arrives and connection is closed.
func TestDaemonPTYWaitTimeout(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	d := newDaemonPTY(clientConn, 42, 80, 24)

	// Close server side to trigger recvLoop exit
	serverConn.Close()

	done := make(chan struct{})
	var code int
	var err error
	go func() {
		code, err = d.Wait()
		close(done)
	}()

	// Wait should eventually return due to closedCh
	d.Close()

	select {
	case <-done:
		_ = code
		_ = err
	case <-time.After(3 * time.Second):
		t.Fatal("Wait() hung after connection close")
	}
}

// TestDaemonPTYReadLargeOutput verifies Read handles large output correctly.
func TestDaemonPTYReadLargeOutput(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	d := newDaemonPTY(clientConn, 42, 80, 24)
	defer func() {
		d.Close()
		serverConn.Close()
	}()

	// Send a large payload (just under max)
	largeData := make([]byte, 64*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	go WriteMessage(serverConn, MsgOutput, largeData)

	// Read it all back
	var received []byte
	buf := make([]byte, 4096)
	d.SetReadDeadline(time.Now().Add(2 * time.Second))
	for len(received) < len(largeData) {
		n, err := d.Read(buf)
		if err != nil {
			if err == os.ErrDeadlineExceeded || err == io.EOF {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
		received = append(received, buf[:n]...)
	}
	assert.Equal(t, largeData, received)
}
