package extension

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// syncAll tests (DB-driven)
// ---------------------------------------------------------------------------

func TestSyncAll_QueriesDBForPlatformSources(t *testing.T) {
	repo := newMockExtensionRepo()

	var mu sync.Mutex
	syncedIDs := []int64{}

	repo.listSkillRegistriesFunc = func(_ context.Context, orgID *int64) ([]*extension.SkillRegistry, error) {
		if orgID != nil {
			t.Errorf("expected nil orgID for platform-level query, got %v", *orgID)
		}
		return []*extension.SkillRegistry{
			{ID: 1, RepositoryURL: "https://github.com/org/repo1", Branch: "main", IsActive: true},
			{ID: 2, RepositoryURL: "https://github.com/org/repo2", Branch: "main", IsActive: true},
			{ID: 3, RepositoryURL: "https://github.com/org/repo3", Branch: "main", IsActive: false},
		}, nil
	}

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
	w.syncAll(context.Background())
}

func TestSyncAll_DBQueryError(t *testing.T) {
	repo := newMockExtensionRepo()

	repo.listSkillRegistriesFunc = func(_ context.Context, _ *int64) ([]*extension.SkillRegistry, error) {
		return nil, errors.New("db connection error")
	}

	w := newTestWorker(repo)
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

		if current == 1 {
			cancel()
		}
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://github.com/org/repo", Branch: "main"}, nil
	}

	w := newTestWorker(repo)
	w.syncAll(ctx)

	mu.Lock()
	defer mu.Unlock()

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
			OrganizationID: nil,
			RepositoryURL:  "https://github.com/org/skills",
			Branch:         "main",
		}, nil
	}

	w := newTestWorker(repo)

	err := w.SyncSingle(context.Background(), 1)
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
			OrganizationID: &orgID,
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
	w.syncInterval = time.Hour

	ctx := context.Background()
	w.Start(ctx)

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
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
	w.syncInterval = time.Hour

	ctx := context.Background()
	w.Start(ctx)

	time.Sleep(11 * time.Second)

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds after initial sync")
	}
}

func TestStop_WithoutStart(t *testing.T) {
	repo := newMockExtensionRepo()
	w := newTestWorker(repo)
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()
	select {
	case <-done:
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

	w.Start(ctx)
	w.Start(ctx)

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds after calling Start() twice; possible goroutine leak")
	}
}
