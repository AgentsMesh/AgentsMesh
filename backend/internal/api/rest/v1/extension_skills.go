package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// InstallSkillFromUpload installs a skill from an uploaded archive (multipart).
// Stays REST because Connect-RPC doesn't handle multipart/form-data.
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
