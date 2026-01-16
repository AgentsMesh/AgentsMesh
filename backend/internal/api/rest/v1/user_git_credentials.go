package v1

import (
	"net/http"
	"strconv"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// UserGitCredentialHandler handles user Git credential requests
type UserGitCredentialHandler struct {
	userService *user.Service
}

// NewUserGitCredentialHandler creates a new user Git credential handler
func NewUserGitCredentialHandler(userSvc *user.Service) *UserGitCredentialHandler {
	return &UserGitCredentialHandler{
		userService: userSvc,
	}
}

// RegisterRoutes registers user Git credential routes
// Note: rg is already prefixed with /users, so we use /git-credentials
// Final path: /api/v1/users/git-credentials
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

// toGitCredentialResponse converts domain model to API response
func toGitCredentialResponse(c *domainUser.GitCredential) *domainUser.GitCredentialResponse {
	return c.ToResponse()
}

// ListCredentials lists all Git credentials for the current user
// GET /api/v1/user/git-credentials
func (h *UserGitCredentialHandler) ListCredentials(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credentials, err := h.userService.ListGitCredentials(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list credentials"})
		return
	}

	// Convert to response format
	responses := make([]*domainUser.GitCredentialResponse, len(credentials))
	for i, cred := range credentials {
		responses[i] = toGitCredentialResponse(cred)
	}

	// Prepend virtual "Runner Local" option
	runnerLocal := domainUser.RunnerLocalCredentialResponse()
	// Check if runner_local is the default (no default credential set)
	defaultCred, _ := h.userService.GetDefaultGitCredential(c.Request.Context(), userID)
	if defaultCred == nil {
		runnerLocal.IsDefault = true
	}

	c.JSON(http.StatusOK, gin.H{
		"credentials":  responses,
		"runner_local": runnerLocal,
	})
}

// CreateCredentialRequest represents a request to create a Git credential
type CreateCredentialRequest struct {
	Name                 string `json:"name" binding:"required"`
	CredentialType       string `json:"credential_type" binding:"required"`
	RepositoryProviderID *int64 `json:"repository_provider_id"`
	PAT                  string `json:"pat"`
	PrivateKey           string `json:"private_key"`
	HostPattern          string `json:"host_pattern"`
}

// CreateCredential creates a new Git credential
// POST /api/v1/user/git-credentials
func (h *UserGitCredentialHandler) CreateCredential(c *gin.Context) {
	var req CreateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		switch err {
		case user.ErrCredentialAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Credential already exists with this name"})
		case user.ErrInvalidCredentialType:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credential type. Valid types: runner_local, oauth, pat, ssh_key"})
		case user.ErrProviderIDRequired:
			c.JSON(http.StatusBadRequest, gin.H{"error": "repository_provider_id is required for oauth type"})
		case user.ErrInvalidSSHKey:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SSH key format"})
		case user.ErrProviderNotFound:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Repository provider not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create credential: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"credential": toGitCredentialResponse(credential)})
}

// GetCredential returns a single Git credential
// GET /api/v1/user/git-credentials/:id
func (h *UserGitCredentialHandler) GetCredential(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credentialID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credential ID"})
		return
	}

	credential, err := h.userService.GetGitCredential(c.Request.Context(), userID, credentialID)
	if err != nil {
		if err == user.ErrCredentialNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Credential not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get credential"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"credential": toGitCredentialResponse(credential)})
}

// UpdateCredentialRequest represents a request to update a Git credential
type UpdateCredentialRequest struct {
	Name        *string `json:"name"`
	PAT         *string `json:"pat"`
	PrivateKey  *string `json:"private_key"`
	HostPattern *string `json:"host_pattern"`
}

// UpdateCredential updates a Git credential
// PUT /api/v1/user/git-credentials/:id
func (h *UserGitCredentialHandler) UpdateCredential(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credentialID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credential ID"})
		return
	}

	var req UpdateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	credential, err := h.userService.UpdateGitCredential(c.Request.Context(), userID, credentialID, &user.UpdateGitCredentialRequest{
		Name:        req.Name,
		PAT:         req.PAT,
		PrivateKey:  req.PrivateKey,
		HostPattern: req.HostPattern,
	})
	if err != nil {
		switch err {
		case user.ErrCredentialNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Credential not found"})
		case user.ErrCredentialAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Credential already exists with this name"})
		case user.ErrInvalidSSHKey:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SSH key format"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update credential"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"credential": toGitCredentialResponse(credential)})
}

// DeleteCredential deletes a Git credential
// DELETE /api/v1/user/git-credentials/:id
func (h *UserGitCredentialHandler) DeleteCredential(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credentialID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credential ID"})
		return
	}

	err = h.userService.DeleteGitCredential(c.Request.Context(), userID, credentialID)
	if err != nil {
		if err == user.ErrCredentialNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Credential not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete credential"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credential deleted"})
}

// GetDefault returns the user's default Git credential
// GET /api/v1/user/git-credentials/default
func (h *UserGitCredentialHandler) GetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	credential, err := h.userService.GetDefaultGitCredential(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get default credential"})
		return
	}

	if credential == nil {
		// No default set, return runner_local as default
		c.JSON(http.StatusOK, gin.H{
			"credential": domainUser.RunnerLocalCredentialResponse(),
			"is_runner_local": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"credential": toGitCredentialResponse(credential),
		"is_runner_local": false,
	})
}

// SetDefaultRequest represents a request to set the default Git credential
type SetDefaultRequest struct {
	CredentialID *int64 `json:"credential_id"` // nil means runner_local
}

// SetDefault sets the user's default Git credential
// POST /api/v1/user/git-credentials/default
func (h *UserGitCredentialHandler) SetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req SetDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.CredentialID == nil {
		// Set runner_local as default (clear default credential)
		err := h.userService.ClearDefaultGitCredential(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Default set to Runner Local",
			"is_runner_local": true,
		})
		return
	}

	err := h.userService.SetDefaultGitCredential(c.Request.Context(), userID, *req.CredentialID)
	if err != nil {
		if err == user.ErrCredentialNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Credential not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Default credential set",
		"is_runner_local": false,
	})
}

// ClearDefault clears the user's default Git credential (falls back to runner_local)
// DELETE /api/v1/user/git-credentials/default
func (h *UserGitCredentialHandler) ClearDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	err := h.userService.ClearDefaultGitCredential(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear default"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Default cleared, using Runner Local",
	})
}
