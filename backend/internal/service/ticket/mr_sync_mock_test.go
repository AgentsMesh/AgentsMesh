package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestMRSyncService creates a MRSyncService backed by an in-memory DB for testing.
func newTestMRSyncService(db *gorm.DB, gitProvider git.Provider) *MRSyncService {
	return NewMRSyncService(infra.NewMRSyncRepository(db), gitProvider)
}

// MockGitProvider implements git.Provider for testing
type MockGitProvider struct {
	ListMRsFunc    func(ctx context.Context, projectID, sourceBranch, state string) ([]*git.MergeRequest, error)
	GetMRFunc      func(ctx context.Context, projectID string, iid int) (*git.MergeRequest, error)
	CreateMRFunc   func(ctx context.Context, projectID string, req *git.CreateMRRequest) (*git.MergeRequest, error)
	GetProjectFunc func(ctx context.Context, projectID string) (*git.Project, error)
	GetFileFunc    func(ctx context.Context, projectID, branch, path string) ([]byte, error)
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
func (m *MockGitProvider) ListProjects(ctx context.Context, page, perPage int) ([]*git.Project, error) {
	return nil, nil
}
func (m *MockGitProvider) SearchProjects(ctx context.Context, query string, page, perPage int) ([]*git.Project, error) {
	return nil, nil
}
func (m *MockGitProvider) ListBranches(ctx context.Context, projectID string) ([]*git.Branch, error) {
	return nil, nil
}
func (m *MockGitProvider) GetBranch(ctx context.Context, projectID, branchName string) (*git.Branch, error) {
	return nil, nil
}
func (m *MockGitProvider) CreateBranch(ctx context.Context, projectID, branchName, ref string) (*git.Branch, error) {
	return nil, nil
}
func (m *MockGitProvider) DeleteBranch(ctx context.Context, projectID, branchName string) error {
	return nil
}
func (m *MockGitProvider) ListMergeRequests(ctx context.Context, projectID string, state string, page, perPage int) ([]*git.MergeRequest, error) {
	return nil, nil
}
func (m *MockGitProvider) UpdateMergeRequest(ctx context.Context, projectID string, mrIID int, title, description string) (*git.MergeRequest, error) {
	return nil, nil
}
func (m *MockGitProvider) MergeMergeRequest(ctx context.Context, projectID string, mrIID int) (*git.MergeRequest, error) {
	return nil, nil
}
func (m *MockGitProvider) CloseMergeRequest(ctx context.Context, projectID string, mrIID int) (*git.MergeRequest, error) {
	return nil, nil
}
func (m *MockGitProvider) GetCommit(ctx context.Context, projectID, sha string) (*git.Commit, error) {
	return nil, nil
}
func (m *MockGitProvider) ListCommits(ctx context.Context, projectID, branch string, page, perPage int) ([]*git.Commit, error) {
	return nil, nil
}
func (m *MockGitProvider) RegisterWebhook(ctx context.Context, projectID string, config *git.WebhookConfig) (string, error) {
	return "", nil
}
func (m *MockGitProvider) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	return nil
}
func (m *MockGitProvider) TriggerPipeline(ctx context.Context, projectID string, req *git.TriggerPipelineRequest) (*git.Pipeline, error) {
	return nil, nil
}
func (m *MockGitProvider) GetPipeline(ctx context.Context, projectID string, pipelineID int) (*git.Pipeline, error) {
	return nil, nil
}
func (m *MockGitProvider) ListPipelines(ctx context.Context, projectID string, ref, status string, page, perPage int) ([]*git.Pipeline, error) {
	return nil, nil
}
func (m *MockGitProvider) CancelPipeline(ctx context.Context, projectID string, pipelineID int) (*git.Pipeline, error) {
	return nil, nil
}
func (m *MockGitProvider) RetryPipeline(ctx context.Context, projectID string, pipelineID int) (*git.Pipeline, error) {
	return nil, nil
}
func (m *MockGitProvider) GetJob(ctx context.Context, projectID string, jobID int) (*git.Job, error) {
	return nil, nil
}
func (m *MockGitProvider) ListPipelineJobs(ctx context.Context, projectID string, pipelineID int) ([]*git.Job, error) {
	return nil, nil
}
func (m *MockGitProvider) RetryJob(ctx context.Context, projectID string, jobID int) (*git.Job, error) {
	return nil, nil
}
func (m *MockGitProvider) CancelJob(ctx context.Context, projectID string, jobID int) (*git.Job, error) {
	return nil, nil
}
func (m *MockGitProvider) GetJobTrace(ctx context.Context, projectID string, jobID int) (string, error) {
	return "", nil
}
func (m *MockGitProvider) GetJobArtifact(ctx context.Context, projectID string, jobID int, artifactPath string) ([]byte, error) {
	return nil, nil
}
func (m *MockGitProvider) DownloadJobArtifacts(ctx context.Context, projectID string, jobID int) ([]byte, error) {
	return nil, nil
}

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
			slug TEXT NOT NULL,
			title TEXT NOT NULL,
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
			repository_id INTEGER NOT NULL DEFAULT 0,
			ticket_id INTEGER,
			pod_id INTEGER,
			mr_iid INTEGER NOT NULL,
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
			http_clone_url TEXT,
			ssh_clone_url TEXT,
			external_id TEXT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			default_branch TEXT DEFAULT 'main',
			ticket_prefix TEXT,
			visibility TEXT NOT NULL DEFAULT 'organization',
			imported_by_user_id INTEGER,
			preparation_script TEXT,
			preparation_timeout INTEGER DEFAULT 300,
			is_active INTEGER DEFAULT 1,
			webhook_config TEXT,
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
			agent_status TEXT NOT NULL DEFAULT 'idle',
			branch_name TEXT,
			ticket_id INTEGER,
			last_activity DATETIME,
			agent_waiting_since DATETIME,
			finished_at DATETIME,
			alias TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	return db
}
