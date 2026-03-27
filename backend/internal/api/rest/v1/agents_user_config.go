package v1

import (
	"net/http"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListUserAgentConfigs returns all personal configs for the current user
func (h *AgentHandler) ListUserAgentConfigs(c *gin.Context) {
	userID := middleware.GetUserID(c)

	configs, err := h.userConfigSvc.ListUserAgentConfigs(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list user configs")
		return
	}

	responses := make([]*agentDomain.UserAgentConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = cfg.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"configs": responses})
}

// GetUserAgentConfig returns the user's personal config for an agent
func (h *AgentHandler) GetUserAgentConfig(c *gin.Context) {
	agentSlug := c.Param("slug")
	userID := middleware.GetUserID(c)

	config, err := h.userConfigSvc.GetUserAgentConfig(c.Request.Context(), userID, agentSlug)
	if err != nil {
		apierr.InternalError(c, "Failed to get user config")
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config.ToResponse()})
}

// SetUserAgentConfig sets the user's personal config for an agent
func (h *AgentHandler) SetUserAgentConfig(c *gin.Context) {
	agentSlug := c.Param("slug")
	userID := middleware.GetUserID(c)

	var req SetUserAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	configValues := make(agentDomain.ConfigValues)
	for k, v := range req.ConfigValues {
		configValues[k] = v
	}

	config, err := h.userConfigSvc.SetUserAgentConfig(c.Request.Context(), userID, agentSlug, configValues)
	if err != nil {
		apierr.InternalError(c, "Failed to set user config")
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config.ToResponse()})
}

// DeleteUserAgentConfig deletes the user's personal config for an agent
func (h *AgentHandler) DeleteUserAgentConfig(c *gin.Context) {
	agentSlug := c.Param("slug")
	userID := middleware.GetUserID(c)

	if err := h.userConfigSvc.DeleteUserAgentConfig(c.Request.Context(), userID, agentSlug); err != nil {
		apierr.InternalError(c, "Failed to delete user config")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User config deleted"})
}
