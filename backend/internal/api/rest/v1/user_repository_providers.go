package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

type RepositoryResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
	Visibility    string `json:"visibility"`
	HttpCloneURL  string `json:"http_clone_url"`
	SSHCloneURL   string `json:"ssh_clone_url"`
	WebURL        string `json:"web_url"`
}

type UserRepositoryProviderHandler struct {
	userService *user.Service
}

func NewUserRepositoryProviderHandler(userSvc *user.Service) *UserRepositoryProviderHandler {
	return &UserRepositoryProviderHandler{
		userService: userSvc,
	}
}

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

type CreateRepositoryProviderRequest struct {
	ProviderType string `json:"provider_type" binding:"required"`
	Name         string `json:"name" binding:"required"`
	BaseURL      string `json:"base_url" binding:"required"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	BotToken     string `json:"bot_token"`
}

type UpdateRepositoryProviderRequest struct {
	Name         *string `json:"name"`
	BaseURL      *string `json:"base_url"`
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
	BotToken     *string `json:"bot_token"`
	IsActive     *bool   `json:"is_active"`
}
