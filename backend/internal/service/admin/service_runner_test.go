package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRunners(t *testing.T) {
	t.Run("should list runners with pagination", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "node-1", OrganizationID: 1}
		db.runners[2] = &runner.Runner{ID: 2, NodeID: "node-2", OrganizationID: 1}
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.totalRunners = 2

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		// Note: The mock returns totalRunners as the count
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Data, 2)
	})
}

func TestGetRunner(t *testing.T) {
	t.Run("should return runner when found", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}

		svc := NewService(db)
		r, err := svc.GetRunner(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), r.ID)
		assert.Equal(t, "test-node", r.NodeID)
	})

	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		r, err := svc.GetRunner(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, r)
	})
}

func TestGetRunnerWithOrg(t *testing.T) {
	t.Run("should return runner with organization", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", OrganizationID: 1}
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}

		svc := NewService(db)
		rwo, err := svc.GetRunnerWithOrg(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), rwo.Runner.ID)
		assert.NotNil(t, rwo.Organization)
		assert.Equal(t, "Test Org", rwo.Organization.Name)
	})
}

func TestDisableRunner(t *testing.T) {
	t.Run("should disable runner successfully", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, IsEnabled: true}

		svc := NewService(db)
		r, err := svc.DisableRunner(context.Background(), 1)

		require.NoError(t, err)
		assert.False(t, r.IsEnabled)
	})

	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		r, err := svc.DisableRunner(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, r)
	})
}

func TestEnableRunner(t *testing.T) {
	t.Run("should enable runner successfully", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, IsEnabled: false}

		svc := NewService(db)
		r, err := svc.EnableRunner(context.Background(), 1)

		require.NoError(t, err)
		assert.True(t, r.IsEnabled)
	})
}

func TestDeleteRunner(t *testing.T) {
	t.Run("should delete runner successfully when no active pods", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 0

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), r.ID)
	})

	t.Run("should return error when runner has active pods", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 3

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerHasActivePods, err)
		assert.Nil(t, r)
	})

	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, r)
	})
}

func TestListRunners_WithFilters(t *testing.T) {
	t.Run("should filter by search term", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.totalRunners = 1

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
			Search:   "test",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by status", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", Status: "online"}
		db.totalRunners = 1

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
			Status:   "online",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by organization ID", func(t *testing.T) {
		db := newMockDB()
		orgID := int64(1)
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", OrganizationID: 1}
		db.totalRunners = 1

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
			OrgID:    &orgID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should return error when count fails", func(t *testing.T) {
		db := newMockDB()
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("should return error when find fails", func(t *testing.T) {
		db := newMockDB()
		db.findErr = errors.New("find failed")

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDeleteRunner_ErrorPaths(t *testing.T) {
	t.Run("should return error when count pods fails", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check pods")
		assert.Nil(t, r)
	})

	t.Run("should return error when delete fails", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 0
		db.deleteErr = errors.New("delete failed")

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestDisableRunner_ErrorPaths(t *testing.T) {
	t.Run("should return error when save fails", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, IsEnabled: true}
		db.saveErr = errors.New("save failed")

		svc := NewService(db)
		r, err := svc.DisableRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestEnableRunner_ErrorPaths(t *testing.T) {
	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		r, err := svc.EnableRunner(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, r)
	})

	t.Run("should return error when save fails", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, IsEnabled: false}
		db.saveErr = errors.New("save failed")

		svc := NewService(db)
		r, err := svc.EnableRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestGetRunnerWithOrg_ErrorPaths(t *testing.T) {
	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		rwo, err := svc.GetRunnerWithOrg(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, rwo)
	})
}
