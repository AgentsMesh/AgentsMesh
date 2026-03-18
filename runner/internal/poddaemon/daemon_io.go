package poddaemon

import (
	"encoding/binary"
	"encoding/json"
	"net"
)

// acceptLoop accepts IPC connections and handles them in goroutines.
// New connections automatically disconnect the previous client.
func (d *daemonServer) acceptLoop() {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			d.log.Debug("accept error (listener closed?)", "error", err)
			return
		}
		go d.handleClient(conn)
	}
}

// handleClient handles a single client connection.
func (d *daemonServer) handleClient(conn net.Conn) {
	log := d.log

	// Read Attach message
	msgType, payload, err := ReadMessage(conn)
	if err != nil {
		log.Error("read attach failed", "error", err)
		conn.Close()
		return
	}
	if msgType != MsgAttach {
		log.Error("expected Attach message", "got", msgType)
		conn.Close()
		return
	}

	version := byte(0)
	if len(payload) > 0 {
		version = payload[0]
	}
	log.Info("client attached", "version", version)

	// Send AttachAck — read terminal size under clientMu to avoid data race
	// with concurrent Resize from a previous client's readClientCommands.
	d.clientMu.Lock()
	ack := attachAckPayload{
		PID:   d.proc.Pid(),
		Cols:  d.state.Cols,
		Rows:  d.state.Rows,
		Alive: true,
	}
	d.clientMu.Unlock()
	ackData, _ := json.Marshal(ack)
	if err := WriteMessage(conn, MsgAttachAck, ackData); err != nil {
		log.Error("write attach ack failed", "error", err)
		conn.Close()
		return
	}

	// Set as current client
	d.clientMu.Lock()
	if d.client != nil {
		// Disconnect previous client
		d.client.Close()
	}
	d.client = conn
	d.clientMu.Unlock()

	// Read commands from client
	d.readClientCommands(conn)
}

// readClientCommands reads and dispatches messages from a client.
func (d *daemonServer) readClientCommands(conn net.Conn) {
	log := d.log

	for {
		msgType, payload, err := ReadMessage(conn)
		if err != nil {
			log.Debug("client disconnected", "error", err)
			d.removeClient(conn)
			return
		}

		switch msgType {
		case MsgInput:
			if _, err := d.proc.Write(payload); err != nil {
				log.Error("write to pty failed", "error", err)
			}

		case MsgResize:
			if len(payload) >= 4 {
				cols := int(binary.BigEndian.Uint16(payload[0:2]))
				rows := int(binary.BigEndian.Uint16(payload[2:4]))
				if cols <= 0 || cols > 1000 || rows <= 0 || rows > 1000 {
					log.Warn("invalid resize dimensions, ignoring", "cols", cols, "rows", rows)
					continue
				}
				if err := d.proc.Resize(cols, rows); err != nil {
					log.Error("resize failed", "error", err)
				}
				d.clientMu.Lock()
				d.state.Cols = cols
				d.state.Rows = rows
				d.clientMu.Unlock()
			}

		case MsgGracefulStop:
			log.Info("graceful stop requested")
			if err := d.proc.GracefulStop(); err != nil {
				log.Error("graceful stop failed", "error", err)
			}

		case MsgKill:
			log.Info("kill requested")
			if err := d.proc.Kill(); err != nil {
				log.Error("kill failed", "error", err)
			}

		case MsgDetach:
			log.Info("client detached")
			d.removeClient(conn)
			return

		case MsgPing:
			d.connWriteMu.Lock()
			_ = WriteMessage(conn, MsgPong, nil)
			d.connWriteMu.Unlock()

		default:
			log.Warn("unknown message from client", "type", msgType)
		}
	}
}

// removeClient clears the current client if it matches conn.
func (d *daemonServer) removeClient(conn net.Conn) {
	d.clientMu.Lock()
	if d.client == conn {
		d.client = nil
	}
	d.clientMu.Unlock()
	conn.Close()
}

// ptyReader reads from the PTY and forwards output to the current client.
//
// Design: clientMu is held only to snapshot the client pointer, then released
// before the (potentially slow) network write. This prevents data-plane
// backpressure from blocking control-plane operations (Pong, Exit notification)
// that also need the client pointer.
//
// Write serialization is handled by connWriteMu. If the client pointer becomes
// stale (replaced by a new connection), WriteMessage returns an error on the
// closed connection, which is safely ignored.
func (d *daemonServer) ptyReader() {
	buf := make([]byte, 4096)
	for {
		n, err := d.proc.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			d.clientMu.Lock()
			client := d.client
			d.clientMu.Unlock()

			if client != nil {
				d.connWriteMu.Lock()
				if writeErr := WriteMessage(client, MsgOutput, data); writeErr != nil {
					d.log.Debug("write output to client failed", "error", writeErr)
				}
				d.connWriteMu.Unlock()
			}
		}
		if err != nil {
			d.log.Debug("pty read ended", "error", err)
			return
		}
	}
}
