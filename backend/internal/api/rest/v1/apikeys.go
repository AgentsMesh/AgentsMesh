package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/apikey"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// APIKeyHandler handles API key management endpoints
type APIKeyHandler struct {
	apiKeyService apikey.Interface
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(apiKeyService apikey.Interface) *APIKeyHandler {
	return &APIKeyHandler{apiKeyService: apiKeyService}
}

// CreateAPIKey creates a new API key for the organization
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	var req struct {
		Name        string   `json:"name" binding:"required,max=255"`
		Description *string  `json:"description"`
		Scopes      []string `json:"scopes" binding:"required,min=1"`
		ExpiresIn   *int     `json:"expires_in"` // seconds, null = never
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	result, err := h.apiKeyService.CreateAPIKey(c.Request.Context(), &apikey.CreateAPIKeyRequest{
		OrganizationID: tenant.OrganizationID,
		CreatedBy:      tenant.UserID,
		Name:           req.Name,
		Description:    req.Description,
		Scopes:         req.Scopes,
		ExpiresIn:      req.ExpiresIn,
	})
	if err != nil {
		handleAPIKeyServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"api_key": result.APIKey,
		"raw_key": result.RawKey,
	})
}

// ListAPIKeys lists all API keys for the organization
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	keys, total, err := h.apiKeyService.ListAPIKeys(c.Request.Context(), &apikey.ListAPIKeysFilter{
		OrganizationID: tenant.OrganizationID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to list API keys")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys": keys,
		"total":    total,
	})
}

// GetAPIKey retrieves a single API key
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid API key ID")
		return
	}

	key, err := h.apiKeyService.GetAPIKey(c.Request.Context(), id, tenant.OrganizationID)
	if err != nil {
		handleAPIKeyServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": key})
}

// UpdateAPIKey updates an API key's metadata
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid API key ID")
		return
	}

	var req apikey.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	key, err := h.apiKeyService.UpdateAPIKey(c.Request.Context(), id, tenant.OrganizationID, &req)
	if err != nil {
		handleAPIKeyServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": key})
}

// DeleteAPIKey permanently deletes an API key
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid API key ID")
		return
	}

	if err := h.apiKeyService.DeleteAPIKey(c.Request.Context(), id, tenant.OrganizationID); err != nil {
		handleAPIKeyServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

// RevokeAPIKey disables an API key
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid API key ID")
		return
	}

	if err := h.apiKeyService.RevokeAPIKey(c.Request.Context(), id, tenant.OrganizationID); err != nil {
		handleAPIKeyServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// handleAPIKeyServiceError maps service errors to HTTP responses using errors.Is()
func handleAPIKeyServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apikey.ErrAPIKeyNotFound):
		apierr.ResourceNotFound(c, "API key not found")
	case errors.Is(err, apikey.ErrNameEmpty):
		apierr.ValidationError(c, err.Error())
	case errors.Is(err, apikey.ErrNameTooLong):
		apierr.ValidationError(c, err.Error())
	case errors.Is(err, apikey.ErrScopesRequired):
		apierr.ValidationError(c, err.Error())
	case errors.Is(err, apikey.ErrInvalidScope):
		apierr.ValidationError(c, err.Error())
	case errors.Is(err, apikey.ErrDuplicateKeyName):
		apierr.Conflict(c, apierr.ALREADY_EXISTS, err.Error())
	case errors.Is(err, apikey.ErrInvalidExpiresIn):
		apierr.ValidationError(c, err.Error())
	default:
		apierr.InternalError(c, "Internal server error")
	}
}
