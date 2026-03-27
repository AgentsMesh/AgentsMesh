package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// maxSkillUploadSize is the maximum file size allowed for skill uploads (50MB)
const maxSkillUploadSize = 50 << 20

// handleServiceError maps service-layer errors to appropriate HTTP error responses
func handleServiceError(c *gin.Context, err error, fallbackMsg string) {
	switch {
	case errors.Is(err, extensionservice.ErrNotFound):
		apierr.ResourceNotFound(c, err.Error())
	case errors.Is(err, extensionservice.ErrForbidden):
		apierr.ForbiddenAdmin(c)
	case errors.Is(err, extensionservice.ErrInvalidScope), errors.Is(err, extensionservice.ErrInvalidInput):
		apierr.ValidationError(c, err.Error())
	case errors.Is(err, extensionservice.ErrAlreadyInstalled):
		apierr.Conflict(c, apierr.ALREADY_EXISTS, err.Error())
	default:
		apierr.InternalError(c, fallbackMsg)
	}
}

// requireOrgAdmin checks if the current user has admin or owner role.
// Returns true if the user is authorized, false otherwise (and sends 403 response).
func requireOrgAdmin(c *gin.Context) bool {
	tenant := middleware.GetTenant(c)
	if tenant.UserRole != "admin" && tenant.UserRole != "owner" {
		apierr.ForbiddenAdmin(c)
		return false
	}
	return true
}

// ExtensionHandler handles extension-related API endpoints
type ExtensionHandler struct {
	extensionSvc *extensionservice.Service
}

// NewExtensionHandler creates a new extension handler
func NewExtensionHandler(extensionSvc *extensionservice.Service) *ExtensionHandler {
	return &ExtensionHandler{
		extensionSvc: extensionSvc,
	}
}

// --- Skill Registries (org admin only) ---

// ListSkillRegistries lists skill registries for the organization
// GET /api/v1/organizations/:slug/skill-registries
func (h *ExtensionHandler) ListSkillRegistries(c *gin.Context) {
	if !requireOrgAdmin(c) {
		return
	}

	tenant := middleware.GetTenant(c)

	registries, err := h.extensionSvc.ListSkillRegistries(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		handleServiceError(c, err, "Failed to list skill registries")
		return
	}

	c.JSON(http.StatusOK, gin.H{"skill_registries": registries})
}

// CreateSkillRegistryRequest represents a skill registry creation request
type CreateSkillRegistryRequest struct {
	RepositoryURL    string   `json:"repository_url" binding:"required,url"`
	Branch           string   `json:"branch"`
	SourceType       string   `json:"source_type"`
	CompatibleAgents []string `json:"compatible_agents"` // agent whitelist, e.g. ["claude-code"]
	AuthType         string   `json:"auth_type"`         // none / github_pat / gitlab_pat / ssh_key
	AuthCredential   string   `json:"auth_credential"`   // PAT or SSH key (encrypted at service layer)
}

// CreateSkillRegistry creates a new skill registry
// POST /api/v1/organizations/:slug/skill-registries
func (h *ExtensionHandler) CreateSkillRegistry(c *gin.Context) {
	if !requireOrgAdmin(c) {
		return
	}

	var req CreateSkillRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	input := extensionservice.CreateSkillRegistryInput{
		RepositoryURL:    req.RepositoryURL,
		Branch:           req.Branch,
		SourceType:       req.SourceType,
		CompatibleAgents: req.CompatibleAgents,
		AuthType:         req.AuthType,
		AuthCredential:   req.AuthCredential,
	}

	registry, err := h.extensionSvc.CreateSkillRegistry(c.Request.Context(), tenant.OrganizationID, input)
	if err != nil {
		handleServiceError(c, err, "Failed to create skill registry")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"registry": registry})
}

// SyncSkillRegistry triggers a sync for a skill registry
// POST /api/v1/organizations/:slug/skill-registries/:id/sync
func (h *ExtensionHandler) SyncSkillRegistry(c *gin.Context) {
	registryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid skill registry ID")
		return
	}

	if !requireOrgAdmin(c) {
		return
	}

	tenant := middleware.GetTenant(c)

	registry, err := h.extensionSvc.SyncSkillRegistry(c.Request.Context(), tenant.OrganizationID, registryID)
	if err != nil {
		handleServiceError(c, err, "Failed to sync skill registry")
		return
	}

	c.JSON(http.StatusOK, gin.H{"registry": registry})
}

// DeleteSkillRegistry deletes a skill registry
// DELETE /api/v1/organizations/:slug/skill-registries/:id
func (h *ExtensionHandler) DeleteSkillRegistry(c *gin.Context) {
	registryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid skill registry ID")
		return
	}

	if !requireOrgAdmin(c) {
		return
	}

	tenant := middleware.GetTenant(c)

	if err := h.extensionSvc.DeleteSkillRegistry(c.Request.Context(), tenant.OrganizationID, registryID); err != nil {
		handleServiceError(c, err, "Failed to delete skill registry")
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Skill Registry Overrides ---

// TogglePlatformRegistryRequest represents a request to toggle a platform registry
type TogglePlatformRegistryRequest struct {
	Disabled bool `json:"disabled"`
}

// TogglePlatformRegistry toggles a platform-level skill registry for the organization
// PUT /api/v1/organizations/:slug/skill-registries/:id/toggle
func (h *ExtensionHandler) TogglePlatformRegistry(c *gin.Context) {
	registryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid skill registry ID")
		return
	}

	if !requireOrgAdmin(c) {
		return
	}

	var req TogglePlatformRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	if err := h.extensionSvc.TogglePlatformRegistry(c.Request.Context(), tenant.OrganizationID, registryID, req.Disabled); err != nil {
		handleServiceError(c, err, "Failed to toggle platform registry")
		return
	}

	// Return updated overrides list
	overrides, err := h.extensionSvc.ListSkillRegistryOverrides(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		handleServiceError(c, err, "Failed to list overrides")
		return
	}

	c.JSON(http.StatusOK, gin.H{"overrides": overrides})
}

// ListSkillRegistryOverrides returns all skill registry overrides for the organization
// GET /api/v1/organizations/:slug/skill-registry-overrides
func (h *ExtensionHandler) ListSkillRegistryOverrides(c *gin.Context) {
	if !requireOrgAdmin(c) {
		return
	}

	tenant := middleware.GetTenant(c)

	overrides, err := h.extensionSvc.ListSkillRegistryOverrides(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		handleServiceError(c, err, "Failed to list skill registry overrides")
		return
	}

	c.JSON(http.StatusOK, gin.H{"overrides": overrides})
}

// --- Marketplace ---

// ListMarketSkills returns skills available in the marketplace
// GET /api/v1/organizations/:slug/market/skills?q=&category=
func (h *ExtensionHandler) ListMarketSkills(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	query := c.Query("q")
	category := c.Query("category")

	skills, err := h.extensionSvc.ListMarketSkills(c.Request.Context(), tenant.OrganizationID, query, category)
	if err != nil {
		handleServiceError(c, err, "Failed to list market skills")
		return
	}

	c.JSON(http.StatusOK, gin.H{"skills": skills})
}

// ListMarketMcpServers returns MCP server templates from the marketplace
// GET /api/v1/organizations/:slug/market/mcp-servers?q=&category=&limit=50&offset=0
func (h *ExtensionHandler) ListMarketMcpServers(c *gin.Context) {
	query := c.Query("q")
	category := c.Query("category")

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	servers, total, err := h.extensionSvc.ListMarketMcpServers(c.Request.Context(), query, category, limit, offset)
	if err != nil {
		handleServiceError(c, err, "Failed to list market MCP servers")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mcp_servers": servers,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

