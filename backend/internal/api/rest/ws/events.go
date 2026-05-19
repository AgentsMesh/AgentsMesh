package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	gorillaWs "github.com/gorilla/websocket"
)

var upgrader = gorillaWs.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type EventsHandler struct {
	hub    *websocket.Hub
	logger *slog.Logger
}

func NewEventsHandler(hub *websocket.Hub) *EventsHandler {
	return &EventsHandler{
		hub:    hub,
		logger: slog.Default().With("component", "events_ws"),
	}
}

type EventsClientMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type EventsServerMessage struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

func (h *EventsHandler) HandleEvents(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "unauthorized")
		return
	}
	userClaims := claims.(*middleware.Claims)

	tenant, exists := c.Get("tenant")
	if !exists {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "missing organization context")
		return
	}
	tenantCtx := tenant.(*middleware.TenantContext)

	h.logger.Debug("events websocket connection request",
		"user_id", userClaims.UserID,
		"org_id", tenantCtx.OrganizationID,
		"org_slug", tenantCtx.OrganizationSlug,
	)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", "error", err)
		return
	}

	h.logger.Info("events websocket connection established",
		"user_id", userClaims.UserID,
		"org_id", tenantCtx.OrganizationID,
	)

	client := websocket.NewEventsClient(h.hub, conn, userClaims.UserID, tenantCtx.OrganizationID)

	h.hub.Register(client)

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

	go client.WritePump()
	go client.ReadPump(func(c *websocket.Client, msg *websocket.Message) {
		h.handleClientMessage(c, msg)
	})
}

func (h *EventsHandler) handleClientMessage(client *websocket.Client, msg *websocket.Message) {
	switch msg.Type {
	case websocket.MessageTypePing:
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
