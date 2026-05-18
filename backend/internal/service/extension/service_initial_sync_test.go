package extension

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// TestCreateSkillRegistry_TriggersInitialSync is the regression test for the
// other half of issue #375: after the row is inserted, the marketplace stays
// empty until SyncSource runs. We verify the goroutine is launched, reaches
// the importer, and runs to completion (success or failure).
func TestCreateSkillRegistry_TriggersInitialSync(t *testing.T) {
	syncStarted := make(chan struct{}, 1)
	syncDone := make(chan struct{}, 1)
	var createdID int64

	repo := &svcMockRepo{
		createSkillRegistryFn: func(_ context.Context, source *extension.SkillRegistry) error {
			source.ID = 7
			atomic.StoreInt64(&createdID, source.ID)
			return nil
		},
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{ID: id, RepositoryURL: "https://github.com/org/repo", Branch: "main"}, nil
		},
		updateSkillRegistryFn: func(_ context.Context, source *extension.SkillRegistry) error {
			// Terminal write — sync either succeeded or failed; unblock the test.
			if source.SyncStatus == extension.SyncStatusSuccess || source.SyncStatus == extension.SyncStatusFailed {
				select {
				case syncDone <- struct{}{}:
				default:
				}
			}
			return nil
		},
	}
	stor := &svcMockStorage{}
	svc := newTestService(repo, stor, nil)

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = func(_ context.Context, _, _, targetDir string) error {
		select {
		case syncStarted <- struct{}{}:
		default:
		}
		return os.MkdirAll(targetDir, 0755)
	}
	imp.gitHeadFn = func(_ context.Context, _ string) (string, error) {
		return "fakesha", nil
	}
	svc.SetSkillImporter(imp)

	_, err := svc.CreateSkillRegistry(context.Background(), 1, CreateSkillRegistryInput{
		RepositoryURL: "https://github.com/org/repo",
	})
	if err != nil {
		t.Fatalf("CreateSkillRegistry returned error: %v", err)
	}

	select {
	case <-syncStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("background sync was never triggered after CreateSkillRegistry — #375 regression")
	}

	select {
	case <-syncDone:
	case <-time.After(5 * time.Second):
		t.Fatal("background sync started but never completed — goroutine may be leaking work into adjacent tests")
	}

	if atomic.LoadInt64(&createdID) != 7 {
		t.Errorf("expected created registry id=7, got %d", createdID)
	}
}

// TestCreateSkillRegistry_NoImporter_DoesNotPanic guards the nil-importer
// branch — Service can run without an importer wired up (early boot, tests).
func TestCreateSkillRegistry_NoImporter_DoesNotPanic(t *testing.T) {
	repo := &svcMockRepo{
		createSkillRegistryFn: func(_ context.Context, source *extension.SkillRegistry) error {
			source.ID = 1
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.CreateSkillRegistry(context.Background(), 1, CreateSkillRegistryInput{
		RepositoryURL: "https://github.com/org/repo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTriggerInitialSync_SwallowsSyncError verifies that SyncSource errors are
// logged but never propagated, and that triggerInitialSync returns quickly
// rather than hanging when the importer fails.
func TestTriggerInitialSync_SwallowsSyncError(t *testing.T) {
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, _ int64) (*extension.SkillRegistry, error) {
			return nil, errors.New("registry vanished")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)
	svc.SetSkillImporter(NewSkillImporter(repo, &svcMockStorage{}))

	done := make(chan struct{})
	go func() {
		svc.triggerInitialSync(123)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("triggerInitialSync did not return on importer error")
	}
}
