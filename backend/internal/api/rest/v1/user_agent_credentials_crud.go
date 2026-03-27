package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	agentService "github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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

// SetDefault sets a profile as the default for its agent
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
