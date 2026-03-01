package v1

import (
	"net/http"
	"strconv"

	agentDomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agent"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ============================================================================
// User Agent Config API (Personal Runtime Configuration)
// ============================================================================

// ListUserAgentConfigs returns all personal configs for the current user
// GET /api/v1/users/me/agent-configs
func (h *AgentHandler) ListUserAgentConfigs(c *gin.Context) {
	userID := middleware.GetUserID(c)

	configs, err := h.userConfigSvc.ListUserAgentConfigs(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list user configs")
		return
	}

	// Convert to response format
	responses := make([]*agentDomain.UserAgentConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = cfg.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"configs": responses})
}

// GetUserAgentConfig returns the user's personal config for an agent type
// GET /api/v1/users/me/agent-configs/:agent_type_id
func (h *AgentHandler) GetUserAgentConfig(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid agent type ID")
		return
	}

	userID := middleware.GetUserID(c)

	config, err := h.userConfigSvc.GetUserAgentConfig(c.Request.Context(), userID, agentTypeID)
	if err != nil {
		apierr.InternalError(c, "Failed to get user config")
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config.ToResponse()})
}

// SetUserAgentConfig sets the user's personal config for an agent type
// PUT /api/v1/users/me/agent-configs/:agent_type_id
func (h *AgentHandler) SetUserAgentConfig(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid agent type ID")
		return
	}

	var req SetUserAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	// Convert to ConfigValues
	configValues := make(agentDomain.ConfigValues)
	for k, v := range req.ConfigValues {
		configValues[k] = v
	}

	config, err := h.userConfigSvc.SetUserAgentConfig(c.Request.Context(), userID, agentTypeID, configValues)
	if err != nil {
		apierr.InternalError(c, "Failed to set user config")
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config.ToResponse()})
}

// DeleteUserAgentConfig deletes the user's personal config for an agent type
// DELETE /api/v1/users/me/agent-configs/:agent_type_id
func (h *AgentHandler) DeleteUserAgentConfig(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid agent type ID")
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.userConfigSvc.DeleteUserAgentConfig(c.Request.Context(), userID, agentTypeID); err != nil {
		apierr.InternalError(c, "Failed to delete user config")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User config deleted"})
}
