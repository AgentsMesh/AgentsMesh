package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/license"
	"github.com/gin-gonic/gin"
)

// RegisterLicenseHandlers registers license routes
func RegisterLicenseHandlers(rg *gin.RouterGroup, licenseService *license.Service) {
	handler := NewLicenseHandler(licenseService)

	// Public endpoints (no auth required for status)
	rg.GET("/status", handler.GetLicenseStatus)
	rg.GET("/limits", handler.GetLicenseLimits)
	rg.GET("/feature", handler.CheckFeature)

	// Protected endpoints
	rg.POST("/activate", handler.ActivateLicense)
	rg.POST("/upload", handler.UploadLicense)
	rg.POST("/refresh", handler.RefreshLicense)
	rg.POST("/validate", handler.ValidateLicense)
}

// LicenseStatusResponse represents the license status response for API documentation
type LicenseStatusResponse struct {
	IsActive         bool     `json:"is_active"`
	LicenseKey       string   `json:"license_key,omitempty"`
	OrganizationName string   `json:"organization_name,omitempty"`
	Plan             string   `json:"plan,omitempty"`
	ExpiresAt        string   `json:"expires_at,omitempty"`
	MaxUsers         int      `json:"max_users,omitempty"`
	MaxRunners       int      `json:"max_runners,omitempty"`
	MaxRepositories  int      `json:"max_repositories,omitempty"`
	MaxPodMinutes    int      `json:"max_pod_minutes,omitempty"`
	Features         []string `json:"features,omitempty"`
	Message          string   `json:"message"`
}
