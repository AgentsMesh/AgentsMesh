package extension

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// TestSyncSource_RejectsConcurrentSyncing verifies that a fresh peer lock
// causes the second caller to bail with ErrSyncInProgress.
func TestSyncSource_RejectsConcurrentSyncing(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.claimSyncLockFunc = func(_ context.Context, _ int64, _ time.Duration) (bool, bool, error) {
		return false, false, nil
	}

	imp := NewSkillImporter(repo, nil)
	err := imp.SyncSource(context.Background(), 1)

	if !errors.Is(err, ErrSyncInProgress) {
		t.Errorf("expected ErrSyncInProgress, got: %v", err)
	}
}

// TestSyncSource_RecoversStaleSyncing is the regression test for risk #1:
// a prior sync that died mid-run leaves sync_status="syncing" wedged. The
// repository's ClaimSyncLock signals wasStale=true; SyncSource proceeds past
// it and continues to doSync.
func TestSyncSource_RecoversStaleSyncing(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.claimSyncLockFunc = func(_ context.Context, _ int64, _ time.Duration) (bool, bool, error) {
		return true, true, nil // reclaimed
	}
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{
			ID:            id,
			RepositoryURL: "https://example.com/repo",
			Branch:        "main",
		}, nil
	}

	updateCalled := false
	repo.updateSourceFunc = func(_ context.Context, _ *extension.SkillRegistry) error {
		updateCalled = true
		return nil
	}

	imp := NewSkillImporter(repo, nil)
	// gitCloneFn is nil → doSync fails, but importantly SyncSource must reach
	// the post-doSync UpdateSkillRegistry to record the failure status.
	_ = imp.SyncSource(context.Background(), 1)

	if !updateCalled {
		t.Fatal("stale lock recovery did not flow through to post-sync UpdateSkillRegistry")
	}
}

// TestSyncSource_ClaimSyncLockError surfaces DB errors from the lock step
// rather than swallowing them as ErrSyncInProgress.
func TestSyncSource_ClaimSyncLockError(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.claimSyncLockFunc = func(_ context.Context, _ int64, _ time.Duration) (bool, bool, error) {
		return false, false, errors.New("db down")
	}

	imp := NewSkillImporter(repo, nil)
	err := imp.SyncSource(context.Background(), 1)

	if err == nil {
		t.Fatal("expected error when ClaimSyncLock fails")
	}
	if errors.Is(err, ErrSyncInProgress) {
		t.Errorf("DB error should not be reported as ErrSyncInProgress: %v", err)
	}
}
