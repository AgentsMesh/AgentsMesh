//go:build !windows

package poddaemon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for daemon_io.go error paths: removeClient, validation,
// disconnection handling, and concurrent write safety.

func TestRemoveClientClearsState(t *testing.T) {
	d, _, _ := setupDaemonServer(t)

	c1, c2 := net.Pipe()
	defer c2.Close()

	d.clientMu.Lock()
	d.client = c1
	d.clientMu.Unlock()

	d.removeClient(c1)

	d.clientMu.Lock()
	assert.Nil(t, d.client)
	d.clientMu.Unlock()
}

func TestRemoveClientDifferentConnIgnored(t *testing.T) {
	d, _, _ := setupDaemonServer(t)

	c1, c1b := net.Pipe()
	c2, c2b := net.Pipe()
	defer c1.Close()
	defer c1b.Close()
	defer c2b.Close()

	d.clientMu.Lock()
	d.client = c1
	d.clientMu.Unlock()

	// Removing c2 should not clear client (it's c1)
	d.removeClient(c2)

	d.clientMu.Lock()
	assert.Equal(t, c1, d.client, "client should still be c1")
	d.clientMu.Unlock()
}

func TestClientReplacementDisconnectsPrevious(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	// First client attaches
	conn1, err := Dial(ipcPath)
	require.NoError(t, err)
	WriteMessage(conn1, MsgAttach, []byte{protocolVersion})
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn1) // AttachAck
	conn1.SetReadDeadline(time.Time{})

	// Second client connects — should kick the first
	conn2, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn2.Close()

	WriteMessage(conn2, MsgAttach, []byte{protocolVersion})
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, _, err := ReadMessage(conn2)
	require.NoError(t, err)
	assert.Equal(t, MsgAttachAck, msgType)

	// First client should detect disconnection
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = ReadMessage(conn1)
	assert.Error(t, err, "first client should be disconnected")
	conn1.Close()
}

func TestAcceptLoopStopsOnListenerClose(t *testing.T) {
	d, _, _ := setupDaemonServer(t)

	done := make(chan struct{})
	go func() {
		d.acceptLoop()
		close(done)
	}()

	d.listener.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("acceptLoop did not return after listener closed")
	}
}

func TestHandleClientReadError(t *testing.T) {
	d, _, _ := setupDaemonServer(t)

	c1, c2 := net.Pipe()
	c2.Close()

	d.handleClient(c1)

	d.clientMu.Lock()
	assert.Nil(t, d.client, "client should not be set on read error")
	d.clientMu.Unlock()
}

func TestReadClientCommandsDisconnect(t *testing.T) {
	d, _, _ := setupDaemonServer(t)

	clientConn, serverConn := net.Pipe()

	d.clientMu.Lock()
	d.client = clientConn
	d.clientMu.Unlock()

	done := make(chan struct{})
	go func() {
		d.readClientCommands(clientConn)
		close(done)
	}()

	serverConn.Close()

	select {
	case <-done:
		d.clientMu.Lock()
		assert.Nil(t, d.client)
		d.clientMu.Unlock()
	case <-time.After(2 * time.Second):
		t.Fatal("readClientCommands did not return on disconnect")
	}
}

func TestPtyReaderStopsOnReadError(t *testing.T) {
	d, proc, _ := setupDaemonServer(t)

	proc.readErr = errors.New("pty gone")
	close(proc.readCh)

	done := make(chan struct{})
	go func() {
		d.ptyReader()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ptyReader did not stop on read error")
	}
}

// TestResizeValidationRejectsInvalidDimensions verifies that resize messages
// with zero or oversized dimensions are ignored (P2 fix).
func TestResizeValidationRejectsInvalidDimensions(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn) // consume AttachAck
	conn.SetReadDeadline(time.Time{})

	originalCols, originalRows := d.state.Cols, d.state.Rows

	tests := []struct {
		name string
		cols uint16
		rows uint16
	}{
		{"zero cols", 0, 50},
		{"zero rows", 120, 0},
		{"both zero", 0, 0},
		{"cols too large", 1001, 50},
		{"rows too large", 120, 1001},
		{"both too large", 2000, 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make([]byte, 4)
			binary.BigEndian.PutUint16(payload[0:2], tt.cols)
			binary.BigEndian.PutUint16(payload[2:4], tt.rows)
			WriteMessage(conn, MsgResize, payload)
			time.Sleep(50 * time.Millisecond)

			d.clientMu.Lock()
			assert.Equal(t, originalCols, d.state.Cols, "cols should not change")
			assert.Equal(t, originalRows, d.state.Rows, "rows should not change")
			d.clientMu.Unlock()
		})
	}
}

// TestResizeShortPayloadIgnored verifies MsgResize with < 4 byte payload
// doesn't crash and is silently ignored.
func TestResizeShortPayloadIgnored(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn)
	conn.SetReadDeadline(time.Time{})

	originalCols, originalRows := d.state.Cols, d.state.Rows

	for _, size := range []int{0, 1, 2, 3} {
		WriteMessage(conn, MsgResize, make([]byte, size))
	}
	time.Sleep(100 * time.Millisecond)

	d.clientMu.Lock()
	assert.Equal(t, originalCols, d.state.Cols, "cols should not change with short payload")
	assert.Equal(t, originalRows, d.state.Rows, "rows should not change with short payload")
	d.clientMu.Unlock()
}

// TestWriteToClosedClientNoPanic verifies that ptyReader writing to a
// concurrently closed client connection doesn't panic.
func TestWriteToClosedClientNoPanic(t *testing.T) {
	d, proc, _ := setupDaemonServer(t)

	c1, c2 := net.Pipe()
	c2.Close()

	d.clientMu.Lock()
	d.client = c1
	d.clientMu.Unlock()

	go d.ptyReader()

	for i := 0; i < 10; i++ {
		proc.readCh <- []byte("data that goes nowhere")
	}
	close(proc.readCh)
	time.Sleep(100 * time.Millisecond)
}

// TestPtyReaderAndPingConcurrentWriteSafety verifies concurrent ptyReader
// output and Ping/Pong responses don't corrupt TLV frames (P0 fix).
func TestPtyReaderAndPingConcurrentWriteSafety(t *testing.T) {
	d, proc, ipcPath := setupDaemonServer(t)

	go d.ptyReader()
	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn) // AttachAck
	conn.SetReadDeadline(time.Time{})

	const iterations = 50

	go func() {
		for i := range iterations {
			proc.readCh <- []byte(fmt.Sprintf("output-%d", i))
			time.Sleep(time.Millisecond)
		}
	}()

	go func() {
		for range iterations {
			WriteMessage(conn, MsgPing, nil)
			time.Sleep(time.Millisecond)
		}
	}()

	outputCount := 0
	pongCount := 0
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for outputCount+pongCount < iterations*2 {
		msgType, _, err := ReadMessage(conn)
		if err != nil {
			break
		}
		switch msgType {
		case MsgOutput:
			outputCount++
		case MsgPong:
			pongCount++
		}
	}

	assert.Greater(t, outputCount, 0, "should receive some Output messages")
	assert.Greater(t, pongCount, 0, "should receive some Pong messages")
}

// TestRapidAttachDetachCycles verifies no resource leak after many cycles.
func TestRapidAttachDetachCycles(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)
	go d.acceptLoop()

	const cycles = 20
	for i := 0; i < cycles; i++ {
		conn, err := Dial(ipcPath)
		require.NoError(t, err)

		WriteMessage(conn, MsgAttach, []byte{protocolVersion})
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		msgType, _, err := ReadMessage(conn)
		require.NoError(t, err)
		assert.Equal(t, MsgAttachAck, msgType)

		conn.SetReadDeadline(time.Time{})
		WriteMessage(conn, MsgDetach, nil)
		time.Sleep(10 * time.Millisecond)
		conn.Close()
	}

	d.clientMu.Lock()
	assert.Nil(t, d.client, "client should be nil after all detach cycles")
	d.clientMu.Unlock()
}
