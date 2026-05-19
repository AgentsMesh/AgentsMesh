package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type AgentPodHandler struct {
	settingsService   *agentpod.SettingsService
	aiProviderService *agentpod.AIProviderService
}

func NewAgentPodHandler(settingsService *agentpod.SettingsService, aiProviderService *agentpod.AIProviderService) *AgentPodHandler {
	return &AgentPodHandler{
		settingsService:   settingsService,
		aiProviderService: aiProviderService,
	}
}

func (h *AgentPodHandler) GetSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)

	settings, err := h.settingsService.GetUserSettings(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to get settings")
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

type UpdateSettingsRequest struct {
	DefaultAgentSlug *string  `json:"default_agent_slug"`
	DefaultModel       *string `json:"default_model"`
	DefaultPermMode    *string `json:"default_perm_mode" binding:"omitempty,oneof=default accept-edits full-auto"`
	TerminalFontSize   *int    `json:"terminal_font_size" binding:"omitempty,min=8,max=32"`
	TerminalTheme      *string `json:"terminal_theme"`
}

func (h *AgentPodHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	updates := &agentpod.UserSettingsUpdate{
		DefaultAgentSlug: req.DefaultAgentSlug,
		DefaultModel:       req.DefaultModel,
		DefaultPermMode:    req.DefaultPermMode,
		TerminalFontSize:   req.TerminalFontSize,
		TerminalTheme:      req.TerminalTheme,
	}

	settings, err := h.settingsService.UpdateUserSettings(c.Request.Context(), userID, updates)
	if err != nil {
		apierr.InternalError(c, "Failed to update settings")
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

func (h *AgentPodHandler) ListProviders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providers, err := h.aiProviderService.GetUserProviders(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list providers")
		return
	}

	for _, p := range providers {
		p.EncryptedCredentials = ""
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

type CreateProviderRequest struct {
	ProviderType string            `json:"provider_type" binding:"required,oneof=claude gemini codex openai"`
	Name         string            `json:"name" binding:"required,min=1,max=100"`
	Credentials  map[string]string `json:"credentials" binding:"required"`
	IsDefault    bool              `json:"is_default"`
}

func (h *AgentPodHandler) CreateProvider(c *gin.Context) {
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if err := h.aiProviderService.ValidateCredentials(req.ProviderType, req.Credentials); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	provider, err := h.aiProviderService.CreateUserProvider(
		c.Request.Context(),
		userID,
		req.ProviderType,
		req.Name,
		req.Credentials,
		req.IsDefault,
	)
	if err != nil {
		apierr.InternalError(c, "Failed to create provider")
		return
	}

	provider.EncryptedCredentials = ""

	c.JSON(http.StatusCreated, gin.H{"provider": provider})
}

type UpdateProviderRequest struct {
	Name        string            `json:"name" binding:"omitempty,min=1,max=100"`
	Credentials map[string]string `json:"credentials"`
	IsDefault   *bool             `json:"is_default"`
	IsEnabled   *bool             `json:"is_enabled"`
}

func (h *AgentPodHandler) UpdateProvider(c *gin.Context) {
	idStr := c.Param("id")
	providerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid provider ID")
		return
	}

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	provider, err := h.aiProviderService.UpdateUserProvider(
		c.Request.Context(),
		providerID,
		req.Name,
		req.Credentials,
		isDefault,
		isEnabled,
	)
	if err != nil {
		if errors.Is(err, agentpod.ErrProviderNotFound) {
			apierr.ResourceNotFound(c, "Provider not found")
			return
		}
		apierr.InternalError(c, "Failed to update provider")
		return
	}

	provider.EncryptedCredentials = ""

	c.JSON(http.StatusOK, gin.H{"provider": provider})
}

func (h *AgentPodHandler) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	providerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid provider ID")
		return
	}

	if err := h.aiProviderService.DeleteUserProvider(c.Request.Context(), providerID); err != nil {
		apierr.InternalError(c, "Failed to delete provider")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}

func (h *AgentPodHandler) SetDefaultProvider(c *gin.Context) {
	idStr := c.Param("id")
	providerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid provider ID")
		return
	}

	if err := h.aiProviderService.SetDefaultProvider(c.Request.Context(), providerID); err != nil {
		if errors.Is(err, agentpod.ErrProviderNotFound) {
			apierr.ResourceNotFound(c, "Provider not found")
			return
		}
		apierr.InternalError(c, "Failed to set default provider")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default provider set"})
}
