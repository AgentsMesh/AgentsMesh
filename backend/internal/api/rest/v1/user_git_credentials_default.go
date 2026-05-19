package v1

import (
	"errors"
	"net/http"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *UserGitCredentialHandler) GetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credential, err := h.userService.GetDefaultGitCredential(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to get default credential")
		return
	}

	if credential == nil {
		c.JSON(http.StatusOK, gin.H{
			"credential":      domainUser.RunnerLocalCredentialResponse(),
			"is_runner_local": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"credential":      toGitCredentialResponse(credential),
		"is_runner_local": false,
	})
}

type SetDefaultRequest struct {
	CredentialID *int64 `json:"credential_id"` // nil means runner_local
}

func (h *UserGitCredentialHandler) SetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req SetDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if req.CredentialID == nil {
		err := h.userService.ClearDefaultGitCredential(c.Request.Context(), userID)
		if err != nil {
			apierr.InternalError(c, "Failed to set default")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":         "Default set to Runner Local",
			"is_runner_local": true,
		})
		return
	}

	err := h.userService.SetDefaultGitCredential(c.Request.Context(), userID, *req.CredentialID)
	if err != nil {
		if errors.Is(err, user.ErrCredentialNotFound) {
			apierr.ResourceNotFound(c, "Credential not found")
			return
		}
		apierr.InternalError(c, "Failed to set default")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Default credential set",
		"is_runner_local": false,
	})
}

func (h *UserGitCredentialHandler) ClearDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	err := h.userService.ClearDefaultGitCredential(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to clear default")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Default cleared, using Runner Local",
	})
}
