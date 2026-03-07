package license

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// NewService creates a new license service
func NewService(repo billing.LicenseRepository, cfg *config.LicenseConfig, logger *slog.Logger) (*Service, error) {
	svc := &Service{
		repo:   repo,
		cfg:    cfg,
		logger: logger,
	}

	// Load public key for signature verification
	if cfg.PublicKeyPath != "" {
		publicKey, err := loadPublicKey(cfg.PublicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load license public key: %w", err)
		}
		svc.publicKey = publicKey
	}

	// Load license file if specified
	if cfg.LicenseFilePath != "" {
		if err := svc.loadLicenseFile(); err != nil {
			logger.Warn("failed to load license file", "error", err)
			// Don't fail startup, allow activation later
		}
	}

	return svc, nil
}

// loadPublicKey loads an RSA public key from a PEM file
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// loadLicenseFile loads and verifies the license file
func (s *Service) loadLicenseFile() error {
	data, err := os.ReadFile(s.cfg.LicenseFilePath)
	if err != nil {
		return fmt.Errorf("failed to read license file: %w", err)
	}

	license, err := s.ParseAndVerify(data)
	if err != nil {
		return fmt.Errorf("failed to verify license: %w", err)
	}

	s.mu.Lock()
	s.currentLicense = license
	s.lastCheck = time.Now()
	s.mu.Unlock()

	s.logger.Info("license loaded successfully",
		"license_key", license.LicenseKey,
		"organization", license.OrganizationName,
		"plan", license.Plan,
		"expires_at", license.ExpiresAt,
	)

	return nil
}

// ParseAndVerify parses license data and verifies the signature
func (s *Service) ParseAndVerify(data []byte) (*LicenseData, error) {
	var license LicenseData
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, fmt.Errorf("failed to parse license JSON: %w", err)
	}

	// Verify signature
	if s.publicKey != nil {
		if err := s.verifySignature(&license); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Check expiration
	if time.Now().After(license.ExpiresAt) {
		return nil, fmt.Errorf("license has expired")
	}

	return &license, nil
}

// verifySignature verifies the license signature using RSA-SHA256
func (s *Service) verifySignature(license *LicenseData) error {
	rsaPub, ok := s.publicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key not available")
	}

	// Build the data that was signed (everything except the signature)
	dataToSign := struct {
		LicenseKey       string        `json:"license_key"`
		OrganizationName string        `json:"organization_name"`
		ContactEmail     string        `json:"contact_email"`
		Plan             string        `json:"plan"`
		Limits           LicenseLimits `json:"limits"`
		Features         []string      `json:"features,omitempty"`
		IssuedAt         time.Time     `json:"issued_at"`
		ExpiresAt        time.Time     `json:"expires_at"`
	}{
		LicenseKey:       license.LicenseKey,
		OrganizationName: license.OrganizationName,
		ContactEmail:     license.ContactEmail,
		Plan:             license.Plan,
		Limits:           license.Limits,
		Features:         license.Features,
		IssuedAt:         license.IssuedAt,
		ExpiresAt:        license.ExpiresAt,
	}

	jsonData, err := json.Marshal(dataToSign)
	if err != nil {
		return fmt.Errorf("failed to marshal license data: %w", err)
	}

	// Decode signature from base64
	signature, err := base64.StdEncoding.DecodeString(license.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Hash the data
	hash := sha256.Sum256(jsonData)

	// Verify signature
	if err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hash[:], signature); err != nil {
		return fmt.Errorf("signature mismatch: %w", err)
	}

	return nil
}

// RefreshLicense reloads the license from file
func (s *Service) RefreshLicense() error {
	if s.cfg.LicenseFilePath == "" {
		return fmt.Errorf("no license file path configured")
	}
	return s.loadLicenseFile()
}
