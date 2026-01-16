package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
}

// TerminalHandler handles terminal WebSocket connections
type TerminalHandler struct {
	hub            *websocket.Hub
	podService     *agentpod.PodService
	terminalRouter *runner.TerminalRouter
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(hub *websocket.Hub, podService *agentpod.PodService) *TerminalHandler {
	return &TerminalHandler{
		hub:        hub,
		podService: podService,
	}
}

// SetTerminalRouter sets the terminal router for routing terminal data
func (h *TerminalHandler) SetTerminalRouter(tr *runner.TerminalRouter) {
	h.terminalRouter = tr
}

// HandleTerminal handles WebSocket connection for terminal
func (h *TerminalHandler) HandleTerminal(c *gin.Context) {
	podKey := c.Param("pod_key")

	log.Printf("[terminal_ws] HandleTerminal called for pod: %s", podKey)

	// Get user from context
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	_ = claims.(*middleware.Claims)

	// Get tenant context
	tenant, exists := c.Get("tenant")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing organization context"})
		return
	}
	tenantCtx := tenant.(*middleware.TenantContext)

	// Verify pod exists and belongs to the organization
	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		log.Printf("[terminal_ws] Pod not found: %s, error: %v", podKey, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "pod not found"})
		return
	}

	if pod.OrganizationID != tenantCtx.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Check if terminal router is available
	if h.terminalRouter == nil {
		log.Printf("[terminal_ws] Terminal router not available")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "terminal router not available"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[terminal_ws] Failed to upgrade connection: %v", err)
		return
	}

	log.Printf("[terminal_ws] WebSocket connection upgraded for pod: %s", podKey)

	// Connect client to terminal router
	client, err := h.terminalRouter.ConnectClient(podKey, conn)
	if err != nil {
		log.Printf("[terminal_ws] Failed to connect client: %v", err)
		conn.Close()
		return
	}

	log.Printf("[terminal_ws] Client connected to terminal router for pod: %s", podKey)

	// Start write pump to send data to client
	go h.writePump(client)

	// Start read loop to receive data from client
	h.readLoop(podKey, client, conn)

	// Disconnect client when done
	h.terminalRouter.DisconnectClient(client)
	log.Printf("[terminal_ws] Client disconnected from pod: %s", podKey)
}

// writePump sends data from the terminal router to the WebSocket client
func (h *TerminalHandler) writePump(client *runner.TerminalClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				// Channel closed
				client.Conn.WriteMessage(gorillaws.CloseMessage, []byte{})
				return
			}

			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			// Use TextMessage for JSON control messages, BinaryMessage for terminal output
			msgType := gorillaws.BinaryMessage
			if msg.IsJSON {
				msgType = gorillaws.TextMessage
			}

			if err := client.Conn.WriteMessage(msgType, msg.Data); err != nil {
				log.Printf("[terminal_ws] Failed to write to client: %v", err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readLoop reads data from the WebSocket client and routes it to the runner
func (h *TerminalHandler) readLoop(podKey string, client *runner.TerminalClient, conn *gorillaws.Conn) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if gorillaws.IsUnexpectedCloseError(err, gorillaws.CloseGoingAway, gorillaws.CloseNormalClosure) {
				log.Printf("[terminal_ws] Read error for pod %s: %v", podKey, err)
			}
			return
		}

		// Reset read deadline
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		if msgType == gorillaws.TextMessage {
			// Parse JSON message
			var msg struct {
				Type string `json:"type"`
				Data string `json:"data"`
				Rows int    `json:"rows"`
				Cols int    `json:"cols"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("[terminal_ws] Failed to parse message: %v", err)
				continue
			}

			switch msg.Type {
			case "input":
				// Route input to runner
				if err := h.terminalRouter.RouteInput(podKey, []byte(msg.Data)); err != nil {
					log.Printf("[terminal_ws] Failed to route input: %v", err)
				}
			case "resize":
				// Route resize to runner
				if err := h.terminalRouter.RouteResize(podKey, msg.Cols, msg.Rows); err != nil {
					log.Printf("[terminal_ws] Failed to route resize: %v", err)
				}
			}
		} else if msgType == gorillaws.BinaryMessage {
			// Binary data is terminal input
			if err := h.terminalRouter.RouteInput(podKey, data); err != nil {
				log.Printf("[terminal_ws] Failed to route binary input: %v", err)
			}
		}
	}
}

