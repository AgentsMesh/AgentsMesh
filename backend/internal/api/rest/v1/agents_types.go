package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// ListAgentTypes lists available agent types
// GET /api/v1/organizations/:slug/agents/types
func (h *AgentHandler) ListAgentTypes(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	// Get builtin types
	builtinTypes, err := h.agentTypeSvc.ListBuiltinAgentTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list builtin agent types"})
		return
	}

	// Get custom types for organization
	customTypes, err := h.agentTypeSvc.ListCustomAgentTypes(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list custom agent types"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"builtin_types": builtinTypes,
		"custom_types":  customTypes,
	})
}

// GetAgentType returns details of a specific agent type
// GET /api/v1/organizations/:slug/agents/types/:agent_type_id
func (h *AgentHandler) GetAgentType(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	agentType, err := h.agentTypeSvc.GetAgentType(c.Request.Context(), agentTypeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent type not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agent_type": agentType})
}

// GetAgentTypeConfigSchema returns the raw config schema for an agent type
// Frontend is responsible for i18n translation using: agent.{slug}.fields.{field.name}.label
// GET /api/v1/organizations/:slug/agents/:agent_type_id/config-schema
func (h *AgentHandler) GetAgentTypeConfigSchema(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	schema, err := h.configBuilder.GetConfigSchema(c.Request.Context(), agentTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config schema"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schema": schema})
}
