package extension

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// TestSyncAll_ConcurrencyBounded verifies syncAll respects the configured
// concurrency cap: even when many registries are ready, no more than that
// many syncs run at the same instant. Without bounding, large org deployments
// would burn through network and disk in a thundering herd.
func TestSyncAll_ConcurrencyBounded(t *testing.T) {
	repo := newMockExtensionRepo()

	const totalRegistries = defaultSyncConcurrency * 3
	registries := make([]*extension.SkillRegistry, totalRegistries)
	for i := range registries {
		registries[i] = &extension.SkillRegistry{
			ID:            int64(i + 1),
			RepositoryURL: "https://example.com/repo",
			Branch:        "main",
			IsActive:      true,
		}
	}
	repo.listAllActiveRegistriesFunc = func(_ context.Context) ([]*extension.SkillRegistry, error) {
		return registries, nil
	}

	var (
		inFlight    int32
		maxInFlight int32
		release     = make(chan struct{})
	)

	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		current := atomic.AddInt32(&inFlight, 1)
		for {
			prev := atomic.LoadInt32(&maxInFlight)
			if current <= prev || atomic.CompareAndSwapInt32(&maxInFlight, prev, current) {
				break
			}
		}
		<-release // hold the goroutine so concurrency stays observable
		atomic.AddInt32(&inFlight, -1)
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://example.com/repo", Branch: "main"}, nil
	}

	done := make(chan struct{})
	go func() {
		newTestWorker(repo).syncAll(context.Background())
		close(done)
	}()

	// Release only after the cap is hit, so maxInFlight reflects the limit
	// rather than a transient mid-ramp value.
	waitForInFlight(t, &inFlight, defaultSyncConcurrency, 3*time.Second)
	close(release)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("syncAll did not finish after releasing goroutines")
	}

	got := atomic.LoadInt32(&maxInFlight)
	if got > defaultSyncConcurrency {
		t.Errorf("max in-flight syncs = %d, exceeds limit %d", got, defaultSyncConcurrency)
	}
	if got == 0 {
		t.Fatal("no syncs ran at all — worker is wired wrong")
	}
}

func waitForInFlight(t *testing.T, counter *int32, target int32, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(counter) >= target {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("never reached %d in-flight goroutines (got %d)", target, atomic.LoadInt32(counter))
}
