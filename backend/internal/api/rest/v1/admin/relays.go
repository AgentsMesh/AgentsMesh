package admin

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type RelayHandler struct {
	adminService *adminservice.Service
	relayManager *relay.Manager
}

func NewRelayHandler(
	adminSvc *adminservice.Service,
	relayMgr *relay.Manager,
) *RelayHandler {
	return &RelayHandler{
		adminService: adminSvc,
		relayManager: relayMgr,
	}
}

func (h *RelayHandler) RegisterRoutes(rg *gin.RouterGroup) {
	relaysGroup := rg.Group("/relays")
	{
		relaysGroup.GET("", h.ListRelays)
		relaysGroup.GET("/stats", h.GetStats)
		relaysGroup.GET("/:id", h.GetRelay)
		relaysGroup.DELETE("/:id", h.ForceUnregister)
	}
}

func (h *RelayHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

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

func (h *RelayHandler) GetStats(c *gin.Context) {
	if h.relayManager == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Relay manager not available")
		return
	}

	stats := h.relayManager.GetStats()

	c.JSON(http.StatusOK, stats)
}

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

	relayInfo := h.relayManager.GetRelayByID(relayID)
	if relayInfo == nil {
		apierr.ResourceNotFound(c, "Relay not found")
		return
	}

	h.relayManager.ForceUnregister(relayID)

	h.logAction(c, admin.AuditActionDelete, admin.TargetType("relay"), 0,
		gin.H{"relay_id": relayID, "url": relayInfo.URL},
		nil)

	c.JSON(http.StatusOK, gin.H{
		"status":   "unregistered",
		"relay_id": relayID,
	})
}
