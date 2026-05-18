package extension

import (
	"context"
	"log/slog"
	"time"
)

// defaultInitialSyncTimeout bounds the initial sync goroutine. Field on
// Service rather than const so tests / deployments can override via
// SetInitialSyncTimeout.
const defaultInitialSyncTimeout = 10 * time.Minute

// SetInitialSyncTimeout overrides the post-create background sync deadline.
// Non-positive values are ignored. Call before CreateSkillRegistry.
func (s *Service) SetInitialSyncTimeout(d time.Duration) {
	if d > 0 {
		s.initialSyncTimeout = d
	}
}

// triggerInitialSync runs SyncSource in the background for a newly created
// registry. The originating HTTP request's context is dead by the time this
// goroutine starts, so we use a detached context with its own timeout.
//
// Errors are intentionally not returned — SyncSource persists success/failure
// to the registry row on normal returns. On panic the row is left in
// sync_status="syncing" and reclaimed by checkSyncLock after the importer's stale timeout.
func (s *Service) triggerInitialSync(registryID int64) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("triggerInitialSync panicked",
				"registry_id", registryID, "panic", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), s.initialSyncTimeout)
	defer cancel()

	if err := s.importer.SyncSource(ctx, registryID); err != nil {
		slog.ErrorContext(ctx, "initial sync after CreateSkillRegistry failed",
			"registry_id", registryID, "error", err)
	}
}
