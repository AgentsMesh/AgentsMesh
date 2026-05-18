package extension

import (
	"context"
	"sync"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// TestSyncAll_IncludesOrgLevelRegistries is the regression test for issue #375:
// MarketplaceWorker.syncAll used to query only platform-level registries
// (organization_id IS NULL), so skills registered at the org level never
// populated skill_market_items. The fix routes syncAll through
// ListAllActiveSkillRegistries which returns every active row regardless of
// organization_id.
func TestSyncAll_IncludesOrgLevelRegistries(t *testing.T) {
	repo := newMockExtensionRepo()

	orgID := int64(42)
	var mu sync.Mutex
	syncedIDs := []int64{}

	repo.listAllActiveRegistriesFunc = func(_ context.Context) ([]*extension.SkillRegistry, error) {
		return []*extension.SkillRegistry{
			{ID: 1, OrganizationID: nil, RepositoryURL: "https://github.com/platform/skills", Branch: "main", IsActive: true},
			{ID: 2, OrganizationID: &orgID, RepositoryURL: "https://github.com/org/skills", Branch: "main", IsActive: true},
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
		t.Fatalf("expected both platform and org-level registries synced, got %d: %v", len(syncedIDs), syncedIDs)
	}
	hasPlatform, hasOrg := false, false
	for _, id := range syncedIDs {
		if id == 1 {
			hasPlatform = true
		}
		if id == 2 {
			hasOrg = true
		}
	}
	if !hasPlatform {
		t.Error("platform-level registry (id=1) was not synced")
	}
	if !hasOrg {
		t.Error("org-level registry (id=2) was not synced — this is the #375 regression path")
	}
}
