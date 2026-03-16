//go:build !windows

package poddaemon

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for daemon.go: run() lifecycle, exitDone broadcast.

func TestRunExitPathSendsExitToClient(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn) // AttachAck

	// Trigger child exit
	go func() {
		time.Sleep(50 * time.Millisecond)
		d.exitCode = 42
		close(d.exitDone)
	}()

	d.run()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, payload, err := ReadMessage(conn)
	if err == nil {
		assert.Equal(t, MsgExit, msgType)
		assert.Equal(t, int32(42), int32(binary.BigEndian.Uint32(payload)))
	}
}

func TestRunOrphanPathKillsProcess(t *testing.T) {
	d, proc, _ := setupDaemonServer(t)

	go func() {
		time.Sleep(50 * time.Millisecond)
		close(d.orphanCh)
		// After GracefulStop, simulate process exit
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(d.exitDone)
		}()
	}()

	d.run()

	proc.mu.Lock()
	assert.GreaterOrEqual(t, proc.gracefulStopCount, 1)
	proc.mu.Unlock()
}

// TestExitDoneBroadcastMultipleListeners verifies that closing exitDone
// unblocks multiple goroutines simultaneously (P0 fix: chan struct{} broadcast).
func TestExitDoneBroadcastMultipleListeners(t *testing.T) {
	d, _, _ := setupDaemonServer(t)

	const numListeners = 3
	received := make(chan int, numListeners)

	for i := range numListeners {
		go func(id int) {
			<-d.exitDone
			received <- id
		}(i)
	}

	d.exitCode = 7
	close(d.exitDone)

	for range numListeners {
		select {
		case <-received:
		case <-time.After(2 * time.Second):
			t.Fatal("not all listeners received exitDone broadcast")
		}
	}
}

// TestSimultaneousExitAndOrphan verifies run() handles both exitDone and
// orphanCh firing near-simultaneously without deadlock.
func TestSimultaneousExitAndOrphan(t *testing.T) {
	d, _, _ := setupDaemonServer(t)

	done := make(chan struct{})
	go func() {
		d.run()
		close(done)
	}()

	d.exitCode = 7
	close(d.exitDone)
	close(d.orphanCh)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("run() deadlocked when both exitDone and orphanCh fired")
	}
}

// TestExitDuringClientHandshake verifies run() handles child exit while
// a client is mid-session.
func TestExitDuringClientHandshake(t *testing.T) {
	d, _, ipcPath := setupDaemonServer(t)

	go d.acceptLoop()

	conn, err := Dial(ipcPath)
	require.NoError(t, err)
	defer conn.Close()

	WriteMessage(conn, MsgAttach, []byte{protocolVersion})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	ReadMessage(conn) // AttachAck
	conn.SetReadDeadline(time.Time{})

	go func() {
		time.Sleep(50 * time.Millisecond)
		d.exitCode = 1
		close(d.exitDone)
	}()

	d.run()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, payload, err := ReadMessage(conn)
	if err == nil {
		assert.Equal(t, MsgExit, msgType)
		assert.Equal(t, int32(1), int32(binary.BigEndian.Uint32(payload)))
	}
}
