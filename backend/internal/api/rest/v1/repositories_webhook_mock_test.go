package v1

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
)

// ===========================================
// Mock Services for Webhook Tests
// ===========================================

// mockRepositoryService implements repository.RepositoryServiceInterface
type mockRepositoryService struct {
	repos          map[int64]*gitprovider.Repository
	webhookService *mockWebhookService
	getByIDError   error
}

func newMockRepositoryService() *mockRepositoryService {
	return &mockRepositoryService{
		repos:          make(map[int64]*gitprovider.Repository),
		webhookService: newMockWebhookService(),
	}
}

func (m *mockRepositoryService) GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error) {
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	if repo, ok := m.repos[id]; ok {
		return repo, nil
	}
	return nil, repository.ErrRepositoryNotFound
}

func (m *mockRepositoryService) GetByIDForUser(ctx context.Context, id int64, userID int64) (*gitprovider.Repository, error) {
	return m.GetByID(ctx, id)
}

func (m *mockRepositoryService) Create(ctx context.Context, req *repository.CreateRequest) (*gitprovider.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryService) CreateWithWebhook(ctx context.Context, req *repository.CreateRequest, orgSlug string) (*gitprovider.Repository, *repository.WebhookResult, error) {
	return nil, nil, nil
}

func (m *mockRepositoryService) Update(ctx context.Context, id int64, updates map[string]interface{}) (*gitprovider.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryService) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m *mockRepositoryService) ListByOrganization(ctx context.Context, orgID int64) ([]*gitprovider.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryService) ListByOrganizationForUser(ctx context.Context, orgID int64, userID int64) ([]*gitprovider.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryService) GetWebhookService() repository.WebhookServiceInterface {
	if m.webhookService == nil {
		return nil
	}
	return m.webhookService
}

func (m *mockRepositoryService) ListBranches(ctx context.Context, repoID int64, accessToken string) ([]string, error) {
	return nil, nil
}

func (m *mockRepositoryService) SyncFromProvider(ctx context.Context, repoID int64, accessToken string) (*gitprovider.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryService) GetBySlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*gitprovider.Repository, error) {
	for _, repo := range m.repos {
		if repo.OrganizationID == orgID && repo.ProviderType == providerType && repo.ProviderBaseURL == providerBaseURL && repo.Slug == slug {
			return repo, nil
		}
	}
	return nil, repository.ErrRepositoryNotFound
}

func (m *mockRepositoryService) ListMergeRequests(ctx context.Context, repoID int64, branch, state string) ([]*repository.MergeRequestInfo, error) {
	return nil, nil
}

func (m *mockRepositoryService) AddRepo(repo *gitprovider.Repository) {
	m.repos[repo.ID] = repo
}

// mockWebhookService implements repository.WebhookServiceInterface
type mockWebhookService struct {
	registerResult     *repository.WebhookResult
	registerError      error
	deleteError        error
	webhookStatus      *gitprovider.WebhookStatus
	webhookSecret      string
	secretError        error
	markConfiguredErr  error
	verifySecretResult bool
	verifySecretError  error
	getRepoError       error
	repoForWebhook     *gitprovider.Repository
}

func newMockWebhookService() *mockWebhookService {
	return &mockWebhookService{
		verifySecretResult: true,
	}
}

func (m *mockWebhookService) RegisterWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, orgSlug string, userID int64) (*repository.WebhookResult, error) {
	if m.registerError != nil {
		return nil, m.registerError
	}
	if m.registerResult != nil {
		return m.registerResult, nil
	}
	return &repository.WebhookResult{
		RepoID:     repo.ID,
		Registered: true,
		WebhookID:  "wh_test123",
	}, nil
}

func (m *mockWebhookService) DeleteWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, userID int64) error {
	return m.deleteError
}

func (m *mockWebhookService) GetWebhookStatus(ctx context.Context, repo *gitprovider.Repository) *gitprovider.WebhookStatus {
	if m.webhookStatus != nil {
		return m.webhookStatus
	}
	return &gitprovider.WebhookStatus{Registered: false}
}

func (m *mockWebhookService) GetWebhookSecret(ctx context.Context, repo *gitprovider.Repository) (string, error) {
	if m.secretError != nil {
		return "", m.secretError
	}
	return m.webhookSecret, nil
}

func (m *mockWebhookService) MarkWebhookAsConfigured(ctx context.Context, repo *gitprovider.Repository) error {
	return m.markConfiguredErr
}

func (m *mockWebhookService) VerifyWebhookSecret(ctx context.Context, repoID int64, secret string) (bool, error) {
	if m.verifySecretError != nil {
		return false, m.verifySecretError
	}
	return m.verifySecretResult, nil
}

func (m *mockWebhookService) GetRepositoryByIDWithWebhook(ctx context.Context, repoID int64) (*gitprovider.Repository, error) {
	if m.getRepoError != nil {
		return nil, m.getRepoError
	}
	return m.repoForWebhook, nil
}
