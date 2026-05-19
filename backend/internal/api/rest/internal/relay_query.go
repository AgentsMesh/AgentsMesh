package internal

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *RelayHandler) Stats(c *gin.Context) {
	stats := h.relayManager.GetStats()
	c.JSON(http.StatusOK, stats)
}

func (h *RelayHandler) List(c *gin.Context) {
	relays := h.relayManager.GetRelays()
	c.JSON(http.StatusOK, gin.H{"relays": relays})
}

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
