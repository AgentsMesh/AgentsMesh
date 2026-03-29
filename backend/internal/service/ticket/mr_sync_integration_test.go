package ticket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMRData(t *testing.T) {
	db := setupMRSyncTestDB(t)
	service := newTestMRSyncService(db, nil)

	t.Run("converts git.MergeRequest to MRData", func(t *testing.T) {
		mergedAt := time.Now()
		mr := &git.MergeRequest{
			IID:            1,
			WebURL:         "https://gitlab.com/org/repo/-/merge_requests/1",
			Title:          "Test MR",
			SourceBranch:   "feature/test",
			TargetBranch:   "main",
			State:          "merged",
			PipelineStatus: "success",
			PipelineID:     12345,
			PipelineURL:    "https://gitlab.com/org/repo/-/pipelines/12345",
			MergeCommitSHA: "abc123",
			MergedAt:       &mergedAt,
		}

		data := service.buildMRData(mr)
		assert.Equal(t, mr.IID, data.IID)
		assert.Equal(t, mr.WebURL, data.WebURL)
		assert.Equal(t, mr.Title, data.Title)
		assert.Equal(t, mr.SourceBranch, data.SourceBranch)
		assert.Equal(t, mr.TargetBranch, data.TargetBranch)
		assert.Equal(t, mr.State, data.State)
		assert.NotNil(t, data.PipelineStatus)
		assert.Equal(t, "success", *data.PipelineStatus)
		assert.NotNil(t, data.PipelineID)
		assert.Equal(t, int64(12345), *data.PipelineID)
		assert.NotNil(t, data.PipelineURL)
		assert.NotNil(t, data.MergeCommitSHA)
		assert.NotNil(t, data.MergedAt)
	})

	t.Run("handles empty optional fields", func(t *testing.T) {
		mr := &git.MergeRequest{
			IID:          2,
			WebURL:       "https://gitlab.com/org/repo/-/merge_requests/2",
			Title:        "Test MR",
			SourceBranch: "feature/test",
			TargetBranch: "main",
			State:        "opened",
		}

		data := service.buildMRData(mr)
		assert.Nil(t, data.PipelineStatus)
		assert.Nil(t, data.PipelineID)
		assert.Nil(t, data.PipelineURL)
		assert.Nil(t, data.MergeCommitSHA)
		assert.Nil(t, data.MergedAt)
	})
}

func TestUpdateMRFromData(t *testing.T) {
	db := setupMRSyncTestDB(t)
	service := newTestMRSyncService(db, nil)

	t.Run("updates MR fields from data", func(t *testing.T) {
		mr := &ticket.MergeRequest{
			Title: "Old Title",
			State: "opened",
		}

		pipelineStatus := "success"
		pipelineID := int64(12345)
		data := &MRData{
			Title:          "New Title",
			State:          "merged",
			PipelineStatus: &pipelineStatus,
			PipelineID:     &pipelineID,
		}

		service.updateMRFromData(mr, data)
		assert.Equal(t, "New Title", mr.Title)
		assert.Equal(t, "merged", mr.State)
		assert.NotNil(t, mr.PipelineStatus)
		assert.Equal(t, "success", *mr.PipelineStatus)
		assert.NotNil(t, mr.LastSyncedAt)
	})
}

func TestMRSyncServiceIntegration(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)

	t.Run("full sync flow with mock provider", func(t *testing.T) {
		provider := &MockGitProvider{
			ListMRsFunc: func(ctx context.Context, projectID, sourceBranch, state string) ([]*git.MergeRequest, error) {
				return []*git.MergeRequest{
					{
						IID:          5,
						WebURL:       "https://gitlab.com/org/repo/-/merge_requests/5",
						Title:        "Auto-discovered MR",
						SourceBranch: sourceBranch,
						TargetBranch: "main",
						State:        "opened",
					},
				}, nil
			},
			GetMRFunc: func(ctx context.Context, projectID string, iid int) (*git.MergeRequest, error) {
				return &git.MergeRequest{
					IID:          iid,
					WebURL:       "https://gitlab.com/org/repo/-/merge_requests/" + string(rune(iid)),
					Title:        "Synced MR",
					SourceBranch: "feature/test",
					TargetBranch: "main",
					State:        "merged",
				}, nil
			},
		}

		service := newTestMRSyncService(db, provider)

		// Create repo
		repo := &gitprovider.Repository{
			OrganizationID: 1,
			Name:           "test-repo",
			Slug:       "org/test-repo",
			ExternalID:     "456",
		}
		db.Create(repo)

		// Create ticket
		tkt := &ticket.Ticket{
			OrganizationID: 1,
			Slug:     "INT-1",
			Title:          "Integration Test Ticket",
	
			Status:         ticket.TicketStatusInProgress,
			Priority:       ticket.TicketPriorityMedium,
			RepositoryID:   &repo.ID,
		}
		db.Create(tkt)

		// Test FindOrCreateMR
		mrData := &MRData{
			IID:          10,
			WebURL:       "https://gitlab.com/org/test-repo/-/merge_requests/10",
			Title:        "Integration Test MR",
			SourceBranch: "feature/INT-1",
			TargetBranch: "main",
			State:        "opened",
		}

		mr, err := service.FindOrCreateMR(ctx, 1, tkt, mrData, nil)
		require.NoError(t, err)
		assert.NotNil(t, mr)
		assert.Equal(t, "Integration Test MR", mr.Title)

		// Test GetTicketMRs
		mrs, err := service.GetTicketMRs(ctx, tkt.ID)
		require.NoError(t, err)
		assert.Len(t, mrs, 1)

		// Test FindTicketByBranch
		foundTicket, err := service.FindTicketByBranch(ctx, int64(1), "feature/INT-1")
		require.NoError(t, err)
		assert.NotNil(t, foundTicket)
		assert.Equal(t, tkt.ID, foundTicket.ID)
	})

	t.Run("handles git provider errors gracefully", func(t *testing.T) {
		provider := &MockGitProvider{
			GetMRFunc: func(ctx context.Context, projectID string, iid int) (*git.MergeRequest, error) {
				return nil, errors.New("network error")
			},
		}

		service := newTestMRSyncService(db, provider)

		// Create repo
		repo := &gitprovider.Repository{
			OrganizationID: 1,
			Name:           "error-repo",
			Slug:       "org/error-repo",
			ExternalID:     "789",
		}
		db.Create(repo)

		// Create ticket
		tkt := &ticket.Ticket{
			OrganizationID: 1,
			Slug:     "ERR-1",
			Title:          "Error Test Ticket",
	
			Status:         ticket.TicketStatusInProgress,
			Priority:       ticket.TicketPriorityMedium,
			RepositoryID:   &repo.ID,
		}
		db.Create(tkt)

		// Create MR
		mr := &ticket.MergeRequest{
			OrganizationID: 1,
			TicketID:       &tkt.ID,
			MRIID:          100,
			MRURL:          "https://gitlab.com/org/error-repo/-/merge_requests/100",
			SourceBranch:   "feature/test",
			TargetBranch:   "main",
			State:          "opened",
		}
		db.Create(mr)

		// SyncMRByURL should return error
		_, err := service.SyncMRByURL(ctx, mr.MRURL)
		assert.Error(t, err)
	})
}
