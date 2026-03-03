package v1

import (
	"net/http"
	"strconv"

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
		// List all profiles grouped by agent type
		credentials.GET("", h.ListProfiles)

		// Agent type specific routes
		credentials.GET("/types/:agent_type_id", h.ListProfilesForAgentType)
		credentials.POST("/types/:agent_type_id", h.CreateProfile)

		// Profile specific routes
		credentials.GET("/profiles/:id", h.GetProfile)
		credentials.PUT("/profiles/:id", h.UpdateProfile)
		credentials.DELETE("/profiles/:id", h.DeleteProfile)
		credentials.POST("/profiles/:id/set-default", h.SetDefault)
	}
}

// ListProfiles lists all credential profiles for the current user, grouped by agent type
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

// ListProfilesForAgentType lists all credential profiles for a specific agent type
// GET /api/v1/users/agent-credentials/types/:agent_type_id
func (h *UserAgentCredentialHandler) ListProfilesForAgentType(c *gin.Context) {
	userID := middleware.GetUserID(c)

	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid agent_type_id")
		return
	}

	profiles, err := h.credentialSvc.ListCredentialProfilesForAgentType(c.Request.Context(), userID, agentTypeID)
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

// CreateProfile creates a new credential profile for a specific agent type
// POST /api/v1/users/agent-credentials/types/:agent_type_id
func (h *UserAgentCredentialHandler) CreateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid agent_type_id")
		return
	}

	var req CreateCredentialProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	profile, err := h.credentialSvc.CreateCredentialProfile(c.Request.Context(), userID, &agentService.CreateCredentialProfileParams{
		AgentTypeID:  agentTypeID,
		Name:         req.Name,
		Description:  req.Description,
		IsRunnerHost: req.IsRunnerHost,
		Credentials:  req.Credentials,
		IsDefault:    req.IsDefault,
	})

	if err != nil {
		switch err {
		case agentService.ErrAgentTypeNotFound:
			apierr.ResourceNotFound(c, "Agent type not found")
		case agentService.ErrCredentialProfileExists:
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Profile with this name already exists")
		default:
			apierr.InternalError(c, "Failed to create profile: "+err.Error())
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"profile": h.credentialSvc.ProfileToResponse(profile)})
}

// GetProfile returns a single credential profile
// GET /api/v1/users/agent-credentials/profiles/:id
func (h *UserAgentCredentialHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid profile ID")
		return
	}

	profile, err := h.credentialSvc.GetCredentialProfile(c.Request.Context(), userID, profileID)
	if err != nil {
		if err == agentService.ErrCredentialProfileNotFound {
			apierr.ResourceNotFound(c, "Profile not found")
			return
		}
		apierr.InternalError(c, "Failed to get profile")
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": h.credentialSvc.ProfileToResponse(profile)})
}

// UpdateCredentialProfileRequest represents a request to update a credential profile
type UpdateCredentialProfileRequest struct {
	Name         *string           `json:"name"`
	Description  *string           `json:"description"`
	IsRunnerHost *bool             `json:"is_runner_host"`
	Credentials  map[string]string `json:"credentials"`
	IsDefault    *bool             `json:"is_default"`
	IsActive     *bool             `json:"is_active"`
}

// UpdateProfile updates a credential profile
// PUT /api/v1/users/agent-credentials/profiles/:id
func (h *UserAgentCredentialHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid profile ID")
		return
	}

	var req UpdateCredentialProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	profile, err := h.credentialSvc.UpdateCredentialProfile(c.Request.Context(), userID, profileID, &agentService.UpdateCredentialProfileParams{
		Name:         req.Name,
		Description:  req.Description,
		IsRunnerHost: req.IsRunnerHost,
		Credentials:  req.Credentials,
		IsDefault:    req.IsDefault,
		IsActive:     req.IsActive,
	})

	if err != nil {
		switch err {
		case agentService.ErrCredentialProfileNotFound:
			apierr.ResourceNotFound(c, "Profile not found")
		case agentService.ErrCredentialProfileExists:
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Profile with this name already exists")
		default:
			apierr.InternalError(c, "Failed to update profile")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": h.credentialSvc.ProfileToResponse(profile)})
}

// DeleteProfile deletes a credential profile
// DELETE /api/v1/users/agent-credentials/profiles/:id
func (h *UserAgentCredentialHandler) DeleteProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid profile ID")
		return
	}

	err = h.credentialSvc.DeleteCredentialProfile(c.Request.Context(), userID, profileID)
	if err != nil {
		if err == agentService.ErrCredentialProfileNotFound {
			apierr.ResourceNotFound(c, "Profile not found")
			return
		}
		apierr.InternalError(c, "Failed to delete profile")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile deleted"})
}

// SetDefault sets a profile as the default for its agent type
// POST /api/v1/users/agent-credentials/profiles/:id/set-default
func (h *UserAgentCredentialHandler) SetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid profile ID")
		return
	}

	profile, err := h.credentialSvc.SetDefaultCredentialProfile(c.Request.Context(), userID, profileID)
	if err != nil {
		if err == agentService.ErrCredentialProfileNotFound {
			apierr.ResourceNotFound(c, "Profile not found")
			return
		}
		apierr.InternalError(c, "Failed to set default")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile set as default",
		"profile": h.credentialSvc.ProfileToResponse(profile),
	})
}
