package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// TerminalConnectHandler handles terminal connection requests via Relay
type TerminalConnectHandler struct {
	podService     PodServiceForHandler
	relayManager   *relay.Manager
	tokenGenerator *relay.TokenGenerator
	commandSender  runner.RunnerCommandSender
}

// NewTerminalConnectHandler creates a new terminal connect handler
func NewTerminalConnectHandler(
	podService PodServiceForHandler,
	relayManager *relay.Manager,
	tokenGenerator *relay.TokenGenerator,
	commandSender runner.RunnerCommandSender,
) *TerminalConnectHandler {
	return &TerminalConnectHandler{
		podService:     podService,
		relayManager:   relayManager,
		tokenGenerator: tokenGenerator,
		commandSender:  commandSender,
	}
}

// TerminalConnectResponse is the response for terminal connect request
type TerminalConnectResponse struct {
	RelayURL  string `json:"relay_url"`
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	PodKey    string `json:"pod_key"`
}

// GetTerminalConnection returns Relay connection info for a pod
// GET /api/v1/orgs/:slug/pods/:key/terminal/connect
func (h *TerminalConnectHandler) GetTerminalConnection(c *gin.Context) {
	podKey := c.Param("key")

	// Check if relay is available
	if h.relayManager == nil || !h.relayManager.HasHealthyRelays() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "relay_not_available",
			"message": "Terminal relay service is not available",
		})
		return
	}

	// Get pod info
	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pod not found"})
		return
	}

	// Check organization access
	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check pod is active
	if !pod.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pod is not active"})
		return
	}

	// Get user ID
	userID := middleware.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Select relay for this pod using org-affinity based selection
	// The runner region is not used anymore, org affinity provides stable relay selection
	relayInfo := h.relayManager.SelectRelayForPod(tenant.OrganizationSlug)
	if relayInfo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "no_relay_available",
			"message": "No healthy relay available",
		})
		return
	}

	// Generate new session ID for each request
	sessionID := uuid.New().String()

	// Always notify runner to connect to relay (runner handles idempotency)
	// Use internal URL for runner (Docker network) if available
	if h.commandSender != nil && pod.RunnerID > 0 {
		// Generate runner token for authentication
		// userID=0 indicates this is a runner token (not a browser token)
		runnerToken, err := h.tokenGenerator.GenerateToken(
			podKey,
			sessionID,
			pod.RunnerID,
			0, // userID=0 for runner token
			tenant.OrganizationID,
			time.Hour,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate runner token"})
			return
		}

		if err := h.commandSender.SendSubscribeTerminal(
			c.Request.Context(),
			pod.RunnerID,
			podKey,
			relayInfo.GetRunnerURL(),
			sessionID,
			runnerToken,
			true,  // include snapshot
			1000,  // snapshot history lines
		); err != nil {
			// Log but don't fail - runner might connect later
			c.Error(err)
		}
	}

	// Generate token for browser
	runnerID := pod.RunnerID

	token, err := h.tokenGenerator.GenerateToken(
		podKey,
		sessionID,
		runnerID,
		userID,
		tenant.OrganizationID,
		time.Hour,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, TerminalConnectResponse{
		RelayURL:  relayInfo.URL,
		Token:     token,
		SessionID: sessionID,
		PodKey:    podKey,
	})
}

// RegisterTerminalConnectRoutes registers terminal connect routes
func RegisterTerminalConnectRoutes(
	router *gin.RouterGroup,
	podService PodServiceForHandler,
	relayManager *relay.Manager,
	tokenGenerator *relay.TokenGenerator,
	commandSender runner.RunnerCommandSender,
) {
	handler := NewTerminalConnectHandler(podService, relayManager, tokenGenerator, commandSender)
	router.GET("/pods/:key/terminal/connect", handler.GetTerminalConnection)
}
