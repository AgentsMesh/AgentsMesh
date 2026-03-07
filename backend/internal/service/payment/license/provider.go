package license

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

var (
	ErrInvalidLicense      = errors.New("invalid license")
	ErrLicenseExpired      = errors.New("license expired")
	ErrLicenseRevoked      = errors.New("license revoked")
	ErrLicenseNotFound     = errors.New("license not found")
	ErrInvalidSignature    = errors.New("invalid license signature")
	ErrNoPublicKey         = errors.New("no public key configured")
	ErrAlreadyActivated    = errors.New("license already activated for another organization")
	ErrLicenseFileNotFound = errors.New("license file not found")
)

// LicenseData represents the JSON structure of a license file
type LicenseData struct {
	LicenseKey        string     `json:"license_key"`
	OrganizationName  string     `json:"organization_name"`
	ContactEmail      string     `json:"contact_email"`
	PlanName          string     `json:"plan_name"`
	MaxUsers          int        `json:"max_users"`
	MaxRunners        int        `json:"max_runners"`
	MaxRepositories   int        `json:"max_repositories"`
	MaxConcurrentPods int        `json:"max_concurrent_pods"`
	Features          []string   `json:"features,omitempty"`
	IssuedAt          time.Time  `json:"issued_at"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	Signature         string     `json:"signature"`
}

// Provider implements the LicenseProvider interface
type Provider struct {
	config    *config.LicenseConfig
	repo      billing.LicenseRepository
	publicKey *rsa.PublicKey
}

// NewProvider creates a new license provider
func NewProvider(cfg *config.LicenseConfig, repo billing.LicenseRepository) (*Provider, error) {
	p := &Provider{
		config: cfg,
		repo:   repo,
	}

	// Load public key if configured
	if cfg.PublicKeyPath != "" {
		key, err := loadPublicKey(cfg.PublicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load public key: %w", err)
		}
		p.publicKey = key
	}

	return p, nil
}

// GetProviderName returns the provider name
func (p *Provider) GetProviderName() string {
	return billing.PaymentProviderLicense
}

// CreateCheckoutSession is not applicable for license provider
// For OnPremise, organizations are activated via license file, not checkout
func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutResponse, error) {
	return nil, errors.New("checkout not supported for license provider - use license activation instead")
}

// GetCheckoutStatus is not applicable for license provider
func (p *Provider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	return "", errors.New("checkout not supported for license provider")
}

// HandleWebhook is not applicable for license provider
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	return nil, errors.New("webhooks not supported for license provider")
}

// RefundPayment is not applicable for license provider
func (p *Provider) RefundPayment(ctx context.Context, req *types.RefundRequest) (*types.RefundResponse, error) {
	return nil, errors.New("refunds not supported for license provider")
}

// CancelSubscription deactivates a license
func (p *Provider) CancelSubscription(ctx context.Context, licenseKey string, immediate bool) error {
	license, err := p.repo.GetByKey(ctx, licenseKey)
	if err != nil {
		return err
	}
	if license == nil {
		return ErrLicenseNotFound
	}

	now := time.Now()
	reason := "User requested cancellation"
	license.IsActive = false
	license.RevokedAt = &now
	license.RevocationReason = &reason

	return p.repo.Save(ctx, license)
}

// VerifyLicense verifies a license file/key and returns the license if valid
func (p *Provider) VerifyLicense(ctx context.Context, licenseData []byte) (*billing.License, error) {
	// Parse license data
	var data LicenseData
	if err := json.Unmarshal(licenseData, &data); err != nil {
		return nil, fmt.Errorf("%w: failed to parse license data", ErrInvalidLicense)
	}

	// Verify signature if public key is available
	if p.publicKey != nil {
		if err := p.verifySignature(&data); err != nil {
			return nil, err
		}
	}

	// Check expiration
	if data.ExpiresAt != nil && time.Now().After(*data.ExpiresAt) {
		return nil, ErrLicenseExpired
	}

	// Convert features to billing.Features
	features := billing.Features{}
	for _, f := range data.Features {
		features[f] = true
	}

	// Create license object
	license := &billing.License{
		LicenseKey:        data.LicenseKey,
		OrganizationName:  data.OrganizationName,
		ContactEmail:      data.ContactEmail,
		PlanName:          data.PlanName,
		MaxUsers:          data.MaxUsers,
		MaxRunners:        data.MaxRunners,
		MaxRepositories:   data.MaxRepositories,
		MaxConcurrentPods: data.MaxConcurrentPods,
		Features:          features,
		IssuedAt:          data.IssuedAt,
		ExpiresAt:         data.ExpiresAt,
		Signature:         data.Signature,
		IsActive:          true,
	}

	return license, nil
}

// GetLicenseStatus returns the current license status
func (p *Provider) GetLicenseStatus(ctx context.Context) (*types.LicenseStatus, error) {
	// Try to load from database first (if license was activated)
	license, err := p.repo.GetActiveLicense(ctx)
	if err != nil {
		return nil, err
	}
	if license != nil {
		return p.licenseToStatus(license), nil
	}

	// Try to load from file if configured
	if p.config.LicenseFilePath != "" {
		licenseData, err := os.ReadFile(p.config.LicenseFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				return &types.LicenseStatus{
					IsValid: false,
					Message: "No license found",
				}, nil
			}
			return nil, fmt.Errorf("failed to read license file: %w", err)
		}

		verifiedLicense, err := p.VerifyLicense(ctx, licenseData)
		if err != nil {
			return &types.LicenseStatus{
				IsValid: false,
				Message: err.Error(),
			}, nil
		}

		return p.licenseToStatus(verifiedLicense), nil
	}

	return &types.LicenseStatus{
		IsValid: false,
		Message: "No license configured",
	}, nil
}

// ActivateLicense activates a license for an organization
func (p *Provider) ActivateLicense(ctx context.Context, licenseKey string, orgID int64) error {
	// Find the license by key
	license, err := p.repo.GetByKey(ctx, licenseKey)
	if err != nil {
		return err
	}
	if license == nil {
		return ErrLicenseNotFound
	}

	// Check if license is valid
	if !license.IsValid() {
		if license.RevokedAt != nil {
			return ErrLicenseRevoked
		}
		if license.ExpiresAt != nil && time.Now().After(*license.ExpiresAt) {
			return ErrLicenseExpired
		}
		return ErrInvalidLicense
	}

	// Check if already activated for another org
	if license.IsActivated() && *license.ActivatedOrgID != orgID {
		return ErrAlreadyActivated
	}

	// Activate the license
	now := time.Now()
	license.ActivatedAt = &now
	license.ActivatedOrgID = &orgID
	license.LastVerifiedAt = &now

	return p.repo.Save(ctx, license)
}

// ActivateLicenseFromFile activates a license from file data
func (p *Provider) ActivateLicenseFromFile(ctx context.Context, licenseData []byte, orgID int64) (*billing.License, error) {
	// Verify the license first
	license, err := p.VerifyLicense(ctx, licenseData)
	if err != nil {
		return nil, err
	}

	// Check if this license key already exists
	existing, err := p.repo.GetByKey(ctx, license.LicenseKey)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// License exists - check if it can be activated
		if existing.IsActivated() && *existing.ActivatedOrgID != orgID {
			return nil, ErrAlreadyActivated
		}
		// Update existing license
		now := time.Now()
		existing.ActivatedAt = &now
		existing.ActivatedOrgID = &orgID
		existing.LastVerifiedAt = &now
		if err := p.repo.Save(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// Create new license record
	now := time.Now()
	license.ActivatedAt = &now
	license.ActivatedOrgID = &orgID
	license.LastVerifiedAt = &now

	if err := p.repo.Create(ctx, license); err != nil {
		return nil, err
	}

	return license, nil
}

// verifySignature verifies the license signature
func (p *Provider) verifySignature(data *LicenseData) error {
	if p.publicKey == nil {
		return ErrNoPublicKey
	}

	// Create the data to verify (all fields except signature)
	dataToSign := LicenseData{
		LicenseKey:        data.LicenseKey,
		OrganizationName:  data.OrganizationName,
		ContactEmail:      data.ContactEmail,
		PlanName:          data.PlanName,
		MaxUsers:          data.MaxUsers,
		MaxRunners:        data.MaxRunners,
		MaxRepositories:   data.MaxRepositories,
		MaxConcurrentPods: data.MaxConcurrentPods,
		Features:          data.Features,
		IssuedAt:          data.IssuedAt,
		ExpiresAt:         data.ExpiresAt,
	}

	jsonData, err := json.Marshal(dataToSign)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal data for verification", ErrInvalidSignature)
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(data.Signature)
	if err != nil {
		return fmt.Errorf("%w: failed to decode signature", ErrInvalidSignature)
	}

	// Hash the data
	hash := sha256.Sum256(jsonData)

	// Verify signature
	if err := rsa.VerifyPKCS1v15(p.publicKey, crypto.SHA256, hash[:], sigBytes); err != nil {
		return ErrInvalidSignature
	}

	return nil
}

// licenseToStatus converts a License to LicenseStatus
func (p *Provider) licenseToStatus(license *billing.License) *types.LicenseStatus {
	status := &types.LicenseStatus{
		IsValid:         license.IsValid(),
		DaysUntilExpiry: license.DaysUntilExpiry(),
		License:         license,
	}

	if !status.IsValid {
		if license.RevokedAt != nil {
			status.Message = "License revoked"
		} else if license.ExpiresAt != nil && time.Now().After(*license.ExpiresAt) {
			status.Message = "License expired"
		} else {
			status.Message = "License inactive"
		}
	} else if status.DaysUntilExpiry >= 0 && status.DaysUntilExpiry <= 30 {
		status.Message = fmt.Sprintf("License expires in %d days", status.DaysUntilExpiry)
	} else {
		status.Message = "License active"
	}

	return status
}

// loadPublicKey loads an RSA public key from a PEM file
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPub, nil
}
