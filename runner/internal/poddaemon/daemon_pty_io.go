package poddaemon

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// Read reads output data from the daemon, supporting deadline-based timeout.
// Terminal uses 100ms polling with SetReadDeadline.
func (d *daemonPTY) Read(p []byte) (int, error) {
	d.readMu.Lock()
	defer d.readMu.Unlock()

	// Drain buffered data first
	if d.readBuf.Len() > 0 {
		return d.readBuf.Read(p)
	}

	d.deadlineMu.Lock()
	deadline := d.deadline
	d.deadlineMu.Unlock()

	var timerCh <-chan time.Time
	var timer *time.Timer

	if !deadline.IsZero() {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, os.ErrDeadlineExceeded
		}
		timer = time.NewTimer(remaining)
		defer timer.Stop()
		timerCh = timer.C
	}

	select {
	case data, ok := <-d.outputCh:
		if !ok {
			// recvLoop exited (MsgExit received or connection error).
			// All buffered output has been drained before this fires.
			return 0, io.EOF
		}
		// Buffer data and read what fits
		d.readBuf.Write(data)
		return d.readBuf.Read(p)
	case <-timerCh:
		return 0, os.ErrDeadlineExceeded
	case <-d.closedCh:
		return 0, io.EOF
	}
}

// Write sends input data to the daemon.
func (d *daemonPTY) Write(data []byte) (int, error) {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	if err := WriteMessage(d.conn, MsgInput, data); err != nil {
		return 0, fmt.Errorf("write input: %w", err)
	}
	return len(data), nil
}

// Close sends a Detach message and closes the connection.
// The daemon process continues running.
func (d *daemonPTY) Close() error {
	var err error
	d.closeOnce.Do(func() {
		close(d.closedCh)
		// Best-effort detach notification. Use a short deadline so we don't
		// block if the pipe/socket is congested or recvLoop holds the read side.
		d.writeMu.Lock()
		d.conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		_ = WriteMessage(d.conn, MsgDetach, nil)
		d.writeMu.Unlock()
		err = d.conn.Close()
	})
	return err
}

// Resize sends a resize request to the daemon.
func (d *daemonPTY) Resize(cols, rows int) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[0:2], uint16(cols))
	binary.BigEndian.PutUint16(payload[2:4], uint16(rows))
	if err := WriteMessage(d.conn, MsgResize, payload); err != nil {
		return err
	}
	d.sizeMu.Lock()
	d.cols = cols
	d.rows = rows
	d.sizeMu.Unlock()
	return nil
}

// GetSize returns the last known terminal size.
func (d *daemonPTY) GetSize() (int, int, error) {
	d.sizeMu.RLock()
	defer d.sizeMu.RUnlock()
	return d.cols, d.rows, nil
}

// Pid returns the daemon child process PID.
func (d *daemonPTY) Pid() int { return d.pid }

// SetReadDeadline sets a deadline for Read operations.
func (d *daemonPTY) SetReadDeadline(t time.Time) error {
	d.deadlineMu.Lock()
	d.deadline = t
	d.deadlineMu.Unlock()
	return nil
}

// Wait blocks until the daemon child process exits.
// exitCh is only consumed here (Read uses outputCh closure for EOF).
func (d *daemonPTY) Wait() (int, error) {
	select {
	case code := <-d.exitCh:
		return code, nil
	case <-d.closedCh:
		return -1, fmt.Errorf("connection closed")
	}
}

// Kill sends a kill command to the daemon.
func (d *daemonPTY) Kill() error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	return WriteMessage(d.conn, MsgKill, nil)
}

// GracefulStop sends a graceful stop signal to the daemon.
func (d *daemonPTY) GracefulStop() error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	return WriteMessage(d.conn, MsgGracefulStop, nil)
}

// recvLoop reads messages from the daemon and dispatches them.
func (d *daemonPTY) recvLoop() {
	defer func() {
		close(d.outputCh)
	}()

	for {
		msgType, payload, err := ReadMessage(d.conn)
		if err != nil {
			select {
			case <-d.closedCh:
				return // Expected on close
			default:
				d.log.Debug("daemon recvLoop error", "error", err)
				return
			}
		}

		switch msgType {
		case MsgOutput:
			select {
			case d.outputCh <- payload:
			case <-d.closedCh:
				return
			}

		case MsgExit:
			if len(payload) >= 4 {
				code := int(int32(binary.BigEndian.Uint32(payload)))
				select {
				case d.exitCh <- code:
				default:
				}
			}
			return

		case MsgPong:
			// Heartbeat response - logged at debug level
			d.log.Debug("daemon pong received")

		default:
			d.log.Warn("daemon recvLoop: unknown message type", "type", msgType)
		}
	}
}
