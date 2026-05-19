package v1

import (
	"io"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *LicenseHandler) ActivateLicense(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	tenant, exists := c.Get("tenant")
	if exists {
		tc := tenant.(*middleware.TenantContext)
		if tc.UserRole != "owner" {
			apierr.ForbiddenOwner(c)
			return
		}
	}

	var req ActivateLicenseRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.LicenseData != "" {
		if err := h.licenseService.ActivateLicense(c.Request.Context(), []byte(req.LicenseData)); err != nil {
			apierr.ValidationError(c, err.Error())
			return
		}
	} else {
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

	status := h.licenseService.GetLicenseStatus()
	c.JSON(http.StatusOK, gin.H{
		"message": "license activated successfully",
		"status":  status,
	})
}

func (h *LicenseHandler) UploadLicense(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	tenant, exists := c.Get("tenant")
	if exists {
		tc := tenant.(*middleware.TenantContext)
		if tc.UserRole != "owner" {
			apierr.ForbiddenOwner(c)
			return
		}
	}

	file, err := c.FormFile("file")
	if err != nil {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "license file is required")
		return
	}

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

	licenseData, err := h.licenseService.ParseAndVerify([]byte(req.LicenseData))
	if err != nil {
		apierr.RespondWithExtra(c, http.StatusBadRequest, apierr.VALIDATION_FAILED, err.Error(), gin.H{"valid": false})
		return
	}

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
