package internal

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// Stats returns relay statistics
// GET /api/internal/relays/stats
func (h *RelayHandler) Stats(c *gin.Context) {
	stats := h.relayManager.GetStats()
	c.JSON(http.StatusOK, stats)
}

// List returns all registered relays
// GET /api/internal/relays
func (h *RelayHandler) List(c *gin.Context) {
	relays := h.relayManager.GetRelays()
	c.JSON(http.StatusOK, gin.H{"relays": relays})
}

// Get returns a specific relay by ID
// GET /api/internal/relays/:relay_id
func (h *RelayHandler) Get(c *gin.Context) {
	relayID := c.Param("relay_id")
	if relayID == "" {
		apierr.InvalidInput(c, "relay_id is required")
		return
	}

	relayInfo := h.relayManager.GetRelayByID(relayID)
	if relayInfo == nil {
		apierr.ResourceNotFound(c, "relay not found")
		return
	}

	c.JSON(http.StatusOK, relayInfo)
}
