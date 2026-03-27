package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	agentService "github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// UserAgentCredentialHandler handles user agent credential profile requests
type UserAgentCredentialHandler struct {
	credentialSvc *agentService.CredentialProfileService
}

// NewUserAgentCredentialHandler creates a new handler
func NewUserAgentCredentialHandler(credentialSvc *agentService.CredentialProfileService) *UserAgentCredentialHandler {
	return &UserAgentCredentialHandler{
		credentialSvc: credentialSvc,
	}
}

// RegisterRoutes registers user agent credential routes
// Base path: /api/v1/users/agent-credentials
func (h *UserAgentCredentialHandler) RegisterRoutes(rg *gin.RouterGroup) {
	credentials := rg.Group("/agent-credentials")
	{
		// List all profiles grouped by agent
		credentials.GET("", h.ListProfiles)

		// Agent specific routes
		credentials.GET("/agents/:slug", h.ListProfilesForAgent)
		credentials.POST("/agents/:slug", h.CreateProfile)

		// Profile specific routes
		credentials.GET("/profiles/:id", h.GetProfile)
		credentials.PUT("/profiles/:id", h.UpdateProfile)
		credentials.DELETE("/profiles/:id", h.DeleteProfile)
		credentials.POST("/profiles/:id/set-default", h.SetDefault)
	}
}

// ListProfiles lists all credential profiles for the current user, grouped by agent
// GET /api/v1/users/agent-credentials
func (h *UserAgentCredentialHandler) ListProfiles(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profiles, err := h.credentialSvc.ListCredentialProfiles(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list credential profiles")
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": profiles})
}

// ListProfilesForAgent lists all credential profiles for a specific agent
// GET /api/v1/users/agent-credentials/types/:agent_slug
func (h *UserAgentCredentialHandler) ListProfilesForAgent(c *gin.Context) {
	userID := middleware.GetUserID(c)

	agentSlug := c.Param("slug")

	profiles, err := h.credentialSvc.ListCredentialProfilesForAgent(c.Request.Context(), userID, agentSlug)
	if err != nil {
		apierr.InternalError(c, "Failed to list credential profiles")
		return
	}

	// Convert to response format
	responses := make([]*agent.CredentialProfileResponse, len(profiles))
	for i, p := range profiles {
		responses[i] = h.credentialSvc.ProfileToResponse(p)
	}

	// Always include RunnerHost as a virtual option
	runnerHostInfo := gin.H{
		"available":   true,
		"description": "Use Runner machine's local environment configuration",
	}

	c.JSON(http.StatusOK, gin.H{
		"profiles":    responses,
		"runner_host": runnerHostInfo,
	})
}

// CreateCredentialProfileRequest represents a request to create a credential profile
type CreateCredentialProfileRequest struct {
	Name         string            `json:"name" binding:"required,max=100"`
	Description  *string           `json:"description"`
	IsRunnerHost bool              `json:"is_runner_host"`
	Credentials  map[string]string `json:"credentials"`
	IsDefault    bool              `json:"is_default"`
}

// CreateProfile creates a new credential profile for a specific agent
// POST /api/v1/users/agent-credentials/types/:agent_slug
func (h *UserAgentCredentialHandler) CreateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	agentSlug := c.Param("slug")

	var req CreateCredentialProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	profile, err := h.credentialSvc.CreateCredentialProfile(c.Request.Context(), userID, &agentService.CreateCredentialProfileParams{
		AgentSlug:  agentSlug,
		Name:         req.Name,
		Description:  req.Description,
		IsRunnerHost: req.IsRunnerHost,
		Credentials:  req.Credentials,
		IsDefault:    req.IsDefault,
	})

	if err != nil {
		switch err {
		case agentService.ErrAgentNotFound:
			apierr.ResourceNotFound(c, "Agent not found")
		case agentService.ErrCredentialProfileExists:
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Profile with this name already exists")
		default:
			apierr.InternalError(c, "Failed to create profile: "+err.Error())
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"profile": h.credentialSvc.ProfileToResponse(profile)})
}
