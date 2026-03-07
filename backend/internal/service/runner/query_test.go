package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByNodeID(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
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

func TestGetByNodeIDAndOrgID(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create runners with same node_id in different orgs
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
	service := newTestService(db)
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

func TestSelectAvailableRunnerForAgent(t *testing.T) {
	ctx := context.Background()

	t.Run("returns runner from cache when it supports the agent", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		r := &runner.Runner{
			ID:              1,
			OrganizationID:  1,
			NodeID:          "runner-1",
			Status:          runner.RunnerStatusOnline,
			IsEnabled:       true,
			MaxConcurrentPods: 5,
			AvailableAgents: runner.StringSlice{"claude-code", "aider"},
			Visibility:      runner.VisibilityOrganization,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{
			Runner:   r,
			LastPing: time.Now(),
			PodCount: 1,
		})

		result, err := service.SelectAvailableRunnerForAgent(ctx, 1, 1, "claude-code")
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.ID)
	})

	t.Run("skips cached runner that does not support the agent", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		// This runner only supports "aider", not "claude-code"
		r := &runner.Runner{
			ID:              1,
			OrganizationID:  1,
			NodeID:          "runner-1",
			Status:          runner.RunnerStatusOnline,
			IsEnabled:       true,
			MaxConcurrentPods: 5,
			AvailableAgents: runner.StringSlice{"aider"},
			Visibility:      runner.VisibilityOrganization,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{
			Runner:   r,
			LastPing: time.Now(),
			PodCount: 0,
		})

		// No DB runners either → expect ErrNoRunnerForAgent
		// Note: SQLite doesn't support @> operator, so DB fallback will fail.
		// We verify that cache filtering works by ensuring the aider-only runner is skipped.
		_, err := service.SelectAvailableRunnerForAgent(ctx, 1, 1, "claude-code")
		assert.Error(t, err)
	})

	t.Run("selects least-loaded runner from cache", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		r1 := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-1",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			AvailableAgents: runner.StringSlice{"claude-code"},
			Visibility: runner.VisibilityOrganization,
		}
		r2 := &runner.Runner{
			ID: 2, OrganizationID: 1, NodeID: "runner-2",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			AvailableAgents: runner.StringSlice{"claude-code"},
			Visibility: runner.VisibilityOrganization,
		}

		service.activeRunners.Store(r1.ID, &ActiveRunner{Runner: r1, LastPing: time.Now(), PodCount: 3})
		service.activeRunners.Store(r2.ID, &ActiveRunner{Runner: r2, LastPing: time.Now(), PodCount: 1})

		result, err := service.SelectAvailableRunnerForAgent(ctx, 1, 1, "claude-code")
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.ID) // runner-2 has fewer pods
	})

	t.Run("ignores runners from different organization", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		r := &runner.Runner{
			ID: 1, OrganizationID: 999, NodeID: "runner-other-org",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			AvailableAgents: runner.StringSlice{"claude-code"},
			Visibility: runner.VisibilityOrganization,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now(), PodCount: 0})

		_, err := service.SelectAvailableRunnerForAgent(ctx, 1, 1, "claude-code")
		assert.Error(t, err)
	})

	t.Run("ignores disabled runners in cache", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		r := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-disabled",
			Status: runner.RunnerStatusOnline, IsEnabled: false, MaxConcurrentPods: 5,
			AvailableAgents: runner.StringSlice{"claude-code"},
			Visibility: runner.VisibilityOrganization,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now(), PodCount: 0})

		_, err := service.SelectAvailableRunnerForAgent(ctx, 1, 1, "claude-code")
		assert.Error(t, err)
	})

	t.Run("ignores runners at capacity in cache", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		r := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-full",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 2,
			AvailableAgents: runner.StringSlice{"claude-code"},
			Visibility: runner.VisibilityOrganization,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now(), PodCount: 2})

		_, err := service.SelectAvailableRunnerForAgent(ctx, 1, 1, "claude-code")
		assert.Error(t, err)
	})

	t.Run("ignores stale runners in cache", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		r := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-stale",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			AvailableAgents: runner.StringSlice{"claude-code"},
			Visibility: runner.VisibilityOrganization,
		}
		// Last ping was 2 minutes ago → stale (>90s)
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now().Add(-2 * time.Minute), PodCount: 0})

		_, err := service.SelectAvailableRunnerForAgent(ctx, 1, 1, "claude-code")
		assert.Error(t, err)
	})

	t.Run("returns private runner only to registrant", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		registrantUserID := int64(10)
		otherUserID := int64(20)

		r := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-private",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			AvailableAgents:    runner.StringSlice{"claude-code"},
			Visibility:         runner.VisibilityPrivate,
			RegisteredByUserID: &registrantUserID,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now(), PodCount: 0})

		// Registrant should see it
		result, err := service.SelectAvailableRunnerForAgent(ctx, 1, registrantUserID, "claude-code")
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.ID)

		// Other user should NOT see it
		_, err = service.SelectAvailableRunnerForAgent(ctx, 1, otherUserID, "claude-code")
		assert.Error(t, err)
	})
}

func TestSelectAvailableRunnerVisibility(t *testing.T) {
	ctx := context.Background()

	t.Run("private runner visible to registrant in cache", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		registrantUserID := int64(10)
		r := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-private",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			Visibility:         runner.VisibilityPrivate,
			RegisteredByUserID: &registrantUserID,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now(), PodCount: 0})

		result, err := service.SelectAvailableRunner(ctx, 1, registrantUserID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.ID)
	})

	t.Run("private runner invisible to other users in cache", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		registrantUserID := int64(10)
		otherUserID := int64(20)
		r := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-private",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			Visibility:         runner.VisibilityPrivate,
			RegisteredByUserID: &registrantUserID,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now(), PodCount: 0})

		_, err := service.SelectAvailableRunner(ctx, 1, otherUserID)
		assert.Equal(t, ErrRunnerOffline, err)
	})

	t.Run("organization runner visible to any org member in cache", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		registrantUserID := int64(10)
		otherUserID := int64(20)
		r := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-org",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			Visibility:         runner.VisibilityOrganization,
			RegisteredByUserID: &registrantUserID,
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now(), PodCount: 0})

		// Both users should see it
		result1, err := service.SelectAvailableRunner(ctx, 1, registrantUserID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result1.ID)

		result2, err := service.SelectAvailableRunner(ctx, 1, otherUserID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result2.ID)
	})

	t.Run("private runner without RegisteredByUserID invisible to all", func(t *testing.T) {
		db := setupTestDB(t)
		service := newTestService(db)

		r := &runner.Runner{
			ID: 1, OrganizationID: 1, NodeID: "runner-private-no-owner",
			Status: runner.RunnerStatusOnline, IsEnabled: true, MaxConcurrentPods: 5,
			Visibility:         runner.VisibilityPrivate,
			RegisteredByUserID: nil, // No registrant
		}
		service.activeRunners.Store(r.ID, &ActiveRunner{Runner: r, LastPing: time.Now(), PodCount: 0})

		_, err := service.SelectAvailableRunner(ctx, 1, 1)
		assert.Equal(t, ErrRunnerOffline, err)
	})
}

func TestUpdateRunnerVisibility(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("updates visibility from organization to private", func(t *testing.T) {
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "runner-vis-1",
			Status:         runner.RunnerStatusOffline,
			IsEnabled:      true,
		}
		require.NoError(t, db.Create(r).Error)

		vis := runner.VisibilityPrivate
		updated, err := service.UpdateRunner(ctx, r.ID, RunnerUpdateInput{Visibility: &vis})
		require.NoError(t, err)
		assert.Equal(t, runner.VisibilityPrivate, updated.Visibility)
	})

	t.Run("updates visibility from private to organization", func(t *testing.T) {
		userID := int64(1)
		r := &runner.Runner{
			OrganizationID:     1,
			NodeID:             "runner-vis-2",
			Status:             runner.RunnerStatusOffline,
			IsEnabled:          true,
			Visibility:         runner.VisibilityPrivate,
			RegisteredByUserID: &userID,
		}
		require.NoError(t, db.Create(r).Error)

		vis := runner.VisibilityOrganization
		updated, err := service.UpdateRunner(ctx, r.ID, RunnerUpdateInput{Visibility: &vis})
		require.NoError(t, err)
		assert.Equal(t, runner.VisibilityOrganization, updated.Visibility)
	})

	t.Run("ignores invalid visibility value", func(t *testing.T) {
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "runner-vis-3",
			Status:         runner.RunnerStatusOffline,
			IsEnabled:      true,
		}
		require.NoError(t, db.Create(r).Error)

		vis := "invalid-visibility"
		updated, err := service.UpdateRunner(ctx, r.ID, RunnerUpdateInput{Visibility: &vis})
		require.NoError(t, err)
		// Should remain default "organization" (from DB default)
		assert.Equal(t, runner.VisibilityOrganization, updated.Visibility)
	})
}
