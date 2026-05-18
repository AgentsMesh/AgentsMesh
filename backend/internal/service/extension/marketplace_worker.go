package extension

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// MarketplaceWorker runs background sync for all active Skill Registries
// (platform-level and org-level) and the official MCP Registry.
type MarketplaceWorker struct {
	importer       SkillSyncer
	registrySyncer *McpRegistrySyncer
	repo           extension.Repository
	syncInterval   time.Duration

	// syncConcurrency caps in-flight per-cycle syncs. Field rather than const
	// so deployments / tests can tune the dispatch fan-out without recompiling.
	syncConcurrency int

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	startOnce sync.Once
}

// defaultSyncConcurrency caps how many registries are cloned/synced in
// parallel per cycle by default. Each sync involves a git clone + storage
// upload, so bounding this keeps wall-clock cycle time sub-linear in
// registry count without flooding the network/disk.
const defaultSyncConcurrency = 4

// NewMarketplaceWorker creates a new MarketplaceWorker.
// Registries are read from the DB on each sync cycle (no static URL list).
// registrySyncer may be nil if MCP Registry sync is disabled.
func NewMarketplaceWorker(
	repo extension.Repository,
	importer SkillSyncer,
	registrySyncer *McpRegistrySyncer,
	syncInterval time.Duration,
) *MarketplaceWorker {
	return &MarketplaceWorker{
		importer:        importer,
		registrySyncer:  registrySyncer,
		repo:            repo,
		syncInterval:    syncInterval,
		syncConcurrency: defaultSyncConcurrency,
	}
}

// SetSyncConcurrency overrides the per-cycle in-flight cap. Non-positive
// values are ignored. Call before Start().
func (w *MarketplaceWorker) SetSyncConcurrency(n int) {
	if n > 0 {
		w.syncConcurrency = n
	}
}

// Start begins the background sync loop.
// It performs an initial sync, then repeats at the configured interval.
// Calling Start multiple times is safe; only the first call has any effect.
func (w *MarketplaceWorker) Start(ctx context.Context) {
	w.startOnce.Do(func() {
		ctx, w.cancel = context.WithCancel(ctx)

		slog.InfoContext(ctx, "MarketplaceWorker starting",
			"interval", w.syncInterval)

		w.wg.Add(1)
		go func() {
			defer w.wg.Done()

			// Initial sync after a short delay to let the system warm up
			timer := time.NewTimer(10 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}

			w.syncAll(ctx)

			// Periodic sync
			ticker := time.NewTicker(w.syncInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					w.syncAll(ctx)
				}
			}
		}()
	})
}

// Stop gracefully stops the worker
func (w *MarketplaceWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	slog.Info("MarketplaceWorker stopped")
}

// SyncSingle triggers a manual sync for a single skill registry by ID.
// Used by the Admin API to trigger immediate sync after creation.
func (w *MarketplaceWorker) SyncSingle(ctx context.Context, registryID int64) error {
	registry, err := w.repo.GetSkillRegistry(ctx, registryID)
	if err != nil {
		return fmt.Errorf("skill registry not found: %w", err)
	}

	// Only allow syncing platform-level registries
	if !registry.IsPlatformLevel() {
		return fmt.Errorf("registry %d is not a platform-level registry", registryID)
	}

	slog.InfoContext(ctx, "MarketplaceWorker: manual sync triggered",
		"registry_id", registryID, "url", registry.RepositoryURL)

	if err := w.importer.SyncSource(ctx, registryID); err != nil {
		slog.ErrorContext(ctx, "MarketplaceWorker: manual sync failed",
			"registry_id", registryID, "url", registry.RepositoryURL, "error", err)
		return err
	}

	slog.InfoContext(ctx, "MarketplaceWorker: manual sync completed",
		"registry_id", registryID, "url", registry.RepositoryURL)
	return nil
}

// syncAll queries the DB for every active skill registry (platform + org-level)
// and syncs each one. Org-level inclusion is what makes org-registered skill
// repos appear in the marketplace without manual refresh — see #375.
func (w *MarketplaceWorker) syncAll(ctx context.Context) {
	registries, err := w.repo.ListAllActiveSkillRegistries(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "MarketplaceWorker: failed to list skill registries", "error", err)
		return
	}

	slog.InfoContext(ctx, "MarketplaceWorker: starting sync cycle",
		"registries", len(registries),
		"concurrency", w.syncConcurrency)

	sem := make(chan struct{}, w.syncConcurrency)
	var wg sync.WaitGroup

	for _, reg := range registries {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(r *extension.SkillRegistry) {
			defer wg.Done()
			defer func() { <-sem }()
			w.syncRegistry(ctx, r)
		}(reg)
	}

	wg.Wait()

	// Sync MCP Registry
	if w.registrySyncer != nil {
		if ctx.Err() != nil {
			return
		}
		slog.InfoContext(ctx, "MarketplaceWorker: starting MCP Registry sync")
		if err := w.registrySyncer.Sync(ctx); err != nil {
			slog.ErrorContext(ctx, "MarketplaceWorker: MCP Registry sync failed", "error", err)
		} else {
			slog.InfoContext(ctx, "MarketplaceWorker: MCP Registry sync completed")
		}
	}

	slog.InfoContext(ctx, "MarketplaceWorker: sync cycle completed")
}

func (w *MarketplaceWorker) syncRegistry(ctx context.Context, registry *extension.SkillRegistry) {
	if err := w.importer.SyncSource(ctx, registry.ID); err != nil {
		slog.ErrorContext(ctx, "MarketplaceWorker: sync failed",
			"registry_id", registry.ID, "url", registry.RepositoryURL, "error", err)
	} else {
		slog.InfoContext(ctx, "MarketplaceWorker: sync completed",
			"registry_id", registry.ID, "url", registry.RepositoryURL)
	}
}
