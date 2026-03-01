package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// GetLicenseStatus returns the current license status
// @Summary Get license status
// @Description Returns the current license status for OnPremise deployments
// @Tags license
// @Produce json
// @Success 200 {object} billing.LicenseStatus
// @Router /api/v1/license/status [get]
func (h *LicenseHandler) GetLicenseStatus(c *gin.Context) {
	if h.licenseService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "license service not configured")
		return
	}

	status := h.licenseService.GetLicenseStatus()
	c.JSON(http.StatusOK, status)
}

// CheckFeature checks if a specific feature is enabled
// @Summary Check feature
// @Description Checks if a specific feature is enabled in the current license
// @Tags license
// @Produce json
// @Param feature query string true "Feature name"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/license/feature [get]
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

// GetLicenseLimits returns the current license limits
// @Summary Get license limits
// @Description Returns the resource limits defined in the current license
// @Tags license
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/license/limits [get]
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
