package ticket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/repository"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockGitProvider implements git.Provider for testing
type MockGitProvider struct {
	ListMRsFunc   func(ctx context.Context, projectID, sourceBranch, state string) ([]*git.MergeRequest, error)
	GetMRFunc     func(ctx context.Context, projectID string, iid int) (*git.MergeRequest, error)
	CreateMRFunc  func(ctx context.Context, projectID string, req *git.CreateMRRequest) (*git.MergeRequest, error)
	GetProjectFunc func(ctx context.Context, projectID string) (*git.Project, error)
	GetFileFunc   func(ctx context.Context, projectID, branch, path string) ([]byte, error)
}

func (m *MockGitProvider) ListMergeRequestsByBranch(ctx context.Context, projectID, sourceBranch, state string) ([]*git.MergeRequest, error) {
	if m.ListMRsFunc != nil {
		return m.ListMRsFunc(ctx, projectID, sourceBranch, state)
	}
	return nil, nil
}

func (m *MockGitProvider) GetMergeRequest(ctx context.Context, projectID string, iid int) (*git.MergeRequest, error) {
	if m.GetMRFunc != nil {
		return m.GetMRFunc(ctx, projectID, iid)
	}
	return nil, nil
}

func (m *MockGitProvider) CreateMergeRequest(ctx context.Context, req *git.CreateMRRequest) (*git.MergeRequest, error) {
	if m.CreateMRFunc != nil {
		return m.CreateMRFunc(ctx, req.ProjectID, req)
	}
	return nil, nil
}

func (m *MockGitProvider) GetProject(ctx context.Context, projectID string) (*git.Project, error) {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockGitProvider) GetFileContent(ctx context.Context, projectID, filePath, ref string) ([]byte, error) {
	if m.GetFileFunc != nil {
		return m.GetFileFunc(ctx, projectID, filePath, ref)
	}
	return nil, nil
}

// Implement remaining Provider interface methods with no-op implementations
func (m *MockGitProvider) GetCurrentUser(ctx context.Context) (*git.User, error) { return nil, nil }
func (m *MockGitProvider) ListProjects(ctx context.Context, page, perPage int) ([]*git.Project, error) { return nil, nil }
func (m *MockGitProvider) SearchProjects(ctx context.Context, query string, page, perPage int) ([]*git.Project, error) { return nil, nil }
func (m *MockGitProvider) ListBranches(ctx context.Context, projectID string) ([]*git.Branch, error) { return nil, nil }
func (m *MockGitProvider) GetBranch(ctx context.Context, projectID, branchName string) (*git.Branch, error) { return nil, nil }
func (m *MockGitProvider) CreateBranch(ctx context.Context, projectID, branchName, ref string) (*git.Branch, error) { return nil, nil }
func (m *MockGitProvider) DeleteBranch(ctx context.Context, projectID, branchName string) error { return nil }
func (m *MockGitProvider) ListMergeRequests(ctx context.Context, projectID string, state string, page, perPage int) ([]*git.MergeRequest, error) { return nil, nil }
func (m *MockGitProvider) UpdateMergeRequest(ctx context.Context, projectID string, mrIID int, title, description string) (*git.MergeRequest, error) { return nil, nil }
func (m *MockGitProvider) MergeMergeRequest(ctx context.Context, projectID string, mrIID int) (*git.MergeRequest, error) { return nil, nil }
func (m *MockGitProvider) CloseMergeRequest(ctx context.Context, projectID string, mrIID int) (*git.MergeRequest, error) { return nil, nil }
func (m *MockGitProvider) GetCommit(ctx context.Context, projectID, sha string) (*git.Commit, error) { return nil, nil }
func (m *MockGitProvider) ListCommits(ctx context.Context, projectID, branch string, page, perPage int) ([]*git.Commit, error) { return nil, nil }
func (m *MockGitProvider) RegisterWebhook(ctx context.Context, projectID string, config *git.WebhookConfig) (string, error) { return "", nil }
func (m *MockGitProvider) DeleteWebhook(ctx context.Context, projectID, webhookID string) error { return nil }
func (m *MockGitProvider) TriggerPipeline(ctx context.Context, projectID string, req *git.TriggerPipelineRequest) (*git.Pipeline, error) { return nil, nil }
func (m *MockGitProvider) GetPipeline(ctx context.Context, projectID string, pipelineID int) (*git.Pipeline, error) { return nil, nil }
func (m *MockGitProvider) ListPipelines(ctx context.Context, projectID string, ref, status string, page, perPage int) ([]*git.Pipeline, error) { return nil, nil }
func (m *MockGitProvider) CancelPipeline(ctx context.Context, projectID string, pipelineID int) (*git.Pipeline, error) { return nil, nil }
func (m *MockGitProvider) RetryPipeline(ctx context.Context, projectID string, pipelineID int) (*git.Pipeline, error) { return nil, nil }
func (m *MockGitProvider) GetJob(ctx context.Context, projectID string, jobID int) (*git.Job, error) { return nil, nil }
func (m *MockGitProvider) ListPipelineJobs(ctx context.Context, projectID string, pipelineID int) ([]*git.Job, error) { return nil, nil }
func (m *MockGitProvider) RetryJob(ctx context.Context, projectID string, jobID int) (*git.Job, error) { return nil, nil }
func (m *MockGitProvider) CancelJob(ctx context.Context, projectID string, jobID int) (*git.Job, error) { return nil, nil }
func (m *MockGitProvider) GetJobTrace(ctx context.Context, projectID string, jobID int) (string, error) { return "", nil }
func (m *MockGitProvider) GetJobArtifact(ctx context.Context, projectID string, jobID int, artifactPath string) ([]byte, error) { return nil, nil }
func (m *MockGitProvider) DownloadJobArtifacts(ctx context.Context, projectID string, jobID int) ([]byte, error) { return nil, nil }

func setupMRSyncTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			number INTEGER NOT NULL DEFAULT 0,
			identifier TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'task',
			title TEXT NOT NULL,
			description TEXT,
			content TEXT,
			status TEXT NOT NULL DEFAULT 'backlog',
			priority TEXT NOT NULL DEFAULT 'none',
			severity TEXT,
			estimate INTEGER,
			due_date DATETIME,
			started_at DATETIME,
			completed_at DATETIME,
			repository_id INTEGER,
			reporter_id INTEGER NOT NULL DEFAULT 0,
			parent_ticket_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_merge_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			ticket_id INTEGER NOT NULL,
			pod_id INTEGER,
			mri_id INTEGER NOT NULL,
			mr_url TEXT NOT NULL UNIQUE,
			source_branch TEXT NOT NULL,
			target_branch TEXT NOT NULL DEFAULT 'main',
			title TEXT,
			state TEXT NOT NULL DEFAULT 'opened',
			pipeline_status TEXT,
			pipeline_id INTEGER,
			pipeline_url TEXT,
			merge_commit_sha TEXT,
			merged_at DATETIME,
			merged_by_id INTEGER,
			last_synced_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL DEFAULT 'github',
			provider_base_url TEXT NOT NULL DEFAULT 'https://github.com',
			clone_url TEXT,
			external_id TEXT,
			name TEXT NOT NULL,
			full_path TEXT NOT NULL,
			default_branch TEXT DEFAULT 'main',
			ticket_prefix TEXT,
			visibility TEXT NOT NULL DEFAULT 'organization',
			imported_by_user_id INTEGER,
			is_active INTEGER DEFAULT 1,
			deleted_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			pod_key TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'initializing',
			branch_name TEXT,
			ticket_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	return db
}

func TestNewMRSyncService(t *testing.T) {
	db := setupMRSyncTestDB(t)
	provider := &MockGitProvider{}

	service := NewMRSyncService(db, provider)
	assert.NotNil(t, service)
}

func TestFindOrCreateMR(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)
	provider := &MockGitProvider{}
	service := NewMRSyncService(db, provider)

	// Create a ticket
	tkt := &ticket.Ticket{
		OrganizationID: 1,
		Identifier:     "MR-1",
		Title:          "Test Ticket",
		Type:           ticket.TicketTypeTask,
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
		assert.Equal(t, tkt.ID, mr.TicketID)
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

func TestCheckPodForNewMR(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)

	t.Run("returns nil for pod without branch", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := NewMRSyncService(db, provider)

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
		service := NewMRSyncService(db, provider)

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
		service := NewMRSyncService(db, nil)

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
		service := NewMRSyncService(db, nil)

		_, err := service.BatchCheckPods(ctx)
		assert.Error(t, err)
		assert.Equal(t, ErrNoGitProvider, err)
	})

	t.Run("returns empty when no matching pods", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := NewMRSyncService(db, provider)

		mrs, err := service.BatchCheckPods(ctx)
		require.NoError(t, err)
		assert.Empty(t, mrs)
	})
}

func TestBatchSyncMRStatus(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)

	t.Run("returns error when git provider is nil", func(t *testing.T) {
		service := NewMRSyncService(db, nil)

		_, err := service.BatchSyncMRStatus(ctx)
		assert.Error(t, err)
		assert.Equal(t, ErrNoGitProvider, err)
	})

	t.Run("returns empty when no open MRs", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := NewMRSyncService(db, provider)

		mrs, err := service.BatchSyncMRStatus(ctx)
		require.NoError(t, err)
		assert.Empty(t, mrs)
	})
}

func TestGetTicketMRs(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)
	service := NewMRSyncService(db, nil)

	// Create a ticket
	tkt := &ticket.Ticket{
		OrganizationID: 1,
		Identifier:     "TMR-1",
		Title:          "Test Ticket",
		Type:           ticket.TicketTypeTask,
		Status:         ticket.TicketStatusInProgress,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(tkt)

	// Create MRs for the ticket
	mr1 := &ticket.MergeRequest{
		OrganizationID: 1,
		TicketID:       tkt.ID,
		MRIID:          1,
		MRURL:          "https://gitlab.com/org/repo/-/merge_requests/1",
		SourceBranch:   "feature/1",
		TargetBranch:   "main",
		State:          "merged",
	}
	mr2 := &ticket.MergeRequest{
		OrganizationID: 1,
		TicketID:       tkt.ID,
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
	service := NewMRSyncService(db, nil)

	podID := int64(100)

	// Create MRs for a pod
	mr := &ticket.MergeRequest{
		OrganizationID: 1,
		TicketID:       1,
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
	service := NewMRSyncService(db, nil)

	// Create a ticket
	tkt := &ticket.Ticket{
		OrganizationID: 1,
		Identifier:     "PRJ-123",
		Title:          "Test Ticket",
		Type:           ticket.TicketTypeTask,
		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(tkt)

	t.Run("finds ticket by branch with identifier", func(t *testing.T) {
		result, err := service.FindTicketByBranch(ctx, "feature/PRJ-123-new-feature")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, tkt.ID, result.ID)
	})

	t.Run("finds ticket by exact identifier branch", func(t *testing.T) {
		result, err := service.FindTicketByBranch(ctx, "PRJ-123")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, tkt.ID, result.ID)
	})

	t.Run("returns nil for branch without identifier", func(t *testing.T) {
		result, err := service.FindTicketByBranch(ctx, "feature/some-branch")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns nil for non-existent ticket", func(t *testing.T) {
		result, err := service.FindTicketByBranch(ctx, "feature/NONEXISTENT-999")
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestSyncMRByURL(t *testing.T) {
	ctx := context.Background()
	db := setupMRSyncTestDB(t)

	t.Run("returns error for non-existent MR", func(t *testing.T) {
		provider := &MockGitProvider{}
		service := NewMRSyncService(db, provider)

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
		service := NewMRSyncService(db, provider)

		// Create repository
		repoID := int64(10)
		repo := &repository.Repository{
			OrganizationID: 1,
			Name:           "repo",
			FullPath:       "org/repo",
			ExternalID:     "123",
		}
		db.Create(repo)
		repoID = repo.ID

		// Create ticket with repository
		tkt := &ticket.Ticket{
			OrganizationID: 1,
			Identifier:     "SYNC-1",
			Title:          "Test Ticket",
			Type:           ticket.TicketTypeTask,
			Status:         ticket.TicketStatusInProgress,
			Priority:       ticket.TicketPriorityMedium,
			RepositoryID:   &repoID,
		}
		db.Create(tkt)

		// Create MR
		mr := &ticket.MergeRequest{
			OrganizationID: 1,
			TicketID:       tkt.ID,
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
		service := NewMRSyncService(db, provider)

		// Create ticket without repository
		tkt := &ticket.Ticket{
			OrganizationID: 1,
			Identifier:     "NOREPO-1",
			Title:          "Test Ticket",
			Type:           ticket.TicketTypeTask,
			Status:         ticket.TicketStatusInProgress,
			Priority:       ticket.TicketPriorityMedium,
		}
		db.Create(tkt)

		// Create MR
		mr := &ticket.MergeRequest{
			OrganizationID: 1,
			TicketID:       tkt.ID,
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

func TestBuildMRData(t *testing.T) {
	db := setupMRSyncTestDB(t)
	service := NewMRSyncService(db, nil)

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
	service := NewMRSyncService(db, nil)

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

		service := NewMRSyncService(db, provider)

		// Create repo
		repo := &repository.Repository{
			OrganizationID: 1,
			Name:           "test-repo",
			FullPath:       "org/test-repo",
			ExternalID:     "456",
		}
		db.Create(repo)

		// Create ticket
		tkt := &ticket.Ticket{
			OrganizationID: 1,
			Identifier:     "INT-1",
			Title:          "Integration Test Ticket",
			Type:           ticket.TicketTypeTask,
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
		foundTicket, err := service.FindTicketByBranch(ctx, "feature/INT-1")
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

		service := NewMRSyncService(db, provider)

		// Create repo
		repo := &repository.Repository{
			OrganizationID: 1,
			Name:           "error-repo",
			FullPath:       "org/error-repo",
			ExternalID:     "789",
		}
		db.Create(repo)

		// Create ticket
		tkt := &ticket.Ticket{
			OrganizationID: 1,
			Identifier:     "ERR-1",
			Title:          "Error Test Ticket",
			Type:           ticket.TicketTypeTask,
			Status:         ticket.TicketStatusInProgress,
			Priority:       ticket.TicketPriorityMedium,
			RepositoryID:   &repo.ID,
		}
		db.Create(tkt)

		// Create MR
		mr := &ticket.MergeRequest{
			OrganizationID: 1,
			TicketID:       tkt.ID,
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
