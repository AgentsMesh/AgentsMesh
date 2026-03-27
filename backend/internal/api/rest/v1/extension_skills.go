package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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
