package license

import (
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

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

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
