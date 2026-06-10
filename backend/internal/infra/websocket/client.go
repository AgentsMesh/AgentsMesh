package websocket

// Client is a connect events subscriber. The hub fans org/user broadcasts into
// its send channel; the Connect server-stream handler (backend/internal/api/
// connect/events) drains Outbound() and forwards the bytes to the stream.
type Client struct {
	hub    *Hub
	send   chan []byte
	userID int64
	orgID  int64
}

func (c *Client) UserID() int64 {
	return c.userID
}

func (c *Client) OrgID() int64 {
	return c.orgID
}

// NewConnectEventsClient creates an events Client without a WebSocket
// connection — for the Connect-RPC server-stream handler. The caller drains
// Outbound() and forwards the bytes to the Connect stream.
func NewConnectEventsClient(hub *Hub, userID, orgID int64) *Client {
	return &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		userID: userID,
		orgID:  orgID,
	}
}

// Outbound exposes the hub→client byte channel so the Connect server stream
// can drain it. Returns a receive-only channel — the hub remains the sole
// writer.
func (c *Client) Outbound() <-chan []byte {
	return c.send
}
