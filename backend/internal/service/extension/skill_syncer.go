package extension

import "context"

// SkillSyncer is the minimal contract callers need to trigger a sync.
// Service and MarketplaceWorker depend on this interface rather than the
// concrete *SkillImporter, so tests can substitute lightweight fakes
// without standing up the git/storage machinery.
type SkillSyncer interface {
	SyncSource(ctx context.Context, sourceID int64) error
}
