package v1

import (
	"net/http"
	"strconv"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/gin-gonic/gin"
)

// CreateCustomAgent creates a custom agent type
// POST /api/v1/organizations/:slug/agents/custom
func (h *AgentHandler) CreateCustomAgent(c *gin.Context) {
	var req CreateCustomAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	// Convert request to service request
	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	var args *string
	if req.DefaultArgs != "" {
		args = &req.DefaultArgs
	}

	// Convert credential schema
	var credSchema agentDomain.CredentialSchema
	if req.CredentialSchema != nil {
		// TODO: properly convert credential schema from map to CredentialSchema
	}

	// Convert status detection
	var statusDetection agentDomain.StatusDetection
	if req.StatusDetection != nil {
		statusDetection = make(agentDomain.StatusDetection)
		for k, v := range req.StatusDetection {
			statusDetection[k] = v
		}
	}

	customAgent, err := h.agentTypeSvc.CreateCustomAgentType(c.Request.Context(), tenant.OrganizationID, &agent.CreateCustomAgentRequest{
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      desc,
		LaunchCommand:    req.LaunchCommand,
		DefaultArgs:      args,
		CredentialSchema: credSchema,
		StatusDetection:  statusDetection,
	})
	if err != nil {
		if err == agent.ErrAgentSlugExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Agent slug already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create custom agent"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"custom_agent": customAgent})
}

// UpdateCustomAgent updates a custom agent type
// PUT /api/v1/organizations/:slug/agents/custom/:id
func (h *AgentHandler) UpdateCustomAgent(c *gin.Context) {
	customAgentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid custom agent ID"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	customAgent, err := h.agentTypeSvc.UpdateCustomAgentType(c.Request.Context(), customAgentID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update custom agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"custom_agent": customAgent})
}

// DeleteCustomAgent deletes a custom agent type
// DELETE /api/v1/organizations/:slug/agents/custom/:id
func (h *AgentHandler) DeleteCustomAgent(c *gin.Context) {
	customAgentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid custom agent ID"})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	if err := h.agentTypeSvc.DeleteCustomAgentType(c.Request.Context(), customAgentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete custom agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Custom agent deleted"})
}
