package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    int64
	orgID     int64
	podKey    string // Empty if not connected to a pod
	channelID int64  // Non-zero if subscribed to a channel
	isEvents  bool   // True if this is an events channel client
	mu        sync.Mutex
}

// UserID returns the user ID of this client
func (c *Client) UserID() int64 {
	return c.userID
}

// OrgID returns the organization ID of this client
func (c *Client) OrgID() int64 {
	return c.orgID
}

// NewClient creates a new client
func NewClient(hub *Hub, conn *websocket.Conn, userID, orgID int64) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		orgID:  orgID,
	}
}

// NewEventsClient creates a new events channel client
func NewEventsClient(hub *Hub, conn *websocket.Conn, userID, orgID int64) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		orgID:    orgID,
		isEvents: true,
	}
}

// SetPod sets the pod for this client
func (c *Client) SetPod(podKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove from old pod
	if c.podKey != "" {
		c.hub.mu.Lock()
		delete(c.hub.podClients[c.podKey], c)
		if len(c.hub.podClients[c.podKey]) == 0 {
			delete(c.hub.podClients, c.podKey)
		}
		c.hub.mu.Unlock()
	}

	c.podKey = podKey

	// Add to new pod
	if podKey != "" {
		c.hub.mu.Lock()
		if c.hub.podClients[podKey] == nil {
			c.hub.podClients[podKey] = make(map[*Client]bool)
		}
		c.hub.podClients[podKey][c] = true
		c.hub.mu.Unlock()
	}
}

// SetChannel sets the channel subscription for this client
func (c *Client) SetChannel(channelID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove from old channel
	if c.channelID != 0 {
		c.hub.mu.Lock()
		delete(c.hub.channelClients[c.channelID], c)
		if len(c.hub.channelClients[c.channelID]) == 0 {
			delete(c.hub.channelClients, c.channelID)
		}
		c.hub.mu.Unlock()
	}

	c.channelID = channelID

	// Add to new channel
	if channelID != 0 {
		c.hub.mu.Lock()
		if c.hub.channelClients[channelID] == nil {
			c.hub.channelClients[channelID] = make(map[*Client]bool)
		}
		c.hub.channelClients[channelID][c] = true
		c.hub.mu.Unlock()
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump(onMessage func(*Client, *Message)) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if onMessage != nil {
			onMessage(c, &msg)
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send sends a message to the client
func (c *Client) Send(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		return nil
	}
}
