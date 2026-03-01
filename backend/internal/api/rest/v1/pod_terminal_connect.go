package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/relay"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/runner"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
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
// Note: SessionID has been removed - channels are now identified by PodKey only
type TerminalConnectResponse struct {
	RelayURL string `json:"relay_url"`
	Token    string `json:"token"`
	PodKey   string `json:"pod_key"`
}

// GetTerminalConnection returns Relay connection info for a pod
// GET /api/v1/orgs/:slug/pods/:key/terminal/connect
//
// The channel is identified by PodKey (not session ID):
// - Multiple browsers can subscribe to the same pod's channel
// - Runner maintains a single connection per pod
// - No new session ID is generated per request
func (h *TerminalConnectHandler) GetTerminalConnection(c *gin.Context) {
	podKey := c.Param("key")

	// Check if relay is available
	if h.relayManager == nil || !h.relayManager.HasHealthyRelays() {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal relay service is not available")
		return
	}

	// Get pod info
	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	// Check organization access
	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	// Check pod is active
	if !pod.IsActive() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not active")
		return
	}

	// Get user ID
	userID := middleware.GetUserID(c)
	if userID == 0 {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "User not found")
		return
	}

	// Select relay for this pod using org-affinity based selection
	// The runner region is not used anymore, org affinity provides stable relay selection
	relayInfo := h.relayManager.SelectRelayForPod(tenant.OrganizationSlug)
	if relayInfo == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "No healthy relay available")
		return
	}

	// Always notify runner to connect to relay
	// Runner handles idempotency - if already connected to same relay, it just updates the token
	// Use internal URL for runner (Docker network) if available
	if h.commandSender != nil && pod.RunnerID > 0 {
		// Generate runner token for authentication
		// userID=0 indicates this is a runner token (not a browser token)
		runnerToken, err := h.tokenGenerator.GenerateToken(
			podKey,
			pod.RunnerID,
			0, // userID=0 for runner token
			tenant.OrganizationID,
			time.Hour,
		)
		if err != nil {
			apierr.InternalError(c, "Failed to generate runner token")
			return
		}

		if err := h.commandSender.SendSubscribeTerminal(
			c.Request.Context(),
			pod.RunnerID,
			podKey,
			relayInfo.GetRunnerURL(),
			runnerToken,
			true, // include snapshot
			1000, // snapshot history lines
		); err != nil {
			// Log but don't fail - runner might connect later
			c.Error(err)
		}
	}

	// Generate token for browser
	runnerID := pod.RunnerID

	token, err := h.tokenGenerator.GenerateToken(
		podKey,
		runnerID,
		userID,
		tenant.OrganizationID,
		time.Hour,
	)
	if err != nil {
		apierr.InternalError(c, "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, TerminalConnectResponse{
		RelayURL: relayInfo.URL,
		Token:    token,
		PodKey:   podKey,
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
