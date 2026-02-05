package license

import (
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// GetCurrentLicense returns the current active license
func (s *Service) GetCurrentLicense() *LicenseData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentLicense
}

// IsLicenseValid checks if the current license is valid
func (s *Service) IsLicenseValid() bool {
	license := s.GetCurrentLicense()
	if license == nil {
		return false
	}
	return time.Now().Before(license.ExpiresAt)
}

// GetLicenseStatus returns the current license status
func (s *Service) GetLicenseStatus() *billing.LicenseStatus {
	license := s.GetCurrentLicense()

	status := &billing.LicenseStatus{
		IsActive: false,
	}

	if license == nil {
		status.Message = "No license installed"
		return status
	}

	if time.Now().After(license.ExpiresAt) {
		status.IsActive = false
		status.Message = "License has expired"
		status.ExpiresAt = &license.ExpiresAt
		return status
	}

	status.IsActive = true
	status.LicenseKey = license.LicenseKey
	status.OrganizationName = license.OrganizationName
	status.Plan = license.Plan
	status.ExpiresAt = &license.ExpiresAt
	status.MaxUsers = license.Limits.MaxUsers
	status.MaxRunners = license.Limits.MaxRunners
	status.MaxRepositories = license.Limits.MaxRepositories
	status.MaxPodMinutes = license.Limits.MaxPodMinutes
	status.Features = license.Features

	daysUntilExpiry := int(time.Until(license.ExpiresAt).Hours() / 24)
	if daysUntilExpiry <= 30 {
		status.Message = fmt.Sprintf("License expires in %d days", daysUntilExpiry)
	} else {
		status.Message = "License is active"
	}

	return status
}

// CheckLimits checks if the current usage is within license limits
func (s *Service) CheckLimits(users, runners, repositories, podMinutes int) error {
	license := s.GetCurrentLicense()
	if license == nil {
		return fmt.Errorf("no active license")
	}

	if !time.Now().Before(license.ExpiresAt) {
		return fmt.Errorf("license has expired")
	}

	limits := license.Limits

	if limits.MaxUsers != -1 && users > limits.MaxUsers {
		return fmt.Errorf("user limit exceeded: %d/%d", users, limits.MaxUsers)
	}

	if limits.MaxRunners != -1 && runners > limits.MaxRunners {
		return fmt.Errorf("runner limit exceeded: %d/%d", runners, limits.MaxRunners)
	}

	if limits.MaxRepositories != -1 && repositories > limits.MaxRepositories {
		return fmt.Errorf("repository limit exceeded: %d/%d", repositories, limits.MaxRepositories)
	}

	if limits.MaxPodMinutes != -1 && podMinutes > limits.MaxPodMinutes {
		return fmt.Errorf("pod minutes exceeded: %d/%d", podMinutes, limits.MaxPodMinutes)
	}

	return nil
}

// HasFeature checks if a specific feature is enabled in the license
func (s *Service) HasFeature(feature string) bool {
	license := s.GetCurrentLicense()
	if license == nil {
		return false
	}

	for _, f := range license.Features {
		if f == feature {
			return true
		}
	}

	return false
}
