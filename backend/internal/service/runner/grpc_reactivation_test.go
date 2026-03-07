package runner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateReactivationToken(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("generates reactivation token", func(t *testing.T) {
		// Create runner
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-react-1",
			Status:         runner.RunnerStatusOffline,
		}
		require.NoError(t, db.Create(r).Error)

		resp, err := service.GenerateReactivationToken(ctx, r.ID, 1) // userID=1
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.Equal(t, 600, resp.ExpiresIn)
		assert.Contains(t, resp.Command, "runner reactivate")

		// Verify token was saved
		var reactivation runner.ReactivationToken
		err = db.Where("runner_id = ?", r.ID).First(&reactivation).Error
		require.NoError(t, err)
		assert.NotEmpty(t, reactivation.TokenHash)
	})

	t.Run("returns error for non-existent runner", func(t *testing.T) {
		_, err := service.GenerateReactivationToken(ctx, 99999, 1)
		assert.Error(t, err)
	})
}

func TestReactivate(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("returns error for invalid token", func(t *testing.T) {
		req := &ReactivateRequest{Token: "invalid-token"}
		_, err := service.Reactivate(ctx, req, nil)
		assert.Error(t, err)
	})

	t.Run("returns error for expired token", func(t *testing.T) {
		// Create runner
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-react-2",
			Status:         runner.RunnerStatusOffline,
		}
		require.NoError(t, db.Create(r).Error)

		// Create expired reactivation token
		token := generateTestAuthKey()
		tokenHash := hashToken(token)
		reactivation := &runner.ReactivationToken{
			RunnerID:  r.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		require.NoError(t, db.Create(reactivation).Error)

		req := &ReactivateRequest{Token: token}
		_, err := service.Reactivate(ctx, req, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("successfully reactivates runner", func(t *testing.T) {
		// Setup PKI
		pkiService, tmpDir := setupTestPKI(t)
		defer os.RemoveAll(tmpDir)

		org := createTestOrg(t, db, "test-org-reactivate-success")

		// Create runner
		r := &runner.Runner{
			OrganizationID: org.ID,
			NodeID:         "test-node-reactivate",
			Status:         runner.RunnerStatusOffline,
		}
		require.NoError(t, db.Create(r).Error)

		// Create valid reactivation token
		token := generateTestAuthKey()
		tokenHash := hashToken(token)
		reactivation := &runner.ReactivationToken{
			RunnerID:  r.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		require.NoError(t, db.Create(reactivation).Error)

		req := &ReactivateRequest{Token: token}
		resp, err := service.Reactivate(ctx, req, pkiService)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Certificate)
		assert.NotEmpty(t, resp.PrivateKey)
		assert.NotEmpty(t, resp.CACertificate)

		// Verify token was marked as used
		var updatedToken runner.ReactivationToken
		require.NoError(t, db.First(&updatedToken, reactivation.ID).Error)
		assert.NotNil(t, updatedToken.UsedAt)
	})
}
