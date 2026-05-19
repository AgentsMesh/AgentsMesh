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

type SkillRegistryHandler struct {
	repo              extension.Repository
	marketplaceWorker *extensionservice.MarketplaceWorker
}

func NewSkillRegistryHandler(repo extension.Repository, worker *extensionservice.MarketplaceWorker) *SkillRegistryHandler {
	return &SkillRegistryHandler{
		repo:              repo,
		marketplaceWorker: worker,
	}
}

func (h *SkillRegistryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	skillRegistries := rg.Group("/skill-registries")
	{
		skillRegistries.GET("", h.List)
		skillRegistries.POST("", h.Create)
		skillRegistries.POST("/:id/sync", h.Sync)
		skillRegistries.DELETE("/:id", h.Delete)
	}
}

func (h *SkillRegistryHandler) List(c *gin.Context) {
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

type CreateSkillRegistryRequest struct {
	RepositoryURL string `json:"repository_url" binding:"required,url"`
	Branch        string `json:"branch"`
}

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

	if h.marketplaceWorker != nil {
		registryID := registry.ID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			_ = h.marketplaceWorker.SyncSingle(ctx, registryID)
		}()
	}

	c.JSON(http.StatusCreated, registry)
}

func (h *SkillRegistryHandler) Sync(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

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

	if err := h.marketplaceWorker.SyncSingle(c.Request.Context(), id); err != nil {
		apierr.InternalError(c, "sync failed: "+err.Error())
		return
	}

	registry, _ = h.repo.GetSkillRegistry(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"message":  "sync completed",
		"registry": registry,
	})
}

func (h *SkillRegistryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

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
