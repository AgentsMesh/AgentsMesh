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
	service := NewService(db)
	ctx := context.Background()

	// Create a runner
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

func TestUpdateLastSeen(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create a runner with offline status
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node",
		Status:         runner.RunnerStatusOffline,
	}
	require.NoError(t, db.Create(r).Error)

	// Update last seen
	err := service.UpdateLastSeen(ctx, r.ID)
	require.NoError(t, err)

	// Verify the runner was updated
	var updated runner.Runner
	require.NoError(t, db.First(&updated, r.ID).Error)

	assert.Equal(t, runner.RunnerStatusOnline, updated.Status)
	assert.NotNil(t, updated.LastHeartbeat)
}

func TestGetRunner(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create a runner
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
		// Add to active runners cache
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
		assert.Equal(t, "cached-node", result.NodeID) // Should return cached version
	})
}

// Note: TestListRunners, TestListAvailableRunners, TestSelectAvailableRunner are defined in registration_test.go
