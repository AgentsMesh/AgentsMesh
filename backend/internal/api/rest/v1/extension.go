package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
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
	CompatibleAgents []string `json:"compatible_agents"` // agent type whitelist, e.g. ["claude-code"]
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

// --- Repo Skills ---

// ListRepoSkills lists installed skills for a repository
// GET /api/v1/organizations/:slug/repositories/:id/skills?scope=org|user|all
func (h *ExtensionHandler) ListRepoSkills(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	tenant := middleware.GetTenant(c)
	scope := c.DefaultQuery("scope", "all")

	skills, err := h.extensionSvc.ListRepoSkills(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, scope)
	if err != nil {
		handleServiceError(c, err, "Failed to list repository skills")
		return
	}

	c.JSON(http.StatusOK, gin.H{"skills": skills})
}

// InstallSkillFromMarketRequest represents a market skill installation request
type InstallSkillFromMarketRequest struct {
	MarketItemID int64  `json:"market_item_id" binding:"required"`
	Scope        string `json:"scope" binding:"required"`
}

// InstallSkillFromMarket installs a skill from the marketplace
// POST /api/v1/organizations/:slug/repositories/:id/skills/install-from-market
func (h *ExtensionHandler) InstallSkillFromMarket(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req InstallSkillFromMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Org-scope installations require admin/owner role
	if req.Scope == "org" {
		if !requireOrgAdmin(c) {
			return
		}
	}

	tenant := middleware.GetTenant(c)

	skill, err := h.extensionSvc.InstallSkillFromMarket(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, req.MarketItemID, req.Scope)
	if err != nil {
		handleServiceError(c, err, "Failed to install skill from market")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"skill": skill})
}

// InstallSkillFromGitHubRequest represents a GitHub skill installation request
type InstallSkillFromGitHubRequest struct {
	URL    string `json:"url" binding:"required"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Scope  string `json:"scope" binding:"required"`
}

// InstallSkillFromGitHub installs a skill from a GitHub URL
// POST /api/v1/organizations/:slug/repositories/:id/skills/install-from-github
func (h *ExtensionHandler) InstallSkillFromGitHub(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req InstallSkillFromGitHubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Org-scope installations require admin/owner role
	if req.Scope == "org" {
		if !requireOrgAdmin(c) {
			return
		}
	}

	tenant := middleware.GetTenant(c)

	skill, err := h.extensionSvc.InstallSkillFromGitHub(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, req.URL, req.Branch, req.Path, req.Scope)
	if err != nil {
		handleServiceError(c, err, "Failed to install skill from GitHub")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"skill": skill})
}

// InstallSkillFromUpload installs a skill from an uploaded archive
// POST /api/v1/organizations/:slug/repositories/:id/skills/install-from-upload
func (h *ExtensionHandler) InstallSkillFromUpload(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		apierr.ValidationError(c, "File is required")
		return
	}

	// Enforce upload size limit
	if file.Size > maxSkillUploadSize {
		apierr.PayloadTooLarge(c, "File too large, maximum 50MB")
		return
	}

	scope := c.PostForm("scope")
	if scope == "" {
		apierr.ValidationError(c, "Scope is required")
		return
	}

	// Org-scope installations require admin/owner role
	if scope == "org" {
		if !requireOrgAdmin(c) {
			return
		}
	}

	tenant := middleware.GetTenant(c)

	f, err := file.Open()
	if err != nil {
		apierr.InternalError(c, "Failed to open uploaded file")
		return
	}
	defer f.Close()

	skill, err := h.extensionSvc.InstallSkillFromUpload(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, f, file.Filename, scope)
	if err != nil {
		handleServiceError(c, err, "Failed to install skill from upload")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"skill": skill})
}

// UpdateSkillRequest represents a skill update request
type UpdateSkillRequest struct {
	IsEnabled     *bool `json:"is_enabled"`
	PinnedVersion *int  `json:"pinned_version"`
}

// UpdateSkill updates an installed skill
// PUT /api/v1/organizations/:slug/repositories/:id/skills/:installId
func (h *ExtensionHandler) UpdateSkill(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	installID, err := strconv.ParseInt(c.Param("installId"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid install ID")
		return
	}

	var req UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	skill, err := h.extensionSvc.UpdateSkill(c.Request.Context(), tenant.OrganizationID, repoID, installID, tenant.UserID, tenant.UserRole, req.IsEnabled, req.PinnedVersion)
	if err != nil {
		handleServiceError(c, err, "Failed to update skill")
		return
	}

	c.JSON(http.StatusOK, gin.H{"skill": skill})
}

// UninstallSkill removes an installed skill
// DELETE /api/v1/organizations/:slug/repositories/:id/skills/:installId
func (h *ExtensionHandler) UninstallSkill(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	installID, err := strconv.ParseInt(c.Param("installId"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid install ID")
		return
	}

	tenant := middleware.GetTenant(c)

	if err := h.extensionSvc.UninstallSkill(c.Request.Context(), tenant.OrganizationID, repoID, installID, tenant.UserID, tenant.UserRole); err != nil {
		handleServiceError(c, err, "Failed to uninstall skill")
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Repo MCP Servers ---

// ListRepoMcpServers lists installed MCP servers for a repository
// GET /api/v1/organizations/:slug/repositories/:id/mcp-servers?scope=org|user|all
func (h *ExtensionHandler) ListRepoMcpServers(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	tenant := middleware.GetTenant(c)
	scope := c.DefaultQuery("scope", "all")

	servers, err := h.extensionSvc.ListRepoMcpServers(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, scope)
	if err != nil {
		handleServiceError(c, err, "Failed to list repository MCP servers")
		return
	}

	c.JSON(http.StatusOK, gin.H{"mcp_servers": servers})
}

// InstallMcpFromMarketRequest represents a market MCP server installation request
type InstallMcpFromMarketRequest struct {
	MarketItemID int64             `json:"market_item_id" binding:"required"`
	EnvVars      map[string]string `json:"env_vars"`
	Scope        string            `json:"scope" binding:"required"`
}

// InstallMcpFromMarket installs an MCP server from a marketplace template
// POST /api/v1/organizations/:slug/repositories/:id/mcp-servers/install-from-market
func (h *ExtensionHandler) InstallMcpFromMarket(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req InstallMcpFromMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Org-scope installations require admin/owner role
	if req.Scope == "org" {
		if !requireOrgAdmin(c) {
			return
		}
	}

	tenant := middleware.GetTenant(c)

	server, err := h.extensionSvc.InstallMcpFromMarket(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, req.MarketItemID, req.EnvVars, req.Scope)
	if err != nil {
		handleServiceError(c, err, "Failed to install MCP server from market")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mcp_server": server})
}

// InstallCustomMcpServerRequest represents a custom MCP server installation request
type InstallCustomMcpServerRequest struct {
	Name          string            `json:"name" binding:"required"`
	Slug          string            `json:"slug" binding:"required"`
	TransportType string            `json:"transport_type" binding:"required"`
	Command       string            `json:"command"`
	Args          json.RawMessage   `json:"args"`
	HttpURL       string            `json:"http_url"`
	HttpHeaders   json.RawMessage   `json:"http_headers"`
	EnvVars       map[string]string `json:"env_vars"`
	Scope         string            `json:"scope" binding:"required"`
}

// InstallCustomMcpServer installs a custom MCP server
// POST /api/v1/organizations/:slug/repositories/:id/mcp-servers/install-custom
func (h *ExtensionHandler) InstallCustomMcpServer(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req InstallCustomMcpServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Validate transport_type
	switch req.TransportType {
	case "stdio", "http", "sse":
		// valid
	default:
		apierr.ValidationError(c, "transport_type must be 'stdio', 'http', or 'sse'")
		return
	}

	// Validate Args is a valid JSON array (if provided)
	if len(req.Args) > 0 {
		var args []interface{}
		if err := json.Unmarshal(req.Args, &args); err != nil {
			apierr.ValidationError(c, "args must be a valid JSON array")
			return
		}
		const maxArgs = 50
		if len(args) > maxArgs {
			apierr.ValidationError(c, "args exceeds maximum of 50 entries")
			return
		}
	}

	// Validate HttpHeaders is a valid JSON object (if provided)
	if len(req.HttpHeaders) > 0 {
		var headers map[string]interface{}
		if err := json.Unmarshal(req.HttpHeaders, &headers); err != nil {
			apierr.ValidationError(c, "http_headers must be a valid JSON object")
			return
		}
		const maxHeaders = 20
		if len(headers) > maxHeaders {
			apierr.ValidationError(c, "http_headers exceeds maximum of 20 entries")
			return
		}
	}

	// Org-scope installations require admin/owner role
	if req.Scope == "org" {
		if !requireOrgAdmin(c) {
			return
		}
	}

	tenant := middleware.GetTenant(c)

	server := &extension.InstalledMcpServer{
		Name:          req.Name,
		Slug:          req.Slug,
		Scope:         req.Scope,
		TransportType: req.TransportType,
		Command:       req.Command,
		Args:          req.Args,
		HttpURL:       req.HttpURL,
		HttpHeaders:   req.HttpHeaders,
	}

	result, err := h.extensionSvc.InstallCustomMcpServer(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, server, req.EnvVars)
	if err != nil {
		handleServiceError(c, err, "Failed to install custom MCP server")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mcp_server": result})
}

// UpdateMcpServerRequest represents an MCP server update request
type UpdateMcpServerRequest struct {
	IsEnabled *bool             `json:"is_enabled"`
	EnvVars   map[string]string `json:"env_vars"`
}

// UpdateMcpServer updates an installed MCP server
// PUT /api/v1/organizations/:slug/repositories/:id/mcp-servers/:installId
func (h *ExtensionHandler) UpdateMcpServer(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	installID, err := strconv.ParseInt(c.Param("installId"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid install ID")
		return
	}

	var req UpdateMcpServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	server, err := h.extensionSvc.UpdateMcpServer(c.Request.Context(), tenant.OrganizationID, repoID, installID, tenant.UserID, tenant.UserRole, req.IsEnabled, req.EnvVars)
	if err != nil {
		handleServiceError(c, err, "Failed to update MCP server")
		return
	}

	c.JSON(http.StatusOK, gin.H{"mcp_server": server})
}

// UninstallMcpServer removes an installed MCP server
// DELETE /api/v1/organizations/:slug/repositories/:id/mcp-servers/:installId
func (h *ExtensionHandler) UninstallMcpServer(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	installID, err := strconv.ParseInt(c.Param("installId"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid install ID")
		return
	}

	tenant := middleware.GetTenant(c)

	if err := h.extensionSvc.UninstallMcpServer(c.Request.Context(), tenant.OrganizationID, repoID, installID, tenant.UserID, tenant.UserRole); err != nil {
		handleServiceError(c, err, "Failed to uninstall MCP server")
		return
	}

	c.Status(http.StatusNoContent)
}
