package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByNodeID(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node-123",
		Status:         runner.RunnerStatusOnline,
	}
	require.NoError(t, db.Create(r).Error)

	t.Run("returns runner by node ID", func(t *testing.T) {
		result, err := service.GetByNodeID(ctx, "test-node-123")
		require.NoError(t, err)
		assert.Equal(t, r.ID, result.ID)
		assert.Equal(t, "test-node-123", result.NodeID)
	})

	t.Run("returns error for non-existent node ID", func(t *testing.T) {
		_, err := service.GetByNodeID(ctx, "non-existent-node")
		assert.Equal(t, ErrRunnerNotFound, err)
	})
}

func TestGetByNodeIDAndOrgID(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r1 := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "shared-node",
		Status:         runner.RunnerStatusOnline,
	}
	r2 := &runner.Runner{
		OrganizationID: 2,
		NodeID:         "shared-node",
		Status:         runner.RunnerStatusOnline,
	}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)

	t.Run("returns correct runner for org 1", func(t *testing.T) {
		result, err := service.GetByNodeIDAndOrgID(ctx, "shared-node", 1)
		require.NoError(t, err)
		assert.Equal(t, r1.ID, result.ID)
		assert.Equal(t, int64(1), result.OrganizationID)
	})

	t.Run("returns correct runner for org 2", func(t *testing.T) {
		result, err := service.GetByNodeIDAndOrgID(ctx, "shared-node", 2)
		require.NoError(t, err)
		assert.Equal(t, r2.ID, result.ID)
		assert.Equal(t, int64(2), result.OrganizationID)
	})

	t.Run("returns error for non-existent org", func(t *testing.T) {
		_, err := service.GetByNodeIDAndOrgID(ctx, "shared-node", 999)
		assert.Equal(t, ErrRunnerNotFound, err)
	})

	t.Run("returns error for non-existent node ID", func(t *testing.T) {
		_, err := service.GetByNodeIDAndOrgID(ctx, "non-existent", 1)
		assert.Equal(t, ErrRunnerNotFound, err)
	})
}

func TestUpdateLastSeen(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node",
		Status:         runner.RunnerStatusOffline,
	}
	require.NoError(t, db.Create(r).Error)

	err := service.UpdateLastSeen(ctx, r.ID)
	require.NoError(t, err)

	var updated runner.Runner
	require.NoError(t, db.First(&updated, r.ID).Error)

	assert.Equal(t, runner.RunnerStatusOnline, updated.Status)
	assert.NotNil(t, updated.LastHeartbeat)
}

func TestGetRunner(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node",
		Status:         runner.RunnerStatusOnline,
	}
	require.NoError(t, db.Create(r).Error)

	t.Run("returns runner from database", func(t *testing.T) {
		result, err := service.GetRunner(ctx, r.ID)
		require.NoError(t, err)
		assert.Equal(t, r.ID, result.ID)
		assert.Equal(t, "test-node", result.NodeID)
	})

	t.Run("returns error for non-existent runner", func(t *testing.T) {
		_, err := service.GetRunner(ctx, 99999)
		assert.Equal(t, ErrRunnerNotFound, err)
	})

	t.Run("returns runner from cache when available", func(t *testing.T) {
		cachedRunner := &runner.Runner{
			ID:             r.ID,
			OrganizationID: 1,
			NodeID:         "cached-node",
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{
			Runner: cachedRunner,
		})

		result, err := service.GetRunner(ctx, r.ID)
		require.NoError(t, err)
		assert.Equal(t, "cached-node", result.NodeID)
	})
}
