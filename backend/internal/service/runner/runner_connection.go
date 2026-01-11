package runner

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// RunnerConnection represents an active connection to a runner
type RunnerConnection struct {
	RunnerID int64
	Conn     *websocket.Conn
	Send     chan []byte
	LastPing time.Time
	mu       sync.Mutex
}

// SendMessage sends a message on the connection
func (rc *RunnerConnection) SendMessage(msg *RunnerMessage) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.Conn == nil {
		return ErrConnectionClosed
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case rc.Send <- data:
		return nil
	default:
		return errors.New("send buffer full")
	}
}

// Close closes the connection
func (rc *RunnerConnection) Close() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.Conn != nil {
		rc.Conn.Close()
		rc.Conn = nil
	}

	// Close send channel safely
	select {
	case <-rc.Send:
	default:
		close(rc.Send)
	}
}

// WritePump pumps messages from the send channel to the WebSocket
func (rc *RunnerConnection) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		rc.Close()
	}()

	for {
		select {
		case message, ok := <-rc.Send:
			rc.mu.Lock()
			conn := rc.Conn
			rc.mu.Unlock()

			if conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			rc.mu.Lock()
			conn := rc.Conn
			rc.mu.Unlock()

			if conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
