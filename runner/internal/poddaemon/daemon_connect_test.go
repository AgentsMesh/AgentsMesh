//go:build !windows

package poddaemon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectDaemonMalformedAttachAck verifies connectDaemon handles
// corrupted JSON in AttachAck gracefully.
func TestConnectDaemonMalformedAttachAck(t *testing.T) {
	dir := shortSocketDir(t)
	ipcPath := IPCPath(dir, "mal")

	listener, err := Listen(ipcPath)
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		ReadMessage(conn) // consume Attach
		WriteMessage(conn, MsgAttachAck, []byte("{broken json"))
	}()

	_, err = connectDaemon(ipcPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal attach ack")
}

// TestConnectDaemonWrongMessageType verifies connectDaemon handles
// unexpected message type instead of AttachAck.
func TestConnectDaemonWrongMessageType(t *testing.T) {
	dir := shortSocketDir(t)
	ipcPath := IPCPath(dir, "wr")

	listener, err := Listen(ipcPath)
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		ReadMessage(conn) // consume Attach
		WriteMessage(conn, MsgOutput, []byte("surprise"))
	}()

	_, err = connectDaemon(ipcPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected AttachAck")
}

// TestConnectDaemonHandshakeTimeout verifies the 5-second timeout on
// AttachAck response.
func TestConnectDaemonHandshakeTimeout(t *testing.T) {
	dir := shortSocketDir(t)
	ipcPath := IPCPath(dir, "to")

	listener, err := Listen(ipcPath)
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		ReadMessage(conn)
		time.Sleep(10 * time.Second)
	}()

	start := time.Now()
	_, err = connectDaemon(ipcPath)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "read attach ack")
	assert.InDelta(t, 5.0, elapsed.Seconds(), 1.0,
		"should timeout around 5 seconds, got %v", elapsed)
}

// TestConnectDaemonDialFailure verifies connectDaemon returns clear error
// when IPC socket doesn't exist.
func TestConnectDaemonDialFailure(t *testing.T) {
	_, err := connectDaemon("/nonexistent/path.sock")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dial daemon")
}

// TestConnectDaemonSuccess verifies the full connectDaemon happy path.
func TestConnectDaemonSuccess(t *testing.T) {
	dir := shortSocketDir(t)
	ipcPath := IPCPath(dir, "test")

	listener, err := Listen(ipcPath)
	require.NoError(t, err)
	defer listener.Close()

	// Mock daemon: accept, read Attach, send AttachAck
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		msgType, _, err := ReadMessage(conn)
		if err != nil || msgType != MsgAttach {
			return
		}

		ack := attachAckPayload{PID: 99, Cols: 80, Rows: 24, Alive: true}
		data, _ := json.Marshal(ack)
		WriteMessage(conn, MsgAttachAck, data)

		// Keep connection open for client
		time.Sleep(500 * time.Millisecond)
	}()

	d, err := connectDaemon(ipcPath)
	require.NoError(t, err)
	defer d.Close()

	assert.Equal(t, 99, d.Pid())
	cols, rows, _ := d.GetSize()
	assert.Equal(t, 80, cols)
	assert.Equal(t, 24, rows)
}
