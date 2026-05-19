package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/license"
	"github.com/gin-gonic/gin"
)

func RegisterLicenseHandlers(rg *gin.RouterGroup, licenseService *license.Service) {
	handler := NewLicenseHandler(licenseService)

	rg.GET("/status", handler.GetLicenseStatus)
	rg.GET("/limits", handler.GetLicenseLimits)
	rg.GET("/feature", handler.CheckFeature)

	rg.POST("/activate", handler.ActivateLicense)
	rg.POST("/upload", handler.UploadLicense)
	rg.POST("/refresh", handler.RefreshLicense)
	rg.POST("/validate", handler.ValidateLicense)
}

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
