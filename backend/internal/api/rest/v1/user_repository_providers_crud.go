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

func (h *UserRepositoryProviderHandler) ListProviders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providers, err := h.userService.ListRepositoryProviders(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list providers")
		return
	}

	responses := make([]*domainUser.RepositoryProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = p.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"providers": responses})
}

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
		switch {
		case errors.Is(err, user.ErrProviderAlreadyExists):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Provider already exists with this name")
		case errors.Is(err, user.ErrInvalidProviderType):
			apierr.InvalidInput(c, "Invalid provider type")
		default:
			apierr.InternalError(c, "Failed to create provider")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"provider": provider.ToResponse()})
}

func (h *UserRepositoryProviderHandler) GetProvider(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid provider ID")
		return
	}

	provider, err := h.userService.GetRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if errors.Is(err, user.ErrProviderNotFound) {
			apierr.ResourceNotFound(c, "Provider not found")
			return
		}
		apierr.InternalError(c, "Failed to get provider")
		return
	}

	c.JSON(http.StatusOK, gin.H{"provider": provider.ToResponse()})
}

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
		switch {
		case errors.Is(err, user.ErrProviderNotFound):
			apierr.ResourceNotFound(c, "Provider not found")
		case errors.Is(err, user.ErrProviderAlreadyExists):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Provider already exists with this name")
		default:
			apierr.InternalError(c, "Failed to update provider")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"provider": provider.ToResponse()})
}

func (h *UserRepositoryProviderHandler) DeleteProvider(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid provider ID")
		return
	}

	err = h.userService.DeleteRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if errors.Is(err, user.ErrProviderNotFound) {
			apierr.ResourceNotFound(c, "Provider not found")
			return
		}
		apierr.InternalError(c, "Failed to delete provider")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}
