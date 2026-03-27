package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		assert.Equal(t, int64(2), result.ID)
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

		result, err := service.SelectAvailableRunnerForAgent(ctx, 1, registrantUserID, "claude-code")
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.ID)

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
			RegisteredByUserID: nil,
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
		assert.Equal(t, runner.VisibilityOrganization, updated.Visibility)
	})
}
