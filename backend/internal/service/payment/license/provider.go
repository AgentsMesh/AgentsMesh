package license

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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

type Provider struct {
	config    *config.LicenseConfig
	repo      billing.LicenseRepository
	publicKey *rsa.PublicKey
}

func NewProvider(cfg *config.LicenseConfig, repo billing.LicenseRepository) (*Provider, error) {
	p := &Provider{
		config: cfg,
		repo:   repo,
	}

	if cfg.PublicKeyPath != "" {
		key, err := loadPublicKey(cfg.PublicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load public key: %w", err)
		}
		p.publicKey = key
	}

	return p, nil
}

func (p *Provider) GetProviderName() string {
	return billing.PaymentProviderLicense
}

func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutResponse, error) {
	return nil, errors.New("checkout not supported for license provider - use license activation instead")
}

func (p *Provider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	return "", errors.New("checkout not supported for license provider")
}

func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	return nil, errors.New("webhooks not supported for license provider")
}

func (p *Provider) RefundPayment(ctx context.Context, req *types.RefundRequest) (*types.RefundResponse, error) {
	return nil, errors.New("refunds not supported for license provider")
}

func (p *Provider) CancelSubscription(ctx context.Context, licenseKey string, immediate bool) error {
	license, err := p.repo.GetByKey(ctx, licenseKey)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get license for cancellation", "license_key", licenseKey, "error", err)
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

	if err := p.repo.Save(ctx, license); err != nil {
		slog.ErrorContext(ctx, "failed to save revoked license", "license_key", licenseKey, "error", err)
		return err
	}
	slog.InfoContext(ctx, "license canceled", "license_key", licenseKey)
	return nil
}

func (p *Provider) VerifyLicense(ctx context.Context, licenseData []byte) (*billing.License, error) {
	var data LicenseData
	if err := json.Unmarshal(licenseData, &data); err != nil {
		slog.ErrorContext(ctx, "failed to parse license data", "error", err)
		return nil, fmt.Errorf("%w: failed to parse license data", ErrInvalidLicense)
	}

	if p.publicKey != nil {
		if err := p.verifySignature(&data); err != nil {
			slog.WarnContext(ctx, "license signature verification failed", "license_key", data.LicenseKey, "error", err)
			return nil, err
		}
	}

	if data.ExpiresAt != nil && time.Now().After(*data.ExpiresAt) {
		slog.WarnContext(ctx, "license expired", "license_key", data.LicenseKey, "expires_at", data.ExpiresAt)
		return nil, ErrLicenseExpired
	}

	features := billing.Features{}
	for _, f := range data.Features {
		features[f] = true
	}

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
