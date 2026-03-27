package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListAgents lists available agents (builtin + custom)
func (h *AgentHandler) ListAgents(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	builtinAgents, err := h.agentSvc.ListBuiltinAgents(c.Request.Context())
	if err != nil {
		apierr.InternalError(c, "Failed to list builtin agents")
		return
	}

	customAgents, err := h.agentSvc.ListCustomAgents(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.InternalError(c, "Failed to list custom agents")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"builtin_agents": builtinAgents,
		"custom_agents":  customAgents,
	})
}

// GetAgent returns details of a specific agent
func (h *AgentHandler) GetAgent(c *gin.Context) {
	agentSlug := c.Param("agent_slug")

	agentDef, err := h.agentSvc.GetAgent(c.Request.Context(), agentSlug)
	if err != nil {
		apierr.ResourceNotFound(c, "Agent not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"agent": agentDef})
}

// GetAgentConfigSchema returns the config schema for an agent
func (h *AgentHandler) GetAgentConfigSchema(c *gin.Context) {
	agentSlug := c.Param("agent_slug")

	schema, err := h.configBuilder.GetConfigSchema(c.Request.Context(), agentSlug)
	if err != nil {
		apierr.InternalError(c, "Failed to get config schema")
		return
	}

	c.JSON(http.StatusOK, gin.H{"schema": schema})
}
