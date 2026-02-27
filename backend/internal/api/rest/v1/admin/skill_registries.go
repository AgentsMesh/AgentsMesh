package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// SkillRegistryHandler handles admin skill registry requests
type SkillRegistryHandler struct {
	repo              extension.Repository
	marketplaceWorker *extensionservice.MarketplaceWorker
}

// NewSkillRegistryHandler creates a new skill registry handler
func NewSkillRegistryHandler(repo extension.Repository, worker *extensionservice.MarketplaceWorker) *SkillRegistryHandler {
	return &SkillRegistryHandler{
		repo:              repo,
		marketplaceWorker: worker,
	}
}

// RegisterRoutes registers skill registry admin routes
func (h *SkillRegistryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	skillRegistries := rg.Group("/skill-registries")
	{
		skillRegistries.GET("", h.List)
		skillRegistries.POST("", h.Create)
		skillRegistries.POST("/:id/sync", h.Sync)
		skillRegistries.DELETE("/:id", h.Delete)
	}
}

// List lists all platform-level skill registries
// GET /api/v1/admin/skill-registries
func (h *SkillRegistryHandler) List(c *gin.Context) {
	// nil orgID = platform-level registries
	registries, err := h.repo.ListSkillRegistries(c.Request.Context(), nil)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": registries,
		"total": len(registries),
	})
}

// CreateSkillRegistryRequest represents the create request body
type CreateSkillRegistryRequest struct {
	RepositoryURL string `json:"repository_url" binding:"required,url"`
	Branch        string `json:"branch"`
}

// Create creates a new platform-level skill registry
// POST /api/v1/admin/skill-registries
func (h *SkillRegistryHandler) Create(c *gin.Context) {
	var req CreateSkillRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	branch := req.Branch
	if branch == "" {
		branch = "main"
	}

	// Check for duplicate
	existing, _ := h.repo.FindSkillRegistryByURL(c.Request.Context(), nil, req.RepositoryURL)
	if existing != nil {
		apierr.Conflict(c, apierr.ALREADY_EXISTS, "platform skill registry with this URL already exists")
		return
	}

	registry := &extension.SkillRegistry{
		OrganizationID: nil, // platform-level
		RepositoryURL:  req.RepositoryURL,
		Branch:         branch,
		SourceType:     extension.SourceTypeAuto,
		SyncStatus:     extension.SyncStatusPending,
		IsActive:       true,
	}

	if err := h.repo.CreateSkillRegistry(c.Request.Context(), registry); err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	// Trigger an async sync if worker is available
	if h.marketplaceWorker != nil {
		registryID := registry.ID
		go func() {
			// Use a background context with timeout since the HTTP request will complete
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			_ = h.marketplaceWorker.SyncSingle(ctx, registryID)
		}()
	}

	c.JSON(http.StatusCreated, registry)
}

// Sync triggers a manual sync for a platform-level skill registry
// POST /api/v1/admin/skill-registries/:id/sync
func (h *SkillRegistryHandler) Sync(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

	// Verify registry exists and is platform-level
	registry, err := h.repo.GetSkillRegistry(c.Request.Context(), id)
	if err != nil {
		apierr.ResourceNotFound(c, "skill registry not found")
		return
	}

	if !registry.IsPlatformLevel() {
		apierr.InvalidInput(c, "not a platform-level skill registry")
		return
	}

	if h.marketplaceWorker == nil {
		apierr.InternalError(c, "marketplace worker not available")
		return
	}

	// Trigger sync synchronously so the caller can see the result
	if err := h.marketplaceWorker.SyncSingle(c.Request.Context(), id); err != nil {
		apierr.InternalError(c, "sync failed: "+err.Error())
		return
	}

	// Reload registry to get updated status
	registry, _ = h.repo.GetSkillRegistry(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"message":  "sync completed",
		"registry": registry,
	})
}

// Delete deletes a platform-level skill registry
// DELETE /api/v1/admin/skill-registries/:id
func (h *SkillRegistryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

	// Verify registry exists and is platform-level
	registry, err := h.repo.GetSkillRegistry(c.Request.Context(), id)
	if err != nil {
		apierr.ResourceNotFound(c, "skill registry not found")
		return
	}

	if !registry.IsPlatformLevel() {
		apierr.InvalidInput(c, "cannot delete non-platform-level skill registry via admin API")
		return
	}

	if err := h.repo.DeleteSkillRegistry(c.Request.Context(), id); err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
