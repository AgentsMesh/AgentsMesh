package v1

import (
	"net/http"
	"strconv"

	domainUser "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListProviders lists all repository providers for the current user
// GET /api/v1/user/repository-providers
func (h *UserRepositoryProviderHandler) ListProviders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providers, err := h.userService.ListRepositoryProviders(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list providers")
		return
	}

	// Convert to response format
	responses := make([]*domainUser.RepositoryProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = p.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"providers": responses})
}

// CreateProvider creates a new repository provider
// POST /api/v1/user/repository-providers
func (h *UserRepositoryProviderHandler) CreateProvider(c *gin.Context) {
	var req CreateRepositoryProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	provider, err := h.userService.CreateRepositoryProvider(c.Request.Context(), userID, &user.CreateRepositoryProviderRequest{
		ProviderType: req.ProviderType,
		Name:         req.Name,
		BaseURL:      req.BaseURL,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		BotToken:     req.BotToken,
	})
	if err != nil {
		switch err {
		case user.ErrProviderAlreadyExists:
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Provider already exists with this name")
		case user.ErrInvalidProviderType:
			apierr.InvalidInput(c, "Invalid provider type")
		default:
			apierr.InternalError(c, "Failed to create provider")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"provider": provider.ToResponse()})
}

// GetProvider returns a single repository provider
// GET /api/v1/user/repository-providers/:id
func (h *UserRepositoryProviderHandler) GetProvider(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid provider ID")
		return
	}

	provider, err := h.userService.GetRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			apierr.ResourceNotFound(c, "Provider not found")
			return
		}
		apierr.InternalError(c, "Failed to get provider")
		return
	}

	c.JSON(http.StatusOK, gin.H{"provider": provider.ToResponse()})
}

// UpdateProvider updates a repository provider
// PUT /api/v1/user/repository-providers/:id
func (h *UserRepositoryProviderHandler) UpdateProvider(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid provider ID")
		return
	}

	var req UpdateRepositoryProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	provider, err := h.userService.UpdateRepositoryProvider(c.Request.Context(), userID, providerID, &user.UpdateRepositoryProviderRequest{
		Name:         req.Name,
		BaseURL:      req.BaseURL,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		BotToken:     req.BotToken,
		IsActive:     req.IsActive,
	})
	if err != nil {
		switch err {
		case user.ErrProviderNotFound:
			apierr.ResourceNotFound(c, "Provider not found")
		case user.ErrProviderAlreadyExists:
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Provider already exists with this name")
		default:
			apierr.InternalError(c, "Failed to update provider")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"provider": provider.ToResponse()})
}

// DeleteProvider deletes a repository provider
// DELETE /api/v1/user/repository-providers/:id
func (h *UserRepositoryProviderHandler) DeleteProvider(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid provider ID")
		return
	}

	err = h.userService.DeleteRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			apierr.ResourceNotFound(c, "Provider not found")
			return
		}
		apierr.InternalError(c, "Failed to delete provider")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}
