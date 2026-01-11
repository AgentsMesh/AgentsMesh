package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub manages WebSocket connections
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Clients by pod
	podClients map[string]map[*Client]bool

	// Clients by channel
	channelClients map[int64]map[*Client]bool

	// Clients by organization (for events channel)
	orgClients map[int64]map[*Client]bool

	// Clients by user (for targeted notifications)
	userClients map[int64]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast to pod
	podBroadcast chan *PodMessage

	// Broadcast to channel
	channelBroadcast chan *ChannelMessage

	// Broadcast to organization
	orgBroadcast chan *OrgMessage

	// Send to specific user
	userSend chan *UserMessage

	mu sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:          make(map[*Client]bool),
		podClients:       make(map[string]map[*Client]bool),
		channelClients:   make(map[int64]map[*Client]bool),
		orgClients:       make(map[int64]map[*Client]bool),
		userClients:      make(map[int64]map[*Client]bool),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		podBroadcast:     make(chan *PodMessage, 256),
		channelBroadcast: make(chan *ChannelMessage, 256),
		orgBroadcast:     make(chan *OrgMessage, 256),
		userSend:         make(chan *UserMessage, 256),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case msg := <-h.podBroadcast:
			h.handlePodBroadcast(msg)

		case msg := <-h.channelBroadcast:
			h.handleChannelBroadcast(msg)

		case msg := <-h.orgBroadcast:
			h.handleOrgBroadcast(msg)

		case msg := <-h.userSend:
			h.handleUserSend(msg)
		}
	}
}

// handleRegister handles client registration
func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	if client.podKey != "" {
		if h.podClients[client.podKey] == nil {
			h.podClients[client.podKey] = make(map[*Client]bool)
		}
		h.podClients[client.podKey][client] = true
	}

	if client.channelID != 0 {
		if h.channelClients[client.channelID] == nil {
			h.channelClients[client.channelID] = make(map[*Client]bool)
		}
		h.channelClients[client.channelID][client] = true
	}

	// Register to org clients (for events channel)
	if client.isEvents && client.orgID != 0 {
		if h.orgClients[client.orgID] == nil {
			h.orgClients[client.orgID] = make(map[*Client]bool)
		}
		h.orgClients[client.orgID][client] = true
	}

	// Register to user clients (for targeted notifications)
	if client.isEvents && client.userID != 0 {
		if h.userClients[client.userID] == nil {
			h.userClients[client.userID] = make(map[*Client]bool)
		}
		h.userClients[client.userID][client] = true
	}
}

// handleUnregister handles client unregistration
func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	close(client.send)

	if client.podKey != "" {
		delete(h.podClients[client.podKey], client)
		if len(h.podClients[client.podKey]) == 0 {
			delete(h.podClients, client.podKey)
		}
	}

	if client.channelID != 0 {
		delete(h.channelClients[client.channelID], client)
		if len(h.channelClients[client.channelID]) == 0 {
			delete(h.channelClients, client.channelID)
		}
	}

	// Unregister from org clients
	if client.isEvents && client.orgID != 0 {
		delete(h.orgClients[client.orgID], client)
		if len(h.orgClients[client.orgID]) == 0 {
			delete(h.orgClients, client.orgID)
		}
	}

	// Unregister from user clients
	if client.isEvents && client.userID != 0 {
		delete(h.userClients[client.userID], client)
		if len(h.userClients[client.userID]) == 0 {
			delete(h.userClients, client.userID)
		}
	}
}

// handlePodBroadcast handles pod broadcast messages
func (h *Hub) handlePodBroadcast(msg *PodMessage) {
	h.mu.RLock()
	clients := h.podClients[msg.PodKey]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- msg.Message:
		default:
			h.unregister <- client
		}
	}
}

// handleChannelBroadcast handles channel broadcast messages
func (h *Hub) handleChannelBroadcast(msg *ChannelMessage) {
	h.mu.RLock()
	clients := h.channelClients[msg.ChannelID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- msg.Message:
		default:
			h.unregister <- client
		}
	}
}

// handleOrgBroadcast handles organization broadcast messages
func (h *Hub) handleOrgBroadcast(msg *OrgMessage) {
	h.mu.RLock()
	clients := h.orgClients[msg.OrgID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- msg.Message:
		default:
			h.unregister <- client
		}
	}
}

// handleUserSend handles user-targeted messages
func (h *Hub) handleUserSend(msg *UserMessage) {
	h.mu.RLock()
	clients := h.userClients[msg.UserID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- msg.Message:
		default:
			h.unregister <- client
		}
	}
}

// ========== Public Broadcast Methods ==========

// BroadcastToPod sends a message to all clients connected to a pod
func (h *Hub) BroadcastToPod(podKey string, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.podBroadcast <- &PodMessage{
		PodKey:  podKey,
		Message: data,
	}
}

// BroadcastToChannel sends a message to all clients subscribed to a channel
func (h *Hub) BroadcastToChannel(channelID int64, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.channelBroadcast <- &ChannelMessage{
		ChannelID: channelID,
		Message:   data,
	}
}

// BroadcastToOrg sends a message to all events channel clients in an organization
func (h *Hub) BroadcastToOrg(orgID int64, data []byte) {
	h.orgBroadcast <- &OrgMessage{
		OrgID:   orgID,
		Message: data,
	}
}

// BroadcastToOrgJSON sends a JSON message to all events channel clients in an organization
func (h *Hub) BroadcastToOrgJSON(orgID int64, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.BroadcastToOrg(orgID, data)
	return nil
}

// SendToUser sends a message to all clients of a specific user
func (h *Hub) SendToUser(userID int64, data []byte) {
	h.userSend <- &UserMessage{
		UserID:  userID,
		Message: data,
	}
}

// SendToUserJSON sends a JSON message to all clients of a specific user
func (h *Hub) SendToUserJSON(userID int64, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.SendToUser(userID, data)
	return nil
}

// ========== Query Methods ==========

// GetOrgClientCount returns the number of events channel clients in an organization
func (h *Hub) GetOrgClientCount(orgID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.orgClients[orgID])
}

// GetUserClientCount returns the number of clients for a specific user
func (h *Hub) GetUserClientCount(userID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userClients[userID])
}

// GetPodClientCount returns the number of clients connected to a pod
func (h *Hub) GetPodClientCount(podKey string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.podClients[podKey])
}

// ========== Registration Methods ==========

// Register registers a client with the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// ========== Deprecated: Use NewClient/NewEventsClient from client.go ==========
// These are kept for backward compatibility but moved to client.go

// NewClientDeprecated creates a new client (use NewClient from client.go)
func NewClientDeprecated(hub *Hub, conn *websocket.Conn, userID, orgID int64) *Client {
	return NewClient(hub, conn, userID, orgID)
}
