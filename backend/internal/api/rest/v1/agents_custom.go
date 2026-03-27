package v1

import (
	"net/http"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/podfile/extract"
	"github.com/anthropics/agentsmesh/podfile/parser"
	"github.com/gin-gonic/gin"
)

// CreateCustomAgent creates a custom agent
func (h *AgentHandler) CreateCustomAgent(c *gin.Context) {
	var req CreateCustomAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	var args *string
	if req.DefaultArgs != "" {
		args = &req.DefaultArgs
	}

	var credSchema agentDomain.CredentialSchema
	var statusDetection agentDomain.StatusDetection
	if req.StatusDetection != nil {
		statusDetection = make(agentDomain.StatusDetection)
		for k, v := range req.StatusDetection {
			statusDetection[k] = v
		}
	}

	launchCommand := req.LaunchCommand
	var podfileSource *string
	if req.PodfileSource != "" {
		podfileSource = &req.PodfileSource
		if launchCommand == "" {
			prog, parseErrors := parser.Parse(req.PodfileSource)
			if len(parseErrors) > 0 {
				apierr.ValidationError(c, "Invalid PodFile: "+parseErrors[0])
				return
			}
			spec := extract.Extract(prog)
			launchCommand = spec.Agent.Command
			if launchCommand == "" {
				apierr.ValidationError(c, "PodFile must declare AGENT command")
				return
			}
		}
	} else if launchCommand == "" {
		apierr.ValidationError(c, "Either podfile_source or launch_command is required")
		return
	}

	customAgent, err := h.agentSvc.CreateCustomAgent(c.Request.Context(), tenant.OrganizationID, &agent.CreateCustomAgentRequest{
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      desc,
		LaunchCommand:    launchCommand,
		DefaultArgs:      args,
		CredentialSchema: credSchema,
		StatusDetection:  statusDetection,
		PodfileSource:    podfileSource,
	})
	if err != nil {
		if err == agent.ErrAgentSlugExists {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Agent slug already exists")
			return
		}
		apierr.InternalError(c, "Failed to create custom agent")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"custom_agent": customAgent})
}

// UpdateCustomAgent updates a custom agent
func (h *AgentHandler) UpdateCustomAgent(c *gin.Context) {
	customAgentSlug := c.Param("agent_slug")

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	customAgent, err := h.agentSvc.UpdateCustomAgent(c.Request.Context(), tenant.OrganizationID, customAgentSlug, req)
	if err != nil {
		apierr.InternalError(c, "Failed to update custom agent")
		return
	}

	c.JSON(http.StatusOK, gin.H{"custom_agent": customAgent})
}

// DeleteCustomAgent deletes a custom agent
func (h *AgentHandler) DeleteCustomAgent(c *gin.Context) {
	customAgentSlug := c.Param("agent_slug")

	tenant := middleware.GetTenant(c)
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	if err := h.agentSvc.DeleteCustomAgent(c.Request.Context(), tenant.OrganizationID, customAgentSlug); err != nil {
		apierr.InternalError(c, "Failed to delete custom agent")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Custom agent deleted"})
}
