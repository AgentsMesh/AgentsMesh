package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// EventsHandler handles events WebSocket connections
type EventsHandler struct {
	hub    *websocket.Hub
	logger *slog.Logger
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(hub *websocket.Hub) *EventsHandler {
	return &EventsHandler{
		hub:    hub,
		logger: slog.Default().With("component", "events_ws"),
	}
}

// EventsClientMessage represents a message from the client
type EventsClientMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// EventsServerMessage represents a message to the client
type EventsServerMessage struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// HandleEvents handles WebSocket connection for events channel
func (h *EventsHandler) HandleEvents(c *gin.Context) {
	// Get user from context
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userClaims := claims.(*middleware.Claims)

	// Get tenant context
	tenant, exists := c.Get("tenant")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing organization context"})
		return
	}
	tenantCtx := tenant.(*middleware.TenantContext)

	h.logger.Debug("events websocket connection request",
		"user_id", userClaims.UserID,
		"org_id", tenantCtx.OrganizationID,
		"org_slug", tenantCtx.OrganizationSlug,
	)

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", "error", err)
		return
	}

	h.logger.Info("events websocket connection established",
		"user_id", userClaims.UserID,
		"org_id", tenantCtx.OrganizationID,
	)

	// Create events client
	client := websocket.NewEventsClient(h.hub, conn, userClaims.UserID, tenantCtx.OrganizationID)

	// Register client with hub
	h.hub.Register(client)

	// Send connected message
	connectedMsg := &EventsServerMessage{
		Type:      "connected",
		Timestamp: time.Now().UnixMilli(),
	}
	if err := client.Send(&websocket.Message{
		Type:      websocket.MessageType("connected"),
		Timestamp: time.Now().UnixMilli(),
	}); err != nil {
		h.logger.Error("failed to send connected message", "error", err)
	}
	_ = connectedMsg // avoid unused variable warning

	// Start read/write pumps
	go client.WritePump()
	go client.ReadPump(func(c *websocket.Client, msg *websocket.Message) {
		h.handleClientMessage(c, msg)
	})
}

// handleClientMessage handles messages from the client
func (h *EventsHandler) handleClientMessage(client *websocket.Client, msg *websocket.Message) {
	switch msg.Type {
	case websocket.MessageTypePing:
		// Respond with pong
		pongMsg := &websocket.Message{
			Type:      websocket.MessageTypePong,
			Timestamp: time.Now().UnixMilli(),
		}
		if err := client.Send(pongMsg); err != nil {
			h.logger.Error("failed to send pong", "error", err)
		}

	default:
		h.logger.Debug("received unknown message type",
			"type", msg.Type,
			"user_id", client.UserID(),
			"org_id", client.OrgID(),
		)
	}
}
