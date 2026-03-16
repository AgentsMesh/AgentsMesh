//go:build !windows

package poddaemon

import (
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleClientAttachHandshake(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	err = WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	require.NoError(t, err)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, payload, err := ReadMessage(conn)
	require.NoError(t, err)
	assert.Equal(t, MsgAttachAck, msgType)

	var ack attachAckPayload
	require.NoError(t, json.Unmarshal(payload, &ack))
	assert.Equal(t, 123, ack.PID)
	assert.Equal(t, 80, ack.Cols)
	assert.Equal(t, 24, ack.Rows)
	assert.True(t, ack.Alive)
}

func TestHandleClientRejectsNonAttachMessage(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	// Send Input instead of Attach — should be rejected
	err = WriteMessage(conn, MsgInput, []byte("bad"))
	require.NoError(t, err)

	// Connection should be closed by daemon
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, _, err = ReadMessage(conn)
	assert.Error(t, err) // EOF or read error
}

func TestReadClientCommandsInput(t *testing.T) {
	d, proc, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn) // consume AttachAck
	conn.SetReadDeadline(time.Time{})

	WriteMessage(conn, MsgInput, []byte("hello daemon"))

	time.Sleep(100 * time.Millisecond)

	proc.mu.Lock()
	got := string(proc.writeData)
	proc.mu.Unlock()
	assert.Equal(t, "hello daemon", got)
}

func TestReadClientCommandsResize(t *testing.T) {
	d, proc, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn)
	conn.SetReadDeadline(time.Time{})

	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[0:2], 200)
	binary.BigEndian.PutUint16(payload[2:4], 50)
	WriteMessage(conn, MsgResize, payload)

	time.Sleep(100 * time.Millisecond)

	proc.mu.Lock()
	assert.Equal(t, 200, proc.resizeCols)
	assert.Equal(t, 50, proc.resizeRows)
	proc.mu.Unlock()

	d.clientMu.Lock()
	assert.Equal(t, 200, d.state.Cols)
	assert.Equal(t, 50, d.state.Rows)
	d.clientMu.Unlock()
}

func TestReadClientCommandsGracefulStop(t *testing.T) {
	d, proc, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn)
	conn.SetReadDeadline(time.Time{})

	WriteMessage(conn, MsgGracefulStop, nil)
	time.Sleep(100 * time.Millisecond)

	proc.mu.Lock()
	assert.Equal(t, 1, proc.gracefulStopCount)
	proc.mu.Unlock()
}

func TestReadClientCommandsKill(t *testing.T) {
	d, proc, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn)
	conn.SetReadDeadline(time.Time{})

	WriteMessage(conn, MsgKill, nil)
	time.Sleep(100 * time.Millisecond)

	proc.mu.Lock()
	assert.Equal(t, 1, proc.killCount)
	proc.mu.Unlock()
}

func TestReadClientCommandsPing(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn)

	WriteMessage(conn, MsgPing, nil)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, _, err := ReadMessage(conn)
	require.NoError(t, err)
	assert.Equal(t, MsgPong, msgType)
}

func TestReadClientCommandsDetach(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn)
	conn.SetReadDeadline(time.Time{})

	WriteMessage(conn, MsgDetach, nil)
	time.Sleep(100 * time.Millisecond)

	d.clientMu.Lock()
	assert.Nil(t, d.client, "client should be nil after detach")
	d.clientMu.Unlock()
}

func TestPtyReaderForwardsOutput(t *testing.T) {
	d, proc, ipcPath := setupDaemonServer(t)

	go d.ptyReader()
	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn) // AttachAck

	proc.readCh <- []byte("pty output line 1")

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, payload, err := ReadMessage(conn)
	require.NoError(t, err)
	assert.Equal(t, MsgOutput, msgType)
	assert.Equal(t, "pty output line 1", string(payload))
}

func TestPtyReaderNoClientDiscards(t *testing.T) {
	d, proc, _ := setupDaemonServer(t)

	go d.ptyReader()

	proc.readCh <- []byte("discarded data")

	close(proc.readCh)
	time.Sleep(100 * time.Millisecond)
}
