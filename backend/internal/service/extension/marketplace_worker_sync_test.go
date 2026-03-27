package extension

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

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

	registry := &extension.SkillRegistry{
		ID:            42,
		RepositoryURL: "https://github.com/org/skills",
		Branch:        "main",
		IsActive:      true,
	}

	w.syncRegistry(context.Background(), registry)
}

// ---------------------------------------------------------------------------
// syncAll: MCP Registry syncer branch coverage
// ---------------------------------------------------------------------------

func TestSyncAll_WithRegistrySyncer_Success(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, nil
	}

	ts := newTestRegistryServer(`{"servers":[],"metadata":{"nextCursor":"","count":0}}`)
	defer ts.Close()

	client := NewMcpRegistryClient(ts.URL)
	syncer := NewMcpRegistrySyncer(client, repo)

	w := &MarketplaceWorker{
		importer:       NewSkillImporter(repo, nil),
		registrySyncer: syncer,
		repo:           repo,
		syncInterval:   time.Hour,
	}

	w.syncAll(context.Background())
}

func TestSyncAll_WithRegistrySyncer_Error(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, nil
	}

	client := NewMcpRegistryClient("http://127.0.0.1:1")
	syncer := NewMcpRegistrySyncer(client, repo)

	w := &MarketplaceWorker{
		importer:       NewSkillImporter(repo, nil),
		registrySyncer: syncer,
		repo:           repo,
		syncInterval:   time.Hour,
	}

	w.syncAll(context.Background())
}

func TestSyncAll_WithRegistrySyncer_ContextCancelled(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return []*extension.SkillRegistry{
			{ID: 1, RepositoryURL: "https://github.com/org/repo", Branch: "main", IsActive: true},
		}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		cancel()
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
			OrganizationID: nil,
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
