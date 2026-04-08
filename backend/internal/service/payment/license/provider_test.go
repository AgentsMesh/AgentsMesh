package license

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

func setupTestProvider(t *testing.T) *Provider {
	db := setupTestDB(t)
	repo := infra.NewLicenseRepository(db)
	cfg := &config.LicenseConfig{}

	provider, err := NewProvider(cfg, repo)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	return provider
}

func TestNewProvider(t *testing.T) {
	provider := setupTestProvider(t)

	if provider.GetProviderName() != billing.PaymentProviderLicense {
		t.Errorf("expected provider name %s, got %s", billing.PaymentProviderLicense, provider.GetProviderName())
	}
}

func TestVerifyLicense(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create valid license data (without signature verification)
	licenseData := LicenseData{
		LicenseKey:        "TEST-LICENSE-001",
		OrganizationName:  "Test Org",
		ContactEmail:      "test@example.com",
		PlanName:          billing.PlanEnterprise,
		MaxUsers:          -1,
		MaxRunners:        -1,
		MaxRepositories:   -1,
		MaxConcurrentPods: -1,
		Features:          []string{"advanced_analytics", "priority_support"},
		IssuedAt:          time.Now(),
		Signature:         "mock_signature",
	}

	jsonData, _ := json.Marshal(licenseData)

	license, err := provider.VerifyLicense(ctx, jsonData)
	if err != nil {
		t.Fatalf("failed to verify license: %v", err)
	}

	if license.LicenseKey != "TEST-LICENSE-001" {
		t.Errorf("expected license key TEST-LICENSE-001, got %s", license.LicenseKey)
	}
	if license.OrganizationName != "Test Org" {
		t.Errorf("expected organization name Test Org, got %s", license.OrganizationName)
	}
	if license.PlanName != billing.PlanEnterprise {
		t.Errorf("expected plan enterprise, got %s", license.PlanName)
	}
}

func TestVerifyLicense_Expired(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	expiredTime := time.Now().Add(-24 * time.Hour)
	licenseData := LicenseData{
		LicenseKey:       "TEST-EXPIRED",
		OrganizationName: "Test Org",
		ContactEmail:     "test@example.com",
		PlanName:         billing.PlanEnterprise,
		IssuedAt:         time.Now().Add(-48 * time.Hour),
		ExpiresAt:        &expiredTime,
		Signature:        "mock_signature",
	}

	jsonData, _ := json.Marshal(licenseData)

	_, err := provider.VerifyLicense(ctx, jsonData)
	if err != ErrLicenseExpired {
		t.Errorf("expected ErrLicenseExpired, got %v", err)
	}
}

func TestVerifyLicense_InvalidJSON(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	_, err := provider.VerifyLicense(ctx, []byte("invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestActivateLicenseFromFile(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	licenseData := LicenseData{
		LicenseKey:        "TEST-ACTIVATE-001",
		OrganizationName:  "Acme Corp",
		ContactEmail:      "admin@acme.com",
		PlanName:          billing.PlanEnterprise,
		MaxUsers:          100,
		MaxRunners:        50,
		MaxRepositories:   -1,
		MaxConcurrentPods: 20,
		IssuedAt:          time.Now(),
		Signature:         "mock_signature",
	}

	jsonData, _ := json.Marshal(licenseData)

	license, err := provider.ActivateLicenseFromFile(ctx, jsonData, 1)
	if err != nil {
		t.Fatalf("failed to activate license: %v", err)
	}

	if license.ActivatedOrgID == nil || *license.ActivatedOrgID != 1 {
		t.Error("expected license to be activated for org 1")
	}
	if license.ActivatedAt == nil {
		t.Error("expected ActivatedAt to be set")
	}
}

func TestActivateLicenseFromFile_AlreadyActivated(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	licenseData := LicenseData{
		LicenseKey:       "TEST-ALREADY-001",
		OrganizationName: "Acme Corp",
		ContactEmail:     "admin@acme.com",
		PlanName:         billing.PlanEnterprise,
		IssuedAt:         time.Now(),
		Signature:        "mock_signature",
	}

	jsonData, _ := json.Marshal(licenseData)

	// Activate for org 1
	_, err := provider.ActivateLicenseFromFile(ctx, jsonData, 1)
	if err != nil {
		t.Fatalf("failed to activate license: %v", err)
	}

	// Try to activate for org 2
	_, err = provider.ActivateLicenseFromFile(ctx, jsonData, 2)
	if err != ErrAlreadyActivated {
		t.Errorf("expected ErrAlreadyActivated, got %v", err)
	}
}

func TestGetLicenseStatus_NoLicense(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	status, err := provider.GetLicenseStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.IsValid {
		t.Error("expected status.IsValid to be false")
	}
}

func TestGetLicenseStatus_WithActiveLicense(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create and activate a license
	licenseData := LicenseData{
		LicenseKey:       "TEST-STATUS-001",
		OrganizationName: "Test Org",
		ContactEmail:     "test@example.com",
		PlanName:         billing.PlanEnterprise,
		IssuedAt:         time.Now(),
		Signature:        "mock_signature",
	}

	jsonData, _ := json.Marshal(licenseData)
	_, _ = provider.ActivateLicenseFromFile(ctx, jsonData, 1)

	status, err := provider.GetLicenseStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.IsValid {
		t.Error("expected status.IsValid to be true")
	}
	if status.License == nil {
		t.Error("expected license to be present")
	}
}

func TestCancelSubscription(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create and activate a license
	licenseData := LicenseData{
		LicenseKey:       "TEST-CANCEL-001",
		OrganizationName: "Test Org",
		ContactEmail:     "test@example.com",
		PlanName:         billing.PlanEnterprise,
		IssuedAt:         time.Now(),
		Signature:        "mock_signature",
	}

	jsonData, _ := json.Marshal(licenseData)
	_, _ = provider.ActivateLicenseFromFile(ctx, jsonData, 1)

	// Cancel the license
	err := provider.CancelSubscription(ctx, "TEST-CANCEL-001", true)
	if err != nil {
		t.Fatalf("failed to cancel: %v", err)
	}

	// Verify status
	status, _ := provider.GetLicenseStatus(ctx)
	if status.IsValid {
		t.Error("expected license to be invalid after cancellation")
	}
}

func TestCancelSubscription_NotFound(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	err := provider.CancelSubscription(ctx, "NONEXISTENT", true)
	if err != ErrLicenseNotFound {
		t.Errorf("expected ErrLicenseNotFound, got %v", err)
	}
}

func TestUnsupportedOperations(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	// CreateCheckoutSession should return error
	_, err := provider.CreateCheckoutSession(ctx, nil)
	if err == nil {
		t.Error("expected error for CreateCheckoutSession")
	}

	// GetCheckoutStatus should return error
	_, err = provider.GetCheckoutStatus(ctx, "test")
	if err == nil {
		t.Error("expected error for GetCheckoutStatus")
	}

	// HandleWebhook should return error
	_, err = provider.HandleWebhook(ctx, nil, "")
	if err == nil {
		t.Error("expected error for HandleWebhook")
	}

	// RefundPayment should return error
	_, err = provider.RefundPayment(ctx, nil)
	if err == nil {
		t.Error("expected error for RefundPayment")
	}
}

func TestLicenseWithExpiry(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create license expiring in 15 days
	expiresAt := time.Now().Add(15 * 24 * time.Hour)
	licenseData := LicenseData{
		LicenseKey:       "TEST-EXPIRY-001",
		OrganizationName: "Test Org",
		ContactEmail:     "test@example.com",
		PlanName:         billing.PlanEnterprise,
		IssuedAt:         time.Now(),
		ExpiresAt:        &expiresAt,
		Signature:        "mock_signature",
	}

	jsonData, _ := json.Marshal(licenseData)
	_, _ = provider.ActivateLicenseFromFile(ctx, jsonData, 1)

	status, _ := provider.GetLicenseStatus(ctx)
	if !status.IsValid {
		t.Error("expected license to be valid")
	}
	if status.DaysUntilExpiry < 14 || status.DaysUntilExpiry > 16 {
		t.Errorf("expected days until expiry around 15, got %d", status.DaysUntilExpiry)
	}
	if status.Message == "" {
		t.Error("expected warning message for expiring license")
	}
}
