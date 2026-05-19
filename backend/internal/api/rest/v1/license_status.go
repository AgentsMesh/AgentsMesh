package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *LicenseHandler) GetLicenseStatus(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	status := h.licenseService.GetLicenseStatus()
	c.JSON(http.StatusOK, status)
}

func (h *LicenseHandler) CheckFeature(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	feature := c.Query("feature")
	if feature == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "feature parameter is required")
		return
	}

	enabled := h.licenseService.HasFeature(feature)
	c.JSON(http.StatusOK, gin.H{
		"feature": feature,
		"enabled": enabled,
	})
}

func (h *LicenseHandler) GetLicenseLimits(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	licenseData := h.licenseService.GetCurrentLicense()
	if licenseData == nil {
		apierr.ResourceNotFound(c, "no active license")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"limits": licenseData.Limits,
		"plan":   licenseData.Plan,
	})
}
