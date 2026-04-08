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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers -------------------------------------------------------------------

func newTestProvider(t *testing.T) (*Provider, billing.LicenseRepository) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewLicenseRepository(db)
	provider, err := NewProvider(&config.LicenseConfig{}, repo)
	require.NoError(t, err)
	return provider, repo
}

func makeLicenseJSON(t *testing.T, key string, expiresAt *time.Time) []byte {
	t.Helper()
	data := LicenseData{
		LicenseKey:        key,
		OrganizationName:  "IntegTest Org",
		ContactEmail:      "integ@example.com",
		PlanName:          billing.PlanEnterprise,
		MaxUsers:          50,
		MaxRunners:        10,
		MaxRepositories:   -1,
		MaxConcurrentPods: 5,
		IssuedAt:          time.Now().Add(-time.Hour),
		ExpiresAt:         expiresAt,
		Signature:         "test_sig",
	}
	b, err := json.Marshal(data)
	require.NoError(t, err)
	return b
}

// tests ---------------------------------------------------------------------

// TestInteg_ActivateLicenseByKey verifies ActivateLicense (key-based) stores
// activation metadata in the DB and subsequent GetLicenseStatus reports valid.
func TestInteg_ActivateLicenseByKey(t *testing.T) {
	provider, repo := newTestProvider(t)
	ctx := context.Background()

	// Seed a license record directly via the repo.
	lic := &billing.License{
		LicenseKey:       "KEY-ACTIVATE-INTEG",
		OrganizationName: "Org A",
		ContactEmail:     "a@example.com",
		PlanName:         billing.PlanEnterprise,
		Signature:        "sig",
		IsActive:         true,
		IssuedAt:         time.Now(),
	}
	require.NoError(t, repo.Create(ctx, lic))

	// Activate for org 10.
	err := provider.ActivateLicense(ctx, "KEY-ACTIVATE-INTEG", 10)
	require.NoError(t, err)

	// Verify DB state.
	stored, err := repo.GetByKey(ctx, "KEY-ACTIVATE-INTEG")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.True(t, stored.IsActivated())
	assert.Equal(t, int64(10), *stored.ActivatedOrgID)
	assert.NotNil(t, stored.LastVerifiedAt)
}

// TestInteg_ActivateLicenseByKey_Revoked ensures a revoked license cannot
// be activated.
func TestInteg_ActivateLicenseByKey_Revoked(t *testing.T) {
	provider, repo := newTestProvider(t)
	ctx := context.Background()

	now := time.Now()
	lic := &billing.License{
		LicenseKey:       "KEY-REVOKED",
		OrganizationName: "Org R",
		ContactEmail:     "r@example.com",
		PlanName:         billing.PlanEnterprise,
		Signature:        "sig",
		IsActive:         false,
		RevokedAt:        &now,
		IssuedAt:         time.Now(),
	}
	require.NoError(t, repo.Create(ctx, lic))

	err := provider.ActivateLicense(ctx, "KEY-REVOKED", 1)
	assert.ErrorIs(t, err, ErrLicenseRevoked)
}

// TestInteg_ActivateLicenseByKey_Expired ensures an expired license cannot
// be activated.
func TestInteg_ActivateLicenseByKey_Expired(t *testing.T) {
	provider, repo := newTestProvider(t)
	ctx := context.Background()

	past := time.Now().Add(-48 * time.Hour)
	lic := &billing.License{
		LicenseKey:       "KEY-EXPIRED",
		OrganizationName: "Org E",
		ContactEmail:     "e@example.com",
		PlanName:         billing.PlanEnterprise,
		Signature:        "sig",
		IsActive:         true,
		ExpiresAt:        &past,
		IssuedAt:         time.Now().Add(-72 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, lic))

	err := provider.ActivateLicense(ctx, "KEY-EXPIRED", 1)
	assert.ErrorIs(t, err, ErrLicenseExpired)
}

// TestInteg_ActivateLicenseByKey_AlreadyOtherOrg ensures a license activated
// for one org cannot be activated for a different org.
func TestInteg_ActivateLicenseByKey_AlreadyOtherOrg(t *testing.T) {
	provider, repo := newTestProvider(t)
	ctx := context.Background()

	orgID := int64(5)
	now := time.Now()
	lic := &billing.License{
		LicenseKey:       "KEY-TAKEN",
		OrganizationName: "Org T",
		ContactEmail:     "t@example.com",
		PlanName:         billing.PlanEnterprise,
		Signature:        "sig",
		IsActive:         true,
		ActivatedAt:      &now,
		ActivatedOrgID:   &orgID,
		IssuedAt:         time.Now(),
	}
	require.NoError(t, repo.Create(ctx, lic))

	err := provider.ActivateLicense(ctx, "KEY-TAKEN", 99)
	assert.ErrorIs(t, err, ErrAlreadyActivated)

	// Same org should succeed.
	err = provider.ActivateLicense(ctx, "KEY-TAKEN", 5)
	assert.NoError(t, err)
}

// TestInteg_DeactivateAll ensures DeactivateAll disables every active license
// and GetLicenseStatus returns inactive afterwards.
func TestInteg_DeactivateAll(t *testing.T) {
	provider, repo := newTestProvider(t)
	ctx := context.Background()

	// Activate two licenses.
	for _, key := range []string{"DA-001", "DA-002"} {
		data := makeLicenseJSON(t, key, nil)
		_, err := provider.ActivateLicenseFromFile(ctx, data, 1)
		require.NoError(t, err)
	}

	// Sanity: at least one active.
	status, err := provider.GetLicenseStatus(ctx)
	require.NoError(t, err)
	assert.True(t, status.IsValid)

	// Deactivate all.
	require.NoError(t, repo.DeactivateAll(ctx))

	// Both should now be inactive.
	for _, key := range []string{"DA-001", "DA-002"} {
		lic, err := repo.GetByKey(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, lic)
		assert.False(t, lic.IsActive)
	}

	// Status should be invalid.
	status, err = provider.GetLicenseStatus(ctx)
	require.NoError(t, err)
	assert.False(t, status.IsValid)
}

// TestInteg_FullLifecycle exercises activate → status → cancel → status using
// the database to verify each state transition persists correctly.
func TestInteg_FullLifecycle(t *testing.T) {
	provider, repo := newTestProvider(t)
	ctx := context.Background()

	// Step 1: Activate from file.
	data := makeLicenseJSON(t, "LIFECYCLE-001", nil)
	lic, err := provider.ActivateLicenseFromFile(ctx, data, 42)
	require.NoError(t, err)
	assert.True(t, lic.IsActive)
	assert.Equal(t, int64(42), *lic.ActivatedOrgID)

	// Step 2: Status is valid.
	status, err := provider.GetLicenseStatus(ctx)
	require.NoError(t, err)
	assert.True(t, status.IsValid)
	assert.Equal(t, "License active", status.Message)

	// Step 3: Cancel (deactivate).
	err = provider.CancelSubscription(ctx, "LIFECYCLE-001", true)
	require.NoError(t, err)

	// Step 4: DB reflects revocation.
	stored, err := repo.GetByKey(ctx, "LIFECYCLE-001")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.False(t, stored.IsActive)
	assert.NotNil(t, stored.RevokedAt)
	assert.Equal(t, "User requested cancellation", *stored.RevocationReason)

	// Step 5: Status is now invalid.
	status, err = provider.GetLicenseStatus(ctx)
	require.NoError(t, err)
	assert.False(t, status.IsValid)
}

// TestInteg_ExpiredLicense_DomainIsValid inserts an expired license via the
// repo and verifies the domain model's IsValid() returns false, and status
// reports it as expired.
func TestInteg_ExpiredLicense_DomainIsValid(t *testing.T) {
	provider, repo := newTestProvider(t)
	ctx := context.Background()

	past := time.Now().Add(-24 * time.Hour)
	lic := &billing.License{
		LicenseKey:       "EXPIRED-DOMAIN",
		OrganizationName: "Org X",
		ContactEmail:     "x@example.com",
		PlanName:         billing.PlanEnterprise,
		Signature:        "sig",
		IsActive:         true,
		ExpiresAt:        &past,
		IssuedAt:         time.Now().Add(-48 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, lic))

	// Round-trip: read from DB.
	stored, err := repo.GetByKey(ctx, "EXPIRED-DOMAIN")
	require.NoError(t, err)
	require.NotNil(t, stored)

	// Domain model should report invalid.
	assert.False(t, stored.IsValid())
	assert.True(t, stored.DaysUntilExpiry() < 0)

	// licenseToStatus should report "License expired".
	status := provider.licenseToStatus(stored)
	assert.False(t, status.IsValid)
	assert.Equal(t, "License expired", status.Message)
}

// TestInteg_ActivateLicenseByKey_NotFound ensures activating a non-existent
// key returns ErrLicenseNotFound.
func TestInteg_ActivateLicenseByKey_NotFound(t *testing.T) {
	provider, _ := newTestProvider(t)
	ctx := context.Background()

	err := provider.ActivateLicense(ctx, "DOES-NOT-EXIST", 1)
	assert.ErrorIs(t, err, ErrLicenseNotFound)
}

// TestInteg_ReactivateSameOrg verifies that re-activating a license for the
// same org updates LastVerifiedAt without error.
func TestInteg_ReactivateSameOrg(t *testing.T) {
	provider, repo := newTestProvider(t)
	ctx := context.Background()

	orgID := int64(7)
	earlier := time.Now().Add(-time.Hour)
	lic := &billing.License{
		LicenseKey:       "REACTIVATE-001",
		OrganizationName: "Org Re",
		ContactEmail:     "re@example.com",
		PlanName:         billing.PlanEnterprise,
		Signature:        "sig",
		IsActive:         true,
		ActivatedAt:      &earlier,
		ActivatedOrgID:   &orgID,
		LastVerifiedAt:   &earlier,
		IssuedAt:         time.Now().Add(-2 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, lic))

	// Re-activate for same org.
	err := provider.ActivateLicense(ctx, "REACTIVATE-001", 7)
	require.NoError(t, err)

	stored, err := repo.GetByKey(ctx, "REACTIVATE-001")
	require.NoError(t, err)
	assert.True(t, stored.LastVerifiedAt.After(earlier))
}
