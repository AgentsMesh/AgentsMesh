package v1

import (
	"errors"
	"net/http"
	"strconv"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type UserGitCredentialHandler struct {
	userService *user.Service
}

func NewUserGitCredentialHandler(userSvc *user.Service) *UserGitCredentialHandler {
	return &UserGitCredentialHandler{
		userService: userSvc,
	}
}

func (h *UserGitCredentialHandler) RegisterRoutes(rg *gin.RouterGroup) {
	credentials := rg.Group("/git-credentials")
	{
		credentials.GET("", h.ListCredentials)
		credentials.POST("", h.CreateCredential)
		credentials.GET("/default", h.GetDefault)
		credentials.POST("/default", h.SetDefault)
		credentials.DELETE("/default", h.ClearDefault)
		credentials.GET("/:id", h.GetCredential)
		credentials.PUT("/:id", h.UpdateCredential)
		credentials.DELETE("/:id", h.DeleteCredential)
	}
}

func toGitCredentialResponse(c *domainUser.GitCredential) *domainUser.GitCredentialResponse {
	return c.ToResponse()
}

func (h *UserGitCredentialHandler) ListCredentials(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credentials, err := h.userService.ListGitCredentials(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list credentials")
		return
	}

	responses := make([]*domainUser.GitCredentialResponse, len(credentials))
	for i, cred := range credentials {
		responses[i] = toGitCredentialResponse(cred)
	}

	runnerLocal := domainUser.RunnerLocalCredentialResponse()
	defaultCred, _ := h.userService.GetDefaultGitCredential(c.Request.Context(), userID)
	if defaultCred == nil {
		runnerLocal.IsDefault = true
	}

	c.JSON(http.StatusOK, gin.H{
		"credentials":  responses,
		"runner_local": runnerLocal,
	})
}

type CreateCredentialRequest struct {
	Name                 string `json:"name" binding:"required"`
	CredentialType       string `json:"credential_type" binding:"required"`
	RepositoryProviderID *int64 `json:"repository_provider_id"`
	PAT                  string `json:"pat"`
	PrivateKey           string `json:"private_key"`
	HostPattern          string `json:"host_pattern"`
}

func (h *UserGitCredentialHandler) CreateCredential(c *gin.Context) {
	var req CreateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	credential, err := h.userService.CreateGitCredential(c.Request.Context(), userID, &user.CreateGitCredentialRequest{
		Name:                 req.Name,
		CredentialType:       req.CredentialType,
		RepositoryProviderID: req.RepositoryProviderID,
		PAT:                  req.PAT,
		PrivateKey:           req.PrivateKey,
		HostPattern:          req.HostPattern,
	})
	if err != nil {
		switch {
		case errors.Is(err, user.ErrCredentialAlreadyExists):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Credential already exists with this name")
		case errors.Is(err, user.ErrInvalidCredentialType):
			apierr.InvalidInput(c, "Invalid credential type. Valid types: runner_local, oauth, pat, ssh_key")
		case errors.Is(err, user.ErrProviderIDRequired):
			apierr.BadRequest(c, apierr.MISSING_REQUIRED, "repository_provider_id is required for oauth type")
		case errors.Is(err, user.ErrInvalidSSHKey):
			apierr.InvalidInput(c, "Invalid SSH key format")
		case errors.Is(err, user.ErrProviderNotFound):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Repository provider not found")
		default:
			apierr.InternalError(c, "Failed to create credential: "+err.Error())
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"credential": toGitCredentialResponse(credential)})
}

func (h *UserGitCredentialHandler) GetCredential(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credentialID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid credential ID")
		return
	}

	credential, err := h.userService.GetGitCredential(c.Request.Context(), userID, credentialID)
	if err != nil {
		if errors.Is(err, user.ErrCredentialNotFound) {
			apierr.ResourceNotFound(c, "Credential not found")
			return
		}
		apierr.InternalError(c, "Failed to get credential")
		return
	}

	c.JSON(http.StatusOK, gin.H{"credential": toGitCredentialResponse(credential)})
}

type UpdateCredentialRequest struct {
	Name        *string `json:"name"`
	PAT         *string `json:"pat"`
	PrivateKey  *string `json:"private_key"`
	HostPattern *string `json:"host_pattern"`
}

func (h *UserGitCredentialHandler) UpdateCredential(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credentialID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid credential ID")
		return
	}

	var req UpdateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	credential, err := h.userService.UpdateGitCredential(c.Request.Context(), userID, credentialID, &user.UpdateGitCredentialRequest{
		Name:        req.Name,
		PAT:         req.PAT,
		PrivateKey:  req.PrivateKey,
		HostPattern: req.HostPattern,
	})
	if err != nil {
		switch {
		case errors.Is(err, user.ErrCredentialNotFound):
			apierr.ResourceNotFound(c, "Credential not found")
		case errors.Is(err, user.ErrCredentialAlreadyExists):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Credential already exists with this name")
		case errors.Is(err, user.ErrInvalidSSHKey):
			apierr.InvalidInput(c, "Invalid SSH key format")
		default:
			apierr.InternalError(c, "Failed to update credential")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"credential": toGitCredentialResponse(credential)})
}

func (h *UserGitCredentialHandler) DeleteCredential(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credentialID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid credential ID")
		return
	}

	err = h.userService.DeleteGitCredential(c.Request.Context(), userID, credentialID)
	if err != nil {
		if errors.Is(err, user.ErrCredentialNotFound) {
			apierr.ResourceNotFound(c, "Credential not found")
			return
		}
		apierr.InternalError(c, "Failed to delete credential")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credential deleted"})
}
