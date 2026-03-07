package ticket

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMRSyncService(t *testing.T) {
	db := setupMRSyncTestDB(t)
	provider := &MockGitProvider{}

	service := newTestMRSyncService(db, provider)
	assert.NotNil(t, service)
}

func TestFindOrCreateMR(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)
	provider := &MockGitProvider{}
	service := newTestMRSyncService(db, provider)

	// Create a ticket
	tkt := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "MR-1",
		Title:          "Test Ticket",

		Status:         ticket.TicketStatusInProgress,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(tkt)

	t.Run("creates new MR", func(t *testing.T) {
		mrData := &MRData{
			IID:          1,
			WebURL:       "https://gitlab.com/org/repo/-/merge_requests/1",
			Title:        "Feature: Add new feature",
			SourceBranch: "feature/MR-1",
			TargetBranch: "main",
			State:        "opened",
		}

		mr, err := service.FindOrCreateMR(ctx, 1, tkt, mrData, nil)
		require.NoError(t, err)
		assert.NotNil(t, mr)
		assert.NotNil(t, mr.TicketID)
		assert.Equal(t, tkt.ID, *mr.TicketID)
		assert.Equal(t, mrData.WebURL, mr.MRURL)
		assert.Equal(t, mrData.SourceBranch, mr.SourceBranch)
		assert.Equal(t, mrData.State, mr.State)
	})

	t.Run("updates existing MR", func(t *testing.T) {
		mrData := &MRData{
			IID:          1,
			WebURL:       "https://gitlab.com/org/repo/-/merge_requests/1",
			Title:        "Updated Title",
			SourceBranch: "feature/MR-1",
			TargetBranch: "main",
			State:        "merged",
		}

		mr, err := service.FindOrCreateMR(ctx, 1, tkt, mrData, nil)
		require.NoError(t, err)
		assert.NotNil(t, mr)
		assert.Equal(t, "Updated Title", mr.Title)
		assert.Equal(t, "merged", mr.State)
	})

	t.Run("returns error for empty URL", func(t *testing.T) {
		mrData := &MRData{
			IID:   1,
			Title: "No URL",
		}

		_, err := service.FindOrCreateMR(ctx, 1, tkt, mrData, nil)
		assert.Error(t, err)
	})

	t.Run("sets pod ID on new MR", func(t *testing.T) {
		mrData := &MRData{
			IID:          2,
			WebURL:       "https://gitlab.com/org/repo/-/merge_requests/2",
			Title:        "Feature with pod",
			SourceBranch: "feature/MR-2",
			TargetBranch: "main",
			State:        "opened",
		}

		podID := int64(100)
		mr, err := service.FindOrCreateMR(ctx, 1, tkt, mrData, &podID)
		require.NoError(t, err)
		assert.NotNil(t, mr.PodID)
		assert.Equal(t, podID, *mr.PodID)
	})

	t.Run("handles pipeline info", func(t *testing.T) {
		pipelineStatus := "success"
		pipelineID := int64(12345)
		pipelineURL := "https://gitlab.com/org/repo/-/pipelines/12345"
		mergeCommitSHA := "abc123"
		mergedAt := time.Now()

		mrData := &MRData{
			IID:            3,
			WebURL:         "https://gitlab.com/org/repo/-/merge_requests/3",
			Title:          "Feature with pipeline",
			SourceBranch:   "feature/MR-3",
			TargetBranch:   "main",
			State:          "merged",
			PipelineStatus: &pipelineStatus,
			PipelineID:     &pipelineID,
			PipelineURL:    &pipelineURL,
			MergeCommitSHA: &mergeCommitSHA,
			MergedAt:       &mergedAt,
		}

		mr, err := service.FindOrCreateMR(ctx, 1, tkt, mrData, nil)
		require.NoError(t, err)
		assert.NotNil(t, mr.PipelineStatus)
		assert.Equal(t, pipelineStatus, *mr.PipelineStatus)
		assert.NotNil(t, mr.PipelineID)
		assert.Equal(t, pipelineID, *mr.PipelineID)
	})
}
