package extension

import (
	"errors"
	"time"
)

// defaultStaleSyncTimeout is the default threshold beyond which a registry
// stuck in sync_status="syncing" is considered abandoned by a crashed/killed
// previous run. Reclaim is handled atomically by Repository.ClaimSyncLock —
// without this knob, an abandoned row would block its registry forever.
// Tunable per-importer via SkillImporter.SetStaleSyncTimeout.
const defaultStaleSyncTimeout = 30 * time.Minute

// ErrSyncInProgress is returned when a fresh peer sync is holding the lock.
// Locks older than the importer's staleSyncTimeout are reclaimed silently
// rather than producing this error.
var ErrSyncInProgress = errors.New("registry sync already in progress")

// SetStaleSyncTimeout overrides the abandoned-lock reclaim threshold. Non-
// positive values are ignored. Call before SyncSource.
func (imp *SkillImporter) SetStaleSyncTimeout(d time.Duration) {
	if d > 0 {
		imp.staleSyncTimeout = d
	}
}
