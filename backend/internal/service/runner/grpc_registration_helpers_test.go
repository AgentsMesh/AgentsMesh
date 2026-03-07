package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRunnerByNodeID(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("returns runner by node ID", func(t *testing.T) {
		// Create runner
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "unique-node-id",
		}
		require.NoError(t, db.Create(r).Error)

		result, err := service.GetRunnerByNodeID(ctx, "unique-node-id")
		require.NoError(t, err)
		assert.Equal(t, r.ID, result.ID)
		assert.Equal(t, "unique-node-id", result.NodeID)
	})

	t.Run("returns error for non-existent node ID", func(t *testing.T) {
		_, err := service.GetRunnerByNodeID(ctx, "non-existent-node")
		assert.Error(t, err)
	})
}

func TestCleanupExpiredPendingAuths(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("cleans up expired pending auths", func(t *testing.T) {
		// Create expired pending auth
		expiredAuth := &runner.PendingAuth{
			AuthKey:    generateTestAuthKey(),
			MachineKey: "expired-machine",
			ExpiresAt:  time.Now().Add(-1 * time.Hour),
		}
		require.NoError(t, db.Create(expiredAuth).Error)

		// Create valid pending auth
		validAuth := &runner.PendingAuth{
			AuthKey:    generateTestAuthKey(),
			MachineKey: "valid-machine",
			ExpiresAt:  time.Now().Add(1 * time.Hour),
		}
		require.NoError(t, db.Create(validAuth).Error)

		// Cleanup
		err := service.CleanupExpiredPendingAuths(ctx)
		require.NoError(t, err)

		// Verify expired was deleted
		var count int64
		db.Model(&runner.PendingAuth{}).Where("id = ?", expiredAuth.ID).Count(&count)
		assert.Zero(t, count)

		// Verify valid was kept
		db.Model(&runner.PendingAuth{}).Where("id = ?", validAuth.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

func TestCleanupExpiredReactivationTokens(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("cleans up expired reactivation tokens", func(t *testing.T) {
		// Create runner
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-cleanup",
		}
		require.NoError(t, db.Create(r).Error)

		// Create expired reactivation token
		expiredToken := &runner.ReactivationToken{
			RunnerID:  r.ID,
			TokenHash: "expired-hash",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		require.NoError(t, db.Create(expiredToken).Error)

		// Create valid reactivation token
		validToken := &runner.ReactivationToken{
			RunnerID:  r.ID,
			TokenHash: "valid-hash",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		require.NoError(t, db.Create(validToken).Error)

		// Cleanup
		err := service.CleanupExpiredReactivationTokens(ctx)
		require.NoError(t, err)

		// Verify expired was deleted
		var count int64
		db.Model(&runner.ReactivationToken{}).Where("id = ?", expiredToken.ID).Count(&count)
		assert.Zero(t, count)

		// Verify valid was kept
		db.Model(&runner.ReactivationToken{}).Where("id = ?", validToken.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

// Helper functions

func generateTestAuthKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
