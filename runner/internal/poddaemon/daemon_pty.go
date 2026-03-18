package poddaemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// attachAckPayload is the JSON payload for MsgAttachAck.
type attachAckPayload struct {
	PID   int  `json:"pid"`
	Cols  int  `json:"cols"`
	Rows  int  `json:"rows"`
	Alive bool `json:"alive"`
}

// protocolVersion is the current attach protocol version.
const protocolVersion = 1

// daemonPTY implements ptyProcess by communicating with a daemon over IPC.
// Close sends Detach but does NOT kill the daemon process.
type daemonPTY struct {
	conn net.Conn
	pid  int
	cols int
	rows int
	log  *slog.Logger

	// sizeMu protects cols/rows from concurrent Resize/GetSize access.
	sizeMu sync.RWMutex

	// Write serialization - only one goroutine may write at a time.
	writeMu sync.Mutex

	// Output channel from recvLoop (closed when recvLoop exits).
	// Exit code is delivered via exitCh, consumed only by Wait().
	outputCh chan []byte
	exitCh   chan int

	// Read buffering with deadline support
	readBuf    bytes.Buffer
	readMu     sync.Mutex
	deadlineMu sync.Mutex
	deadline   time.Time

	closeOnce sync.Once
	closedCh  chan struct{}
}

// connectDaemon dials the IPC socket, performs the Attach handshake,
// and returns a ready daemonPTY.
func connectDaemon(ipcPath string) (*daemonPTY, error) {
	conn, err := Dial(ipcPath)
	if err != nil {
		return nil, fmt.Errorf("dial daemon: %w", err)
	}

	// Send Attach message
	if err := WriteMessage(conn, MsgAttach, []byte{protocolVersion}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send attach: %w", err)
	}

	// Wait for AttachAck (with timeout)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	msgType, payload, err := ReadMessage(conn)
	conn.SetReadDeadline(time.Time{}) // clear deadline
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("read attach ack: %w", err)
	}
	if msgType != MsgAttachAck {
		conn.Close()
		return nil, fmt.Errorf("expected AttachAck (0x%02x), got 0x%02x", MsgAttachAck, msgType)
	}

	var ack attachAckPayload
	if err := json.Unmarshal(payload, &ack); err != nil {
		conn.Close()
		return nil, fmt.Errorf("unmarshal attach ack: %w", err)
	}

	return newDaemonPTY(conn, ack.PID, ack.Cols, ack.Rows), nil
}

// newDaemonPTY creates a daemonPTY after a successful handshake.
func newDaemonPTY(conn net.Conn, pid, cols, rows int) *daemonPTY {
	d := &daemonPTY{
		conn:     conn,
		pid:      pid,
		cols:     cols,
		rows:     rows,
		log:      slog.Default().With("component", "daemon-pty", "pid", pid),
		outputCh: make(chan []byte, 64),
		exitCh:   make(chan int, 1),
		closedCh: make(chan struct{}),
	}
	go d.recvLoop()
	return d
}
