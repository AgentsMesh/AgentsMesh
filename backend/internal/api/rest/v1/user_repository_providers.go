package v1

import (
	"net/http"
	"strconv"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// RepositoryResponse represents a repository in API responses
type RepositoryResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	FullPath      string `json:"full_path"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
	Visibility    string `json:"visibility"`
	CloneURL      string `json:"clone_url"`
	SSHCloneURL   string `json:"ssh_clone_url"`
	WebURL        string `json:"web_url"`
}

// UserRepositoryProviderHandler handles user repository provider requests
type UserRepositoryProviderHandler struct {
	userService *user.Service
}

// NewUserRepositoryProviderHandler creates a new user repository provider handler
func NewUserRepositoryProviderHandler(userSvc *user.Service) *UserRepositoryProviderHandler {
	return &UserRepositoryProviderHandler{
		userService: userSvc,
	}
}

// RegisterRoutes registers user repository provider routes
// Note: rg is already prefixed with /users, so we use /repository-providers
// Final path: /api/v1/users/repository-providers
func (h *UserRepositoryProviderHandler) RegisterRoutes(rg *gin.RouterGroup) {
	providers := rg.Group("/repository-providers")
	{
		providers.GET("", h.ListProviders)
		providers.POST("", h.CreateProvider)
		providers.GET("/:id", h.GetProvider)
		providers.PUT("/:id", h.UpdateProvider)
		providers.DELETE("/:id", h.DeleteProvider)
		providers.POST("/:id/default", h.SetDefault)
		providers.POST("/:id/test", h.TestConnection)
		providers.GET("/:id/repositories", h.ListRepositories)
	}
}

// ListProviders lists all repository providers for the current user
// GET /api/v1/user/repository-providers
func (h *UserRepositoryProviderHandler) ListProviders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providers, err := h.userService.ListRepositoryProviders(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list providers"})
		return
	}

	// Convert to response format
	responses := make([]*domainUser.RepositoryProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = p.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"providers": responses})
}

// CreateRepositoryProviderRequest represents a request to create a repository provider
type CreateRepositoryProviderRequest struct {
	ProviderType string `json:"provider_type" binding:"required"`
	Name         string `json:"name" binding:"required"`
	BaseURL      string `json:"base_url" binding:"required"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	BotToken     string `json:"bot_token"`
}

// CreateProvider creates a new repository provider
// POST /api/v1/user/repository-providers
func (h *UserRepositoryProviderHandler) CreateProvider(c *gin.Context) {
	var req CreateRepositoryProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusConflict, gin.H{"error": "Provider already exists with this name"})
		case user.ErrInvalidProviderType:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider type"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	provider, err := h.userService.GetRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"provider": provider.ToResponse()})
}

// UpdateRepositoryProviderRequest represents a request to update a repository provider
type UpdateRepositoryProviderRequest struct {
	Name         *string `json:"name"`
	BaseURL      *string `json:"base_url"`
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
	BotToken     *string `json:"bot_token"`
	IsActive     *bool   `json:"is_active"`
}

// UpdateProvider updates a repository provider
// PUT /api/v1/user/repository-providers/:id
func (h *UserRepositoryProviderHandler) UpdateProvider(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var req UpdateRepositoryProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		case user.ErrProviderAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Provider already exists with this name"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	err = h.userService.DeleteRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}

// SetDefault sets a repository provider as default
// POST /api/v1/user/repository-providers/:id/default
func (h *UserRepositoryProviderHandler) SetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	err = h.userService.SetDefaultRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default provider set"})
}

// TestConnection tests the connection to a repository provider
// POST /api/v1/user/repository-providers/:id/test
func (h *UserRepositoryProviderHandler) TestConnection(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	// Get provider
	provider, err := h.userService.GetRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get provider"})
		return
	}

	// Get decrypted token
	token, err := h.userService.GetDecryptedProviderToken(c.Request.Context(), userID, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt token"})
		return
	}

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No token configured for this provider"})
		return
	}

	// Create git provider and test connection
	gitProvider, err := git.NewProvider(provider.ProviderType, provider.BaseURL, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create git provider: " + err.Error()})
		return
	}

	// Try to list projects to verify connection
	_, err = gitProvider.ListProjects(c.Request.Context(), 1, 1)
	if err != nil {
		if err == git.ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authentication failed - token may be invalid or expired",
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"error":   "Connection failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection successful",
	})
}

// ListRepositories lists repositories accessible through a repository provider
// GET /api/v1/user/repository-providers/:id/repositories
func (h *UserRepositoryProviderHandler) ListRepositories(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Get provider
	provider, err := h.userService.GetRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get provider"})
		return
	}

	// Get decrypted token
	token, err := h.userService.GetDecryptedProviderToken(c.Request.Context(), userID, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt token"})
		return
	}

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No token configured for this provider"})
		return
	}

	// Create git provider
	gitProvider, err := git.NewProvider(provider.ProviderType, provider.BaseURL, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create git provider: " + err.Error()})
		return
	}

	// Fetch repositories
	var projects []*git.Project
	if search != "" {
		projects, err = gitProvider.SearchProjects(c.Request.Context(), search, page, perPage)
	} else {
		projects, err = gitProvider.ListProjects(c.Request.Context(), page, perPage)
	}

	if err != nil {
		if err == git.ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Access token is invalid or expired"})
			return
		}
		if err == git.ErrRateLimited {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limited by git provider"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch repositories: " + err.Error()})
		return
	}

	// Convert to response format
	repositories := make([]*RepositoryResponse, len(projects))
	for i, p := range projects {
		repositories[i] = &RepositoryResponse{
			ID:            p.ID,
			Name:          p.Name,
			FullPath:      p.FullPath,
			Description:   p.Description,
			DefaultBranch: p.DefaultBranch,
			Visibility:    p.Visibility,
			CloneURL:      p.CloneURL,
			SSHCloneURL:   p.SSHCloneURL,
			WebURL:        p.WebURL,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": repositories,
		"page":         page,
		"per_page":     perPage,
	})
}
