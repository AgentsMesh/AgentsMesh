package v1

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/service/license"
)

type LicenseHandler struct {
	licenseService *license.Service
}

func NewLicenseHandler(licenseService *license.Service) *LicenseHandler {
	return &LicenseHandler{licenseService: licenseService}
}

type ActivateLicenseRequest struct {
	LicenseData string `json:"license_data"` // Base64 encoded or raw JSON license data
}

type ValidateLicenseRequest struct {
	LicenseData string `json:"license_data" binding:"required"`
}

func (r *ActivateLicenseRequest) UnmarshalJSON(data []byte) error {
	type Alias ActivateLicenseRequest
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err == nil {
		return nil
	}

	r.LicenseData = string(data)
	return nil
}
