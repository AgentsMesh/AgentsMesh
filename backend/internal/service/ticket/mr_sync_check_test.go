package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPodForNewMR(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)

	t.Run("returns nil for pod without branch", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := newTestMRSyncService(db, provider)

		pod := &agentpod.Pod{
			ID:             1,
			OrganizationID: 1,
			BranchName:     nil,
		}

		mr, err := service.CheckPodForNewMR(ctx, pod)
		assert.NoError(t, err)
		assert.Nil(t, mr)
	})

	t.Run("returns nil for pod without ticket", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := newTestMRSyncService(db, provider)

		branchName := "feature/test"
		pod := &agentpod.Pod{
			ID:             2,
			OrganizationID: 1,
			BranchName:     &branchName,
			TicketID:       nil,
		}

		mr, err := service.CheckPodForNewMR(ctx, pod)
		assert.NoError(t, err)
		assert.Nil(t, mr)
	})

	t.Run("returns error when git provider is nil", func(t *testing.T) {
		service := newTestMRSyncService(db, nil)

		branchName := "feature/test"
		ticketID := int64(1)
		pod := &agentpod.Pod{
			ID:             3,
			OrganizationID: 1,
			BranchName:     &branchName,
			TicketID:       &ticketID,
		}

		_, err := service.CheckPodForNewMR(ctx, pod)
		assert.Error(t, err)
		assert.Equal(t, ErrNoGitProvider, err)
	})
}

func TestBatchCheckPods(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)

	t.Run("returns error when git provider is nil", func(t *testing.T) {
		service := newTestMRSyncService(db, nil)

		_, err := service.BatchCheckPods(ctx)
		assert.Error(t, err)
		assert.Equal(t, ErrNoGitProvider, err)
	})

	t.Run("returns empty when no matching pods", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := newTestMRSyncService(db, provider)

		mrs, err := service.BatchCheckPods(ctx)
		require.NoError(t, err)
		assert.Empty(t, mrs)
	})
}

func TestBatchSyncMRStatus(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)

	t.Run("returns error when git provider is nil", func(t *testing.T) {
		service := newTestMRSyncService(db, nil)

		_, err := service.BatchSyncMRStatus(ctx)
		assert.Error(t, err)
		assert.Equal(t, ErrNoGitProvider, err)
	})

	t.Run("returns empty when no open MRs", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := newTestMRSyncService(db, provider)

		mrs, err := service.BatchSyncMRStatus(ctx)
		require.NoError(t, err)
		assert.Empty(t, mrs)
	})
}
