package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	runner "github.com/anthropics/agentsmesh/backend/internal/service/runner"
	gorillaws "github.com/gorilla/websocket"
	"github.com/gin-gonic/gin"
)

// RunnerHandler handles runner WebSocket connections
type RunnerHandler struct {
	runnerService     *runner.Service
	connectionManager *runner.ConnectionManager
	logger            *slog.Logger
}

// NewRunnerHandler creates a new runner WebSocket handler
func NewRunnerHandler(
	runnerService *runner.Service,
	connectionManager *runner.ConnectionManager,
	logger *slog.Logger,
) *RunnerHandler {
	h := &RunnerHandler{
		runnerService:     runnerService,
		connectionManager: connectionManager,
		logger:            logger,
	}

	// Set up initialization callback to persist available agents
	connectionManager.SetInitializedCallback(h.onRunnerInitialized)

	return h
}

// onRunnerInitialized is called when a runner completes the initialization handshake
func (h *RunnerHandler) onRunnerInitialized(runnerID int64, availableAgents []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.runnerService.UpdateAvailableAgents(ctx, runnerID, availableAgents); err != nil {
		h.logger.Error("failed to update runner available agents",
			"runner_id", runnerID,
			"available_agents", availableAgents,
			"error", err)
		return
	}

	h.logger.Info("runner initialization completed, available agents updated",
		"runner_id", runnerID,
		"available_agents", availableAgents)
}

// HandleRunnerWS handles WebSocket connection from runner
// GET /api/v1/runners/ws?node_id=xxx&token=xxx
func (h *RunnerHandler) HandleRunnerWS(c *gin.Context) {
	nodeID := c.Query("node_id")
	authToken := c.Query("token")

	// Also check Authorization header
	if authToken == "" {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			authToken = authHeader[7:]
		}
	}

	// Also check X-Runner-ID header
	if nodeID == "" {
		nodeID = c.GetHeader("X-Runner-ID")
	}

	if nodeID == "" || authToken == "" {
		h.logger.Warn("runner ws connection missing credentials",
			"node_id", nodeID,
			"has_token", authToken != "")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing node_id or token"})
		return
	}

	// Validate runner authentication
	r, err := h.runnerService.ValidateRunnerAuth(c.Request.Context(), nodeID, authToken)
	if err != nil {
		h.logger.Warn("runner ws auth failed",
			"node_id", nodeID,
			"error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authentication"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade runner connection",
			"runner_id", r.ID,
			"error", err)
		return
	}

	h.logger.Info("runner ws connected",
		"runner_id", r.ID,
		"node_id", nodeID)

	// Add to connection manager
	rc := h.connectionManager.AddConnection(r.ID, conn)

	// Update runner status to online
	if err := h.runnerService.SetRunnerStatus(c.Request.Context(), r.ID, "online"); err != nil {
		h.logger.Warn("failed to update runner status",
			"runner_id", r.ID,
			"error", err)
	}

	// Start write pump
	go rc.WritePump()

	// Run read loop (blocking)
	h.readLoop(r.ID, conn)

	// Cleanup on disconnect
	h.connectionManager.RemoveConnection(r.ID)

	// Update runner status to offline
	if err := h.runnerService.SetRunnerStatus(c.Request.Context(), r.ID, "offline"); err != nil {
		h.logger.Warn("failed to update runner status on disconnect",
			"runner_id", r.ID,
			"error", err)
	}
}

// readLoop reads messages from the runner WebSocket
func (h *RunnerHandler) readLoop(runnerID int64, conn *gorillaws.Conn) {
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			// Log all close errors for debugging
			h.logger.Warn("runner ws read error",
				"runner_id", runnerID,
				"error", err,
				"is_unexpected_close", gorillaws.IsUnexpectedCloseError(err, gorillaws.CloseGoingAway, gorillaws.CloseNormalClosure))
			return
		}

		// Reset read deadline on any message
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))

		// Handle the message
		h.connectionManager.HandleMessage(runnerID, msgType, data)
	}
}

// SendToRunner sends a message to a specific runner
func (h *RunnerHandler) SendToRunner(runnerID int64, msgType string, data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.connectionManager.SendMessage(nil, runnerID, &runner.RunnerMessage{
		Type:      msgType,
		Data:      dataBytes,
		Timestamp: time.Now().UnixMilli(),
	})
}
