package promocode_test

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	svc "github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromoCode_ValidateAndRedeem(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	past := time.Now().AddDate(0, -1, 0)
	future := time.Now().AddDate(0, 1, 0)

	db.Create(&promocode.PromoCode{
		Code:           "VALREDEEM01",
		Name:           "Validate Redeem Test",
		Type:           promocode.PromoTypePartner,
		PlanName:       "pro",
		DurationMonths: 2,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &future,
		MaxUsesPerOrg:  1,
	})

	// Validate
	vResp, err := service.Validate(ctx, &svc.ValidateRequest{
		Code:           "VALREDEEM01",
		OrganizationID: 1,
	})
	require.NoError(t, err)
	assert.True(t, vResp.Valid)
	assert.Equal(t, "pro", vResp.PlanName)
	assert.Equal(t, "Pro", vResp.PlanDisplayName)
	assert.Equal(t, 2, vResp.DurationMonths)

	// Redeem
	rResp, err := service.Redeem(ctx, &svc.RedeemRequest{
		Code:           "VALREDEEM01",
		OrganizationID: 1,
		UserID:         1,
		UserRole:       "owner",
		IPAddress:      "10.0.0.1",
		UserAgent:      "test",
	})
	require.NoError(t, err)
	assert.True(t, rResp.Success)
	assert.Equal(t, "pro", rResp.PlanName)
	assert.Equal(t, 2, rResp.DurationMonths)
	assert.False(t, rResp.NewPeriodEnd.IsZero())

	// Verify redemption history
	history, err := service.GetRedemptionHistory(ctx, 1)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, int64(1), history[0].OrganizationID)
	assert.Equal(t, "pro", history[0].PlanName)
}

func TestPromoCode_MaxUsesExceeded(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	past := time.Now().AddDate(0, -1, 0)
	future := time.Now().AddDate(0, 1, 0)

	db.Create(&promocode.PromoCode{
		Code:           "MAXUSE1",
		Name:           "Max Uses 1",
		Type:           promocode.PromoTypeCampaign,
		PlanName:       "pro",
		DurationMonths: 1,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &future,
		MaxUses:        intPtr(1),
		MaxUsesPerOrg:  1,
	})

	// First redeem should succeed
	resp, err := service.Redeem(ctx, &svc.RedeemRequest{
		Code:           "MAXUSE1",
		OrganizationID: 1,
		UserID:         1,
		UserRole:       "owner",
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Create a second org so max_uses_per_org doesn't block first
	db.Exec(`INSERT INTO organizations (id, name, slug) VALUES (3, 'Org3', 'org-3')`)

	// Second redeem with a different org should fail because max_uses=1
	resp2, err := service.Redeem(ctx, &svc.RedeemRequest{
		Code:           "MAXUSE1",
		OrganizationID: 3,
		UserID:         1,
		UserRole:       "owner",
	})
	require.NoError(t, err)
	assert.False(t, resp2.Success)
	assert.Equal(t, svc.ErrCodeMaxUsed, resp2.MessageCode)
}

func TestPromoCode_ExpiredCode(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	past := time.Now().AddDate(0, -2, 0)
	expired := time.Now().AddDate(0, -1, 0)

	db.Create(&promocode.PromoCode{
		Code:           "EXPIRED01",
		Name:           "Expired Code",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 3,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &expired,
		MaxUsesPerOrg:  1,
	})

	// Validate should report not valid with expired message code
	resp, err := service.Validate(ctx, &svc.ValidateRequest{
		Code:           "EXPIRED01",
		OrganizationID: 1,
	})
	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, svc.ErrCodeExpired, resp.MessageCode)
}
