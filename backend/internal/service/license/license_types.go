package license

import (
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// LicenseData represents the decoded license file structure
type LicenseData struct {
	LicenseKey       string        `json:"license_key"`
	OrganizationName string        `json:"organization_name"`
	ContactEmail     string        `json:"contact_email"`
	Plan             string        `json:"plan"`
	Limits           LicenseLimits `json:"limits"`
	Features         []string      `json:"features,omitempty"`
	IssuedAt         time.Time     `json:"issued_at"`
	ExpiresAt        time.Time     `json:"expires_at"`
	Signature        string        `json:"signature"`
}

// LicenseLimits defines the resource limits for the license
type LicenseLimits struct {
	MaxUsers        int `json:"max_users"`
	MaxRunners      int `json:"max_runners"`
	MaxRepositories int `json:"max_repositories"`
	MaxPodMinutes   int `json:"max_pod_minutes"` // -1 for unlimited
}

// Service handles license verification and management
type Service struct {
	repo      billing.LicenseRepository
	cfg       *config.LicenseConfig
	logger    *slog.Logger
	publicKey interface{} // *rsa.PublicKey, but stored as interface to avoid import in types file

	// Cache for current license
	mu             sync.RWMutex
	currentLicense *LicenseData
	lastCheck      time.Time
}
