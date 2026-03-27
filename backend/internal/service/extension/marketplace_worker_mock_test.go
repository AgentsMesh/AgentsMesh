package extension

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// mockExtensionRepo -- shared mock of extension.Repository.
// This type is also embedded by packagerMockRepo in skill_packager_test.go.
// ---------------------------------------------------------------------------

type mockExtensionRepo struct {
	mu sync.Mutex

	// Configurable function hooks (used by marketplace worker tests).
	findSourceFunc          func(ctx context.Context, orgID *int64, repoURL string) (*extension.SkillRegistry, error)
	createSourceFunc        func(ctx context.Context, source *extension.SkillRegistry) error
	getSourceFunc           func(ctx context.Context, id int64) (*extension.SkillRegistry, error)
	updateSourceFunc        func(ctx context.Context, source *extension.SkillRegistry) error
	listSkillRegistriesFunc func(ctx context.Context, orgID *int64) ([]*extension.SkillRegistry, error)

	// Track what was created (used for assertions).
	createdSources []*extension.SkillRegistry
}

func newMockExtensionRepo() *mockExtensionRepo {
	return &mockExtensionRepo{}
}

// --- Skill Registries ---

func (m *mockExtensionRepo) FindSkillRegistryByURL(ctx context.Context, orgID *int64, repoURL string) (*extension.SkillRegistry, error) {
	if m.findSourceFunc != nil {
		return m.findSourceFunc(ctx, orgID, repoURL)
	}
	return nil, errors.New("not found")
}

func (m *mockExtensionRepo) CreateSkillRegistry(ctx context.Context, source *extension.SkillRegistry) error {
	if m.createSourceFunc != nil {
		return m.createSourceFunc(ctx, source)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	source.ID = int64(len(m.createdSources) + 1)
	m.createdSources = append(m.createdSources, source)
	return nil
}

func (m *mockExtensionRepo) GetSkillRegistry(ctx context.Context, id int64) (*extension.SkillRegistry, error) {
	if m.getSourceFunc != nil {
		return m.getSourceFunc(ctx, id)
	}
	return &extension.SkillRegistry{ID: id, RepositoryURL: "https://example.com/repo", Branch: "main"}, nil
}

func (m *mockExtensionRepo) UpdateSkillRegistry(ctx context.Context, source *extension.SkillRegistry) error {
	if m.updateSourceFunc != nil {
		return m.updateSourceFunc(ctx, source)
	}
	return nil
}

func (m *mockExtensionRepo) ListSkillRegistries(ctx context.Context, orgID *int64) ([]*extension.SkillRegistry, error) {
	if m.listSkillRegistriesFunc != nil {
		return m.listSkillRegistriesFunc(ctx, orgID)
	}
	return nil, nil
}

func (m *mockExtensionRepo) DeleteSkillRegistry(_ context.Context, _ int64) error {
	return nil
}

// --- Skill Market Items ---

func (m *mockExtensionRepo) ListSkillMarketItems(_ context.Context, _ *int64, _ string, _ string) ([]*extension.SkillMarketItem, error) {
	return nil, nil
}

func (m *mockExtensionRepo) GetSkillMarketItem(_ context.Context, _ int64) (*extension.SkillMarketItem, error) {
	return nil, nil
}

func (m *mockExtensionRepo) FindSkillMarketItemBySlug(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
	return nil, nil
}

func (m *mockExtensionRepo) CreateSkillMarketItem(_ context.Context, _ *extension.SkillMarketItem) error {
	return nil
}

func (m *mockExtensionRepo) UpdateSkillMarketItem(_ context.Context, _ *extension.SkillMarketItem) error {
	return nil
}

func (m *mockExtensionRepo) DeactivateSkillMarketItemsNotIn(_ context.Context, _ int64, _ []string) error {
	return nil
}

// --- MCP Market Items ---

func (m *mockExtensionRepo) ListMcpMarketItems(_ context.Context, _ string, _ string, _, _ int) ([]*extension.McpMarketItem, int64, error) {
	return nil, 0, nil
}

func (m *mockExtensionRepo) GetMcpMarketItem(_ context.Context, _ int64) (*extension.McpMarketItem, error) {
	return nil, nil
}

func (m *mockExtensionRepo) FindMcpMarketItemByRegistryName(_ context.Context, _ string) (*extension.McpMarketItem, error) {
	return nil, errors.New("not found")
}

func (m *mockExtensionRepo) UpsertMcpMarketItem(_ context.Context, _ *extension.McpMarketItem) error {
	return nil
}

func (m *mockExtensionRepo) BatchUpsertMcpMarketItems(_ context.Context, _ []*extension.McpMarketItem) error {
	return nil
}

func (m *mockExtensionRepo) DeactivateMcpMarketItemsNotIn(_ context.Context, _ string, _ []string) (int64, error) {
	return 0, nil
}

// --- Installed MCP Servers ---

func (m *mockExtensionRepo) ListInstalledMcpServers(_ context.Context, _, _, _ int64, _ string) ([]*extension.InstalledMcpServer, error) {
	return nil, nil
}

func (m *mockExtensionRepo) GetInstalledMcpServer(_ context.Context, _ int64) (*extension.InstalledMcpServer, error) {
	return nil, nil
}

func (m *mockExtensionRepo) CreateInstalledMcpServer(_ context.Context, _ *extension.InstalledMcpServer) error {
	return nil
}

func (m *mockExtensionRepo) UpdateInstalledMcpServer(_ context.Context, _ *extension.InstalledMcpServer) error {
	return nil
}

func (m *mockExtensionRepo) DeleteInstalledMcpServer(_ context.Context, _ int64) error {
	return nil
}

func (m *mockExtensionRepo) GetEffectiveMcpServers(_ context.Context, _, _, _ int64) ([]*extension.InstalledMcpServer, error) {
	return nil, nil
}

// --- Installed Skills ---

func (m *mockExtensionRepo) ListInstalledSkills(_ context.Context, _, _, _ int64, _ string) ([]*extension.InstalledSkill, error) {
	return nil, nil
}

func (m *mockExtensionRepo) GetInstalledSkill(_ context.Context, _ int64) (*extension.InstalledSkill, error) {
	return nil, nil
}

func (m *mockExtensionRepo) CreateInstalledSkill(_ context.Context, _ *extension.InstalledSkill) error {
	return nil
}

func (m *mockExtensionRepo) UpdateInstalledSkill(_ context.Context, _ *extension.InstalledSkill) error {
	return nil
}

func (m *mockExtensionRepo) DeleteInstalledSkill(_ context.Context, _ int64) error {
	return nil
}

func (m *mockExtensionRepo) GetEffectiveSkills(_ context.Context, _, _, _ int64) ([]*extension.InstalledSkill, error) {
	return nil, nil
}

func (m *mockExtensionRepo) SetSkillRegistryOverride(_ context.Context, _ int64, _ int64, _ bool) error {
	return nil
}

func (m *mockExtensionRepo) ListSkillRegistryOverrides(_ context.Context, _ int64) ([]*extension.SkillRegistryOverride, error) {
	return nil, nil
}

// Compile-time check that mockExtensionRepo satisfies extension.Repository.
var _ extension.Repository = (*mockExtensionRepo)(nil)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestWorker(repo *mockExtensionRepo) *MarketplaceWorker {
	return &MarketplaceWorker{
		importer:     NewSkillImporter(repo, nil),
		repo:         repo,
		syncInterval: time.Hour,
	}
}

// newTestRegistryServer creates a test HTTP server that returns the given JSON body.
func newTestRegistryServer(body string) *httpTestServer {
	return newHTTPTestServer(body)
}

type httpTestServer struct {
	*httptest.Server
}

func newHTTPTestServer(body string) *httpTestServer {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	return &httpTestServer{srv}
}
