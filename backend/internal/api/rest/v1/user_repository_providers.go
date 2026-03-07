package v1

import (
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

// CreateRepositoryProviderRequest represents a request to create a repository provider
type CreateRepositoryProviderRequest struct {
	ProviderType string `json:"provider_type" binding:"required"`
	Name         string `json:"name" binding:"required"`
	BaseURL      string `json:"base_url" binding:"required"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	BotToken     string `json:"bot_token"`
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
