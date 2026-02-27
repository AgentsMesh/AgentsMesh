package extension

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// mockExtensionRepo — shared mock of extension.Repository.
// This type is also embedded by packagerMockRepo in skill_packager_test.go.
// ---------------------------------------------------------------------------

type mockExtensionRepo struct {
	mu sync.Mutex

	// Configurable function hooks (used by marketplace worker tests).
	findSourceFunc        func(ctx context.Context, orgID *int64, repoURL string) (*extension.SkillRegistry, error)
	createSourceFunc      func(ctx context.Context, source *extension.SkillRegistry) error
	getSourceFunc         func(ctx context.Context, id int64) (*extension.SkillRegistry, error)
	updateSourceFunc      func(ctx context.Context, source *extension.SkillRegistry) error
	listSkillRegistriesFunc  func(ctx context.Context, orgID *int64) ([]*extension.SkillRegistry, error)

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
// helper: build a MarketplaceWorker with a mock repo (no real storage)
// ---------------------------------------------------------------------------

func newTestWorker(repo *mockExtensionRepo) *MarketplaceWorker {
	return &MarketplaceWorker{
		importer:     NewSkillImporter(repo, nil), // storage=nil; SyncSource will fail on git clone
		repo:         repo,
		syncInterval: time.Hour, // irrelevant for unit tests
	}
}

// ---------------------------------------------------------------------------
// syncAll tests (DB-driven)
// ---------------------------------------------------------------------------

func TestSyncAll_QueriesDBForPlatformSources(t *testing.T) {
	repo := newMockExtensionRepo()

	var mu sync.Mutex
	syncedIDs := []int64{}

	// ListSkillRegistries returns platform-level sources from DB
	repo.listSkillRegistriesFunc = func(_ context.Context, orgID *int64) ([]*extension.SkillRegistry, error) {
		if orgID != nil {
			t.Errorf("expected nil orgID for platform-level query, got %v", *orgID)
		}
		return []*extension.SkillRegistry{
			{ID: 1, RepositoryURL: "https://github.com/org/repo1", Branch: "main", IsActive: true},
			{ID: 2, RepositoryURL: "https://github.com/org/repo2", Branch: "main", IsActive: true},
			{ID: 3, RepositoryURL: "https://github.com/org/repo3", Branch: "main", IsActive: false}, // inactive
		}, nil
	}

	// Track which source IDs get synced (via GetSkillRegistry in SyncSource)
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		mu.Lock()
		syncedIDs = append(syncedIDs, id)
		mu.Unlock()
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://github.com/org/repo", Branch: "main"}, nil
	}

	w := newTestWorker(repo)
	w.syncAll(context.Background())

	mu.Lock()
	defer mu.Unlock()

	// Should have synced source 1 and 2 (not 3 because it's inactive)
	if len(syncedIDs) != 2 {
		t.Fatalf("expected 2 synced sources, got %d: %v", len(syncedIDs), syncedIDs)
	}
	if syncedIDs[0] != 1 || syncedIDs[1] != 2 {
		t.Errorf("expected synced IDs [1, 2], got %v", syncedIDs)
	}
}

func TestSyncAll_EmptyDBSources(t *testing.T) {
	repo := newMockExtensionRepo()

	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, nil
	}

	w := newTestWorker(repo)
	// Should not panic
	w.syncAll(context.Background())
}

func TestSyncAll_DBQueryError(t *testing.T) {
	repo := newMockExtensionRepo()

	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, errors.New("db connection error")
	}

	w := newTestWorker(repo)
	// Should not panic, should log error
	w.syncAll(context.Background())
}

func TestSyncAll_ContextCancelledStopsEarly(t *testing.T) {
	repo := newMockExtensionRepo()

	var mu sync.Mutex
	callCount := 0

	ctx, cancel := context.WithCancel(context.Background())

	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return []*extension.SkillRegistry{
			{ID: 1, RepositoryURL: "https://github.com/org/repo1", Branch: "main", IsActive: true},
			{ID: 2, RepositoryURL: "https://github.com/org/repo2", Branch: "main", IsActive: true},
			{ID: 3, RepositoryURL: "https://github.com/org/repo3", Branch: "main", IsActive: true},
		}, nil
	}

	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		mu.Lock()
		callCount++
		current := callCount
		mu.Unlock()

		// Cancel the context after processing the first source
		if current == 1 {
			cancel()
		}
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://github.com/org/repo", Branch: "main"}, nil
	}

	w := newTestWorker(repo)
	w.syncAll(ctx)

	mu.Lock()
	defer mu.Unlock()

	// The first source is processed. After that, ctx.Err() != nil, so the loop
	// should stop before processing all three.
	if callCount >= 3 {
		t.Errorf("expected early stop, but all %d sources were processed", callCount)
	}
}

// ---------------------------------------------------------------------------
// SyncSingle tests
// ---------------------------------------------------------------------------

func TestSyncSingle_PlatformLevel(t *testing.T) {
	repo := newMockExtensionRepo()

	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{
			ID:             id,
			OrganizationID: nil, // platform-level
			RepositoryURL:  "https://github.com/org/skills",
			Branch:         "main",
		}, nil
	}

	w := newTestWorker(repo)

	// SyncSingle should attempt sync (which will fail at git clone since no real storage)
	// but should not return "not a platform-level registry" error
	err := w.SyncSingle(context.Background(), 1)
	// We expect a git clone error, not a platform-level error
	if err != nil && strings.Contains(err.Error(), "not a platform-level registry") {
		t.Errorf("unexpected platform-level error: %v", err)
	}
}

func TestSyncSingle_NonPlatformLevel(t *testing.T) {
	repo := newMockExtensionRepo()

	orgID := int64(42)
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{
			ID:             id,
			OrganizationID: &orgID, // org-level
			RepositoryURL:  "https://github.com/org/skills",
			Branch:         "main",
		}, nil
	}

	w := newTestWorker(repo)

	err := w.SyncSingle(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for non-platform-level registry")
	}
	if !strings.Contains(err.Error(), "not a platform-level registry") {
		t.Errorf("expected platform-level error, got: %v", err)
	}
}

func TestSyncSingle_SourceNotFound(t *testing.T) {
	repo := newMockExtensionRepo()

	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return nil, errors.New("not found")
	}

	w := newTestWorker(repo)

	err := w.SyncSingle(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent source")
	}
}

// ---------------------------------------------------------------------------
// Start / Stop tests
// ---------------------------------------------------------------------------

func TestMarketplaceWorker_GracefulShutdown(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return []*extension.SkillRegistry{
			{ID: 1, RepositoryURL: "https://github.com/org/repo", Branch: "main", IsActive: true},
		}, nil
	}
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{ID: id, Branch: "main"}, nil
	}

	w := newTestWorker(repo)
	// Use a very long interval so the periodic ticker does not fire during the test.
	w.syncInterval = time.Hour

	ctx := context.Background()
	w.Start(ctx)

	// Immediately stop -- should not deadlock.
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK: graceful shutdown completed.
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds; possible deadlock")
	}
}

// ---------------------------------------------------------------------------
// NewMarketplaceWorker tests
// ---------------------------------------------------------------------------

func TestNewMarketplaceWorker(t *testing.T) {
	repo := newMockExtensionRepo()
	imp := NewSkillImporter(repo, nil)
	w := NewMarketplaceWorker(repo, imp, nil, time.Hour)
	if w == nil {
		t.Fatal("expected non-nil worker")
	}
	if w.syncInterval != time.Hour {
		t.Errorf("expected sync interval 1h, got %v", w.syncInterval)
	}
	if w.importer == nil {
		t.Error("expected non-nil importer")
	}
	if w.repo != repo {
		t.Error("expected repo to be set")
	}
}

func TestNewMarketplaceWorker_CustomInterval(t *testing.T) {
	repo := newMockExtensionRepo()
	imp := NewSkillImporter(repo, nil)
	w := NewMarketplaceWorker(repo, imp, nil, 30*time.Minute)
	if w.syncInterval != 30*time.Minute {
		t.Errorf("expected sync interval 30m, got %v", w.syncInterval)
	}
}

func TestMarketplaceWorker_Start_TimerFires(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires 11-second wait")
	}

	repo := newMockExtensionRepo()
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, nil
	}

	w := newTestWorker(repo)
	// Use a long sync interval so only the initial sync fires
	w.syncInterval = time.Hour

	ctx := context.Background()
	w.Start(ctx)

	// Wait for the 10-second initial timer to fire and syncAll to run
	time.Sleep(11 * time.Second)

	// Stop after initial sync
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()
	select {
	case <-done:
		// OK: graceful shutdown after initial timer
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds after initial sync")
	}
}

func TestStop_WithoutStart(t *testing.T) {
	repo := newMockExtensionRepo()
	w := newTestWorker(repo)
	// Stop without Start should not panic
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()
	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() without Start() did not return within 2 seconds")
	}
}

func TestStart_CalledTwice_NoLeak(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, nil
	}

	w := newTestWorker(repo)
	w.syncInterval = time.Hour

	ctx := context.Background()

	// Start twice — second call should be a no-op thanks to sync.Once
	w.Start(ctx)
	w.Start(ctx)

	// Stop once — should not deadlock or panic
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK: graceful shutdown completed with no goroutine leak.
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds after calling Start() twice; possible goroutine leak")
	}
}

// ---------------------------------------------------------------------------
// syncRegistry: SyncSource succeeds (covers the success branch)
// ---------------------------------------------------------------------------

func TestSyncRegistry_SyncSourceSuccess(t *testing.T) {
	repo := newMockExtensionRepo()

	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://github.com/org/skills", Branch: "main"}, nil
	}

	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)
	// Mock git functions to simulate a successful empty-repo sync
	imp.gitCloneFn = func(_ context.Context, _, _, targetDir string) error {
		// Create an empty directory (no SKILL.md => collection with 0 skills => success)
		return os.MkdirAll(targetDir, 0755)
	}
	imp.gitHeadFn = func(_ context.Context, _ string) (string, error) {
		return "abc" + strings.Repeat("0", 37), nil
	}

	w := &MarketplaceWorker{
		importer:     imp,
		repo:         repo,
		syncInterval: time.Hour,
	}

	registry := &extension.SkillRegistry{
		ID:            42,
		RepositoryURL: "https://github.com/org/skills",
		Branch:        "main",
		IsActive:      true,
	}

	// syncRegistry should succeed — covers the SyncSource success branch
	w.syncRegistry(context.Background(), registry)
}

// ---------------------------------------------------------------------------
// syncAll: MCP Registry syncer branch coverage
// ---------------------------------------------------------------------------

func TestSyncAll_WithRegistrySyncer_Success(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, nil // no skill registries, skip to MCP sync
	}

	// Create a real McpRegistryClient pointing to a test server that returns empty list
	ts := newTestRegistryServer(t, `{"servers":[],"metadata":{"nextCursor":"","count":0}}`)
	defer ts.Close()

	client := NewMcpRegistryClient(ts.URL)
	syncer := NewMcpRegistrySyncer(client, repo)

	w := &MarketplaceWorker{
		importer:       NewSkillImporter(repo, nil),
		registrySyncer: syncer,
		repo:           repo,
		syncInterval:   time.Hour,
	}

	// syncAll should succeed and reach the MCP Registry sync branch
	w.syncAll(context.Background())
}

func TestSyncAll_WithRegistrySyncer_Error(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, nil
	}

	// Create a client pointing to an invalid URL that will fail
	client := NewMcpRegistryClient("http://127.0.0.1:1") // connection refused
	syncer := NewMcpRegistrySyncer(client, repo)

	w := &MarketplaceWorker{
		importer:       NewSkillImporter(repo, nil),
		registrySyncer: syncer,
		repo:           repo,
		syncInterval:   time.Hour,
	}

	// Should not panic; error is logged
	w.syncAll(context.Background())
}

func TestSyncAll_WithRegistrySyncer_ContextCancelled(t *testing.T) {
	repo := newMockExtensionRepo()
	// Return one active source that will consume the context
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return []*extension.SkillRegistry{
			{ID: 1, RepositoryURL: "https://github.com/org/repo", Branch: "main", IsActive: true},
		}, nil
	}
	// Cancel context during source sync
	ctx, cancel := context.WithCancel(context.Background())
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		cancel() // cancel after first source processes
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://github.com/org/repo", Branch: "main"}, nil
	}

	client := NewMcpRegistryClient("http://localhost:9999")
	syncer := NewMcpRegistrySyncer(client, repo)

	w := &MarketplaceWorker{
		importer:       NewSkillImporter(repo, nil),
		registrySyncer: syncer,
		repo:           repo,
		syncInterval:   time.Hour,
	}

	// syncAll should stop before MCP Registry sync due to cancelled context
	w.syncAll(ctx)
}

// ---------------------------------------------------------------------------
// SyncSingle: success path
// ---------------------------------------------------------------------------

func TestSyncSingle_SuccessPath(t *testing.T) {
	repo := newMockExtensionRepo()

	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{
			ID:             id,
			OrganizationID: nil, // platform-level
			RepositoryURL:  "https://github.com/org/skills",
			Branch:         "main",
		}, nil
	}

	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = func(_ context.Context, _, _, targetDir string) error {
		return os.MkdirAll(targetDir, 0755)
	}
	imp.gitHeadFn = func(_ context.Context, _ string) (string, error) {
		return "abc" + strings.Repeat("0", 37), nil
	}

	w := &MarketplaceWorker{
		importer:     imp,
		repo:         repo,
		syncInterval: time.Hour,
	}

	err := w.SyncSingle(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error for successful sync, got: %v", err)
	}
}

// newTestRegistryServer creates a test HTTP server that returns the given JSON body.
func newTestRegistryServer(t *testing.T, body string) *httpTestServer {
	t.Helper()
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
