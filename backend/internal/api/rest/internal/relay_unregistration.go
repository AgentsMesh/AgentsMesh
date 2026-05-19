package internal

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *RelayHandler) Unregister(c *gin.Context) {
	var req UnregisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	relayInfo := h.relayManager.GetRelayByID(req.RelayID)
	if relayInfo == nil {
		// Relay not found, but that's OK for unregister (idempotent)
		h.logger.Info("Unregister request for unknown relay",
			"relay_id", req.RelayID,
			"reason", req.Reason)
		c.JSON(http.StatusOK, UnregisterResponse{Status: "not_found"})
		return
	}

	h.relayManager.GracefulUnregister(req.RelayID, req.Reason)

	h.logger.Info("Relay gracefully unregistered",
		"relay_id", req.RelayID,
		"reason", req.Reason)

	c.JSON(http.StatusOK, UnregisterResponse{
		Status: "unregistered",
		Reason: req.Reason,
	})
}

func (h *RelayHandler) ForceUnregister(c *gin.Context) {
	relayID := c.Param("relay_id")
	if relayID == "" {
		apierr.InvalidInput(c, "relay_id is required")
		return
	}

	// Check if relay exists (idempotent: not-found is OK)
	relayInfo := h.relayManager.GetRelayByID(relayID)
	if relayInfo == nil {
		h.logger.Info("Force unregister request for unknown relay", "relay_id", relayID)
		c.JSON(http.StatusOK, UnregisterResponse{Status: "not_found"})
		return
	}

	h.relayManager.ForceUnregister(relayID)

	h.logger.Info("Relay force unregistered", "relay_id", relayID)

	c.JSON(http.StatusOK, UnregisterResponse{
		Status:  "unregistered",
		RelayID: relayID,
	})
}
