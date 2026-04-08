package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
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
	return testkit.SetupTestDB(t)
}
