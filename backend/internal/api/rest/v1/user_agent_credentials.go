package v1

import (
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	agentService "github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type UserAgentCredentialHandler struct {
	credentialSvc *agentService.CredentialProfileService
}

func NewUserAgentCredentialHandler(credentialSvc *agentService.CredentialProfileService) *UserAgentCredentialHandler {
	return &UserAgentCredentialHandler{
		credentialSvc: credentialSvc,
	}
}

func (h *UserAgentCredentialHandler) RegisterRoutes(rg *gin.RouterGroup) {
	credentials := rg.Group("/agent-credentials")
	{
		credentials.GET("", h.ListProfiles)

		credentials.GET("/agents/:slug", h.ListProfilesForAgent)
		credentials.POST("/agents/:slug", h.CreateProfile)

		credentials.GET("/profiles/:id", h.GetProfile)
		credentials.PUT("/profiles/:id", h.UpdateProfile)
		credentials.DELETE("/profiles/:id", h.DeleteProfile)
		credentials.POST("/profiles/:id/set-default", h.SetDefault)
	}
}

func (h *UserAgentCredentialHandler) ListProfiles(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profiles, err := h.credentialSvc.ListCredentialProfiles(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list credential profiles")
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": profiles})
}

func (h *UserAgentCredentialHandler) ListProfilesForAgent(c *gin.Context) {
	userID := middleware.GetUserID(c)

	agentSlug := c.Param("slug")

	profiles, err := h.credentialSvc.ListCredentialProfilesForAgent(c.Request.Context(), userID, agentSlug)
	if err != nil {
		apierr.InternalError(c, "Failed to list credential profiles")
		return
	}

	responses := make([]*agent.CredentialProfileResponse, len(profiles))
	for i, p := range profiles {
		responses[i] = h.credentialSvc.ProfileToResponse(p)
	}

	runnerHostInfo := gin.H{
		"available":   true,
		"description": "Use Runner machine's local environment configuration",
	}

	c.JSON(http.StatusOK, gin.H{
		"profiles":    responses,
		"runner_host": runnerHostInfo,
	})
}

type CreateCredentialProfileRequest struct {
	Name         string            `json:"name" binding:"required,max=100"`
	Description  *string           `json:"description"`
	IsRunnerHost bool              `json:"is_runner_host"`
	Credentials  map[string]string `json:"credentials"`
	IsDefault    bool              `json:"is_default"`
}

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
		switch {
		case errors.Is(err, agentService.ErrAgentNotFound):
			apierr.ResourceNotFound(c, "Agent not found")
		case errors.Is(err, agentService.ErrCredentialProfileExists):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Profile with this name already exists")
		default:
			apierr.InternalError(c, "Failed to create profile: "+err.Error())
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"profile": h.credentialSvc.ProfileToResponse(profile)})
}
