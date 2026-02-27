package extension

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// MarketplaceWorker runs background sync for platform-level Skill Registries
// and the official MCP Registry.
type MarketplaceWorker struct {
	importer       *SkillImporter
	registrySyncer *McpRegistrySyncer
	repo           extension.Repository
	syncInterval   time.Duration

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	startOnce sync.Once
}

// NewMarketplaceWorker creates a new MarketplaceWorker.
// Registries are read from the DB on each sync cycle (no static URL list).
// registrySyncer may be nil if MCP Registry sync is disabled.
func NewMarketplaceWorker(
	repo extension.Repository,
	importer *SkillImporter,
	registrySyncer *McpRegistrySyncer,
	syncInterval time.Duration,
) *MarketplaceWorker {
	return &MarketplaceWorker{
		importer:       importer,
		registrySyncer: registrySyncer,
		repo:           repo,
		syncInterval:   syncInterval,
	}
}

// Start begins the background sync loop.
// It performs an initial sync, then repeats at the configured interval.
// Calling Start multiple times is safe; only the first call has any effect.
func (w *MarketplaceWorker) Start(ctx context.Context) {
	w.startOnce.Do(func() {
		ctx, w.cancel = context.WithCancel(ctx)

		slog.Info("MarketplaceWorker starting",
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

	slog.Info("MarketplaceWorker: manual sync triggered",
		"registry_id", registryID, "url", registry.RepositoryURL)

	if err := w.importer.SyncSource(ctx, registryID); err != nil {
		slog.Error("MarketplaceWorker: manual sync failed",
			"registry_id", registryID, "url", registry.RepositoryURL, "error", err)
		return err
	}

	slog.Info("MarketplaceWorker: manual sync completed",
		"registry_id", registryID, "url", registry.RepositoryURL)
	return nil
}

// syncAll queries the DB for all platform-level skill registries and syncs each one.
func (w *MarketplaceWorker) syncAll(ctx context.Context) {
	// Query platform-level registries (organization_id IS NULL)
	registries, err := w.repo.ListSkillRegistries(ctx, nil)
	if err != nil {
		slog.Error("MarketplaceWorker: failed to list platform skill registries", "error", err)
		return
	}

	slog.Info("MarketplaceWorker: starting sync cycle",
		"registries", len(registries))

	for _, reg := range registries {
		if ctx.Err() != nil {
			return
		}
		if !reg.IsActive {
			continue
		}
		w.syncRegistry(ctx, reg)
	}

	// Sync MCP Registry
	if w.registrySyncer != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Info("MarketplaceWorker: starting MCP Registry sync")
		if err := w.registrySyncer.Sync(ctx); err != nil {
			slog.Error("MarketplaceWorker: MCP Registry sync failed", "error", err)
		} else {
			slog.Info("MarketplaceWorker: MCP Registry sync completed")
		}
	}

	slog.Info("MarketplaceWorker: sync cycle completed")
}

// syncRegistry syncs a single platform-level skill registry
func (w *MarketplaceWorker) syncRegistry(ctx context.Context, registry *extension.SkillRegistry) {
	if err := w.importer.SyncSource(ctx, registry.ID); err != nil {
		slog.Error("MarketplaceWorker: sync failed",
			"registry_id", registry.ID, "url", registry.RepositoryURL, "error", err)
	} else {
		slog.Info("MarketplaceWorker: sync completed",
			"registry_id", registry.ID, "url", registry.RepositoryURL)
	}
}
