package admin

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/relay"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// RelayHandler handles relay management requests
type RelayHandler struct {
	adminService *adminservice.Service
	relayManager *relay.Manager
}

// NewRelayHandler creates a new relay handler
func NewRelayHandler(
	adminSvc *adminservice.Service,
	relayMgr *relay.Manager,
) *RelayHandler {
	return &RelayHandler{
		adminService: adminSvc,
		relayManager: relayMgr,
	}
}

// RegisterRoutes registers relay management routes
func (h *RelayHandler) RegisterRoutes(rg *gin.RouterGroup) {
	relaysGroup := rg.Group("/relays")
	{
		relaysGroup.GET("", h.ListRelays)
		relaysGroup.GET("/stats", h.GetStats)
		relaysGroup.GET("/:id", h.GetRelay)
		relaysGroup.DELETE("/:id", h.ForceUnregister)
	}
}

// logAction is a helper method for audit logging
func (h *RelayHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

// ListRelays returns all registered relays
func (h *RelayHandler) ListRelays(c *gin.Context) {
	if h.relayManager == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Relay manager not available")
		return
	}

	relays := h.relayManager.GetRelays()

	c.JSON(http.StatusOK, gin.H{
		"data":  relays,
		"total": len(relays),
	})
}

// GetStats returns relay statistics
func (h *RelayHandler) GetStats(c *gin.Context) {
	if h.relayManager == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Relay manager not available")
		return
	}

	stats := h.relayManager.GetStats()

	c.JSON(http.StatusOK, stats)
}

// GetRelay returns a specific relay by ID
func (h *RelayHandler) GetRelay(c *gin.Context) {
	if h.relayManager == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Relay manager not available")
		return
	}

	relayID := c.Param("id")
	if relayID == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "Relay ID is required")
		return
	}

	relayInfo := h.relayManager.GetRelayByID(relayID)
	if relayInfo == nil {
		apierr.ResourceNotFound(c, "Relay not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"relay": relayInfo,
	})
}

// ForceUnregister removes a relay
func (h *RelayHandler) ForceUnregister(c *gin.Context) {
	if h.relayManager == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Relay manager not available")
		return
	}

	relayID := c.Param("id")
	if relayID == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "Relay ID is required")
		return
	}

	// Get relay info before deletion for logging
	relayInfo := h.relayManager.GetRelayByID(relayID)
	if relayInfo == nil {
		apierr.ResourceNotFound(c, "Relay not found")
		return
	}

	// Force unregister
	h.relayManager.ForceUnregister(relayID)

	// Log admin action
	h.logAction(c, admin.AuditActionDelete, admin.TargetType("relay"), 0,
		gin.H{"relay_id": relayID, "url": relayInfo.URL},
		nil)

	c.JSON(http.StatusOK, gin.H{
		"status":   "unregistered",
		"relay_id": relayID,
	})
}
