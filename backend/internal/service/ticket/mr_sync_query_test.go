package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTicketMRs(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)
	service := newTestMRSyncService(db, nil)

	// Create a ticket
	tkt := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "TMR-1",
		Title:          "Test Ticket",

		Status:         ticket.TicketStatusInProgress,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(tkt)

	// Create MRs for the ticket
	mr1 := &ticket.MergeRequest{
		OrganizationID: 1,
		TicketID:       &tkt.ID,
		MRIID:          1,
		MRURL:          "https://gitlab.com/org/repo/-/merge_requests/1",
		SourceBranch:   "feature/1",
		TargetBranch:   "main",
		State:          "merged",
	}
	mr2 := &ticket.MergeRequest{
		OrganizationID: 1,
		TicketID:       &tkt.ID,
		MRIID:          2,
		MRURL:          "https://gitlab.com/org/repo/-/merge_requests/2",
		SourceBranch:   "feature/2",
		TargetBranch:   "main",
		State:          "opened",
	}
	db.Create(mr1)
	db.Create(mr2)

	t.Run("returns all MRs for ticket", func(t *testing.T) {
		mrs, err := service.GetTicketMRs(ctx, tkt.ID)
		require.NoError(t, err)
		assert.Len(t, mrs, 2)
	})

	t.Run("returns empty for ticket without MRs", func(t *testing.T) {
		mrs, err := service.GetTicketMRs(ctx, 9999)
		require.NoError(t, err)
		assert.Empty(t, mrs)
	})
}

func TestGetPodMRs(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)
	service := newTestMRSyncService(db, nil)

	podID := int64(100)
	ticketID := int64(1)

	// Create MRs for a pod
	mr := &ticket.MergeRequest{
		OrganizationID: 1,
		TicketID:       &ticketID,
		PodID:          &podID,
		MRIID:          1,
		MRURL:          "https://gitlab.com/org/repo/-/merge_requests/1",
		SourceBranch:   "feature/1",
		TargetBranch:   "main",
		State:          "opened",
	}
	db.Create(mr)

	t.Run("returns MRs for pod", func(t *testing.T) {
		mrs, err := service.GetPodMRs(ctx, podID)
		require.NoError(t, err)
		assert.Len(t, mrs, 1)
	})

	t.Run("returns empty for pod without MRs", func(t *testing.T) {
		mrs, err := service.GetPodMRs(ctx, 9999)
		require.NoError(t, err)
		assert.Empty(t, mrs)
	})
}

func TestFindTicketByBranch(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)
	service := newTestMRSyncService(db, nil)

	// Create a ticket
	tkt := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "PRJ-123",
		Title:          "Test Ticket",

		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(tkt)

	t.Run("finds ticket by branch with slug", func(t *testing.T) {
		result, err := service.FindTicketByBranch(ctx, int64(1), "feature/PRJ-123-new-feature")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, tkt.ID, result.ID)
	})

	t.Run("finds ticket by exact slug branch", func(t *testing.T) {
		result, err := service.FindTicketByBranch(ctx, int64(1), "PRJ-123")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, tkt.ID, result.ID)
	})

	t.Run("returns nil for branch without slug", func(t *testing.T) {
		result, err := service.FindTicketByBranch(ctx, int64(1), "feature/some-branch")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns nil for non-existent ticket", func(t *testing.T) {
		result, err := service.FindTicketByBranch(ctx, int64(1), "feature/NONEXISTENT-999")
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestSyncMRByURL(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)

	t.Run("returns error for non-existent MR", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := newTestMRSyncService(db, provider)

		_, err := service.SyncMRByURL(ctx, "https://gitlab.com/org/repo/-/merge_requests/999")
		assert.Error(t, err)
		assert.Equal(t, ErrMRNotFound, err)
	})

	t.Run("syncs existing MR", func(t *testing.T) {
		provider := &MockGitProvider{
			GetMRFunc: func(ctx context.Context, projectID string, iid int) (*git.MergeRequest, error) {
				return &git.MergeRequest{
					IID:          iid,
					WebURL:       "https://gitlab.com/org/repo/-/merge_requests/1",
					Title:        "Updated Title",
					SourceBranch: "feature/test",
					TargetBranch: "main",
					State:        "merged",
				}, nil
			},
		}
		service := newTestMRSyncService(db, provider)

		// Create repository
		repoID := int64(10)
		repo := &gitprovider.Repository{
			OrganizationID: 1,
			Name:           "repo",
			Slug:       "org/repo",
			ExternalID:     "123",
		}
		db.Create(repo)
		repoID = repo.ID

		// Create ticket with repository
		tkt := &ticket.Ticket{
			OrganizationID: 1,
			Slug:     "SYNC-1",
			Title:          "Test Ticket",
	
			Status:         ticket.TicketStatusInProgress,
			Priority:       ticket.TicketPriorityMedium,
			RepositoryID:   &repoID,
		}
		db.Create(tkt)

		// Create MR
		mr := &ticket.MergeRequest{
			OrganizationID: 1,
			TicketID:       &tkt.ID,
			MRIID:          1,
			MRURL:          "https://gitlab.com/org/repo/-/merge_requests/1",
			SourceBranch:   "feature/test",
			TargetBranch:   "main",
			State:          "opened",
			Title:          "Original Title",
		}
		db.Create(mr)

		result, err := service.SyncMRByURL(ctx, mr.MRURL)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", result.Title)
		assert.Equal(t, "merged", result.State)
	})

	t.Run("returns error when ticket has no repository", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := newTestMRSyncService(db, provider)

		// Create ticket without repository
		tkt := &ticket.Ticket{
			OrganizationID: 1,
			Slug:     "NOREPO-1",
			Title:          "Test Ticket",
	
			Status:         ticket.TicketStatusInProgress,
			Priority:       ticket.TicketPriorityMedium,
		}
		db.Create(tkt)

		// Create MR
		mr := &ticket.MergeRequest{
			OrganizationID: 1,
			TicketID:       &tkt.ID,
			MRIID:          99,
			MRURL:          "https://gitlab.com/org/repo/-/merge_requests/99",
			SourceBranch:   "feature/test",
			TargetBranch:   "main",
			State:          "opened",
		}
		db.Create(mr)

		_, err := service.SyncMRByURL(ctx, mr.MRURL)
		assert.Error(t, err)
		assert.Equal(t, ErrNoRepositoryLink, err)
	})
}
