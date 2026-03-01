package v1

import (
	"io"
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ActivateLicense activates a new license
// @Summary Activate license
// @Description Activates a new license for OnPremise deployments
// @Tags license
// @Accept json
// @Produce json
// @Param request body ActivateLicenseRequest true "License activation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/license/activate [post]
func (h *LicenseHandler) ActivateLicense(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	// Check if user is system admin (for multi-tenant, only admins can activate licenses)
	tenant, exists := c.Get("tenant")
	if exists {
		tc := tenant.(*middleware.TenantContext)
		if tc.UserRole != "owner" {
			apierr.ForbiddenOwner(c)
			return
		}
	}

	// Try to parse JSON request first
	var req ActivateLicenseRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.LicenseData != "" {
		if err := h.licenseService.ActivateLicense(c.Request.Context(), []byte(req.LicenseData)); err != nil {
			apierr.ValidationError(c, err.Error())
			return
		}
	} else {
		// Fall back to reading raw body (for file upload)
		c.Request.Body = io.NopCloser(c.Request.Body)
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "failed to read request body")
			return
		}

		if len(body) == 0 {
			apierr.BadRequest(c, apierr.MISSING_REQUIRED, "license data is required")
			return
		}

		if err := h.licenseService.ActivateLicense(c.Request.Context(), body); err != nil {
			apierr.ValidationError(c, err.Error())
			return
		}
	}

	// Return updated status
	status := h.licenseService.GetLicenseStatus()
	c.JSON(http.StatusOK, gin.H{
		"message": "license activated successfully",
		"status":  status,
	})
}

// UploadLicense handles license file upload
// @Summary Upload license file
// @Description Uploads and activates a license file for OnPremise deployments
// @Tags license
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "License file"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/v1/license/upload [post]
func (h *LicenseHandler) UploadLicense(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	// Check permissions
	tenant, exists := c.Get("tenant")
	if exists {
		tc := tenant.(*middleware.TenantContext)
		if tc.UserRole != "owner" {
			apierr.ForbiddenOwner(c)
			return
		}
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "license file is required")
		return
	}

	// Open and read file
	f, err := file.Open()
	if err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "failed to open license file")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "failed to read license file")
		return
	}

	// Activate license
	if err := h.licenseService.ActivateLicense(c.Request.Context(), data); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	status := h.licenseService.GetLicenseStatus()
	c.JSON(http.StatusOK, gin.H{
		"message": "license activated successfully",
		"status":  status,
	})
}

// RefreshLicense reloads the license from file
// @Summary Refresh license
// @Description Reloads the license from the configured file path
// @Tags license
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/v1/license/refresh [post]
func (h *LicenseHandler) RefreshLicense(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	if err := h.licenseService.RefreshLicense(); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	status := h.licenseService.GetLicenseStatus()
	c.JSON(http.StatusOK, gin.H{
		"message": "license refreshed successfully",
		"status":  status,
	})
}

// ValidateLicense validates license data without activating
// @Summary Validate license
// @Description Validates license data and returns parsed information without activating
// @Tags license
// @Accept json
// @Produce json
// @Param request body ValidateLicenseRequest true "License validation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/v1/license/validate [post]
func (h *LicenseHandler) ValidateLicense(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	var req ValidateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Parse and verify without activating
	licenseData, err := h.licenseService.ParseAndVerify([]byte(req.LicenseData))
	if err != nil {
		apierr.RespondWithExtra(c, http.StatusBadRequest, apierr.VALIDATION_FAILED, err.Error(), gin.H{"valid": false})
		return
	}

	// Return parsed license info (without signature)
	c.JSON(http.StatusOK, gin.H{
		"valid":             true,
		"license_key":       licenseData.LicenseKey,
		"organization_name": licenseData.OrganizationName,
		"contact_email":     licenseData.ContactEmail,
		"plan":              licenseData.Plan,
		"limits":            licenseData.Limits,
		"features":          licenseData.Features,
		"issued_at":         licenseData.IssuedAt,
		"expires_at":        licenseData.ExpiresAt,
	})
}
