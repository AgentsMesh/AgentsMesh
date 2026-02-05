package v1

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/service/license"
)

// LicenseHandler handles license-related HTTP requests
type LicenseHandler struct {
	licenseService *license.Service
}

// NewLicenseHandler creates a new license handler
func NewLicenseHandler(licenseService *license.Service) *LicenseHandler {
	return &LicenseHandler{licenseService: licenseService}
}

// ActivateLicenseRequest represents the license activation request
type ActivateLicenseRequest struct {
	LicenseData string `json:"license_data"` // Base64 encoded or raw JSON license data
}

// ValidateLicenseRequest represents a license validation request
type ValidateLicenseRequest struct {
	LicenseData string `json:"license_data" binding:"required"`
}

// UnmarshalJSON implements custom JSON unmarshaling for license activation
func (r *ActivateLicenseRequest) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as JSON object first
	type Alias ActivateLicenseRequest
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err == nil {
		return nil
	}

	// If that fails, treat the entire body as license data
	r.LicenseData = string(data)
	return nil
}
