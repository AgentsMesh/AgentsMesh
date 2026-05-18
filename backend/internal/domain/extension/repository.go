package extension

import (
	"context"
	"errors"
	"time"
)

// Domain-level repository errors
var (
	// ErrDuplicateInstall is returned when attempting to install a skill or MCP server
	// that already exists with the same unique key (org + repo + scope + user + slug).
	ErrDuplicateInstall = errors.New("already installed with the same slug in this scope")
)

// SkillRegistryRepository owns skill-registry rows: lifecycle, lookup,
// per-cycle listing for the marketplace worker, and the atomic CAS lock
// used to prevent concurrent syncs of the same registry.
type SkillRegistryRepository interface {
	ListSkillRegistries(ctx context.Context, orgID *int64) ([]*SkillRegistry, error)
	// ListAllActiveSkillRegistries returns every active registry across the
	// platform and all organizations. Used by MarketplaceWorker so org-level
	// registries get periodically re-synced just like platform-level ones.
	ListAllActiveSkillRegistries(ctx context.Context) ([]*SkillRegistry, error)
	GetSkillRegistry(ctx context.Context, id int64) (*SkillRegistry, error)
	CreateSkillRegistry(ctx context.Context, registry *SkillRegistry) error
	UpdateSkillRegistry(ctx context.Context, registry *SkillRegistry) error
	DeleteSkillRegistry(ctx context.Context, id int64) error
	FindSkillRegistryByURL(ctx context.Context, orgID *int64, repoURL string) (*SkillRegistry, error)
	// ClaimSyncLock atomically marks the registry as syncing if no fresh
	// peer is already holding the lock. Returns claimed=true if the caller
	// owns the lock (and may proceed to clone/import), claimed=false if a
	// fresh peer sync is in progress. wasStale signals whether the lock was
	// reclaimed from an abandoned previous run (older than staleAfter).
	ClaimSyncLock(ctx context.Context, registryID int64, staleAfter time.Duration) (claimed bool, wasStale bool, err error)
}

// SkillMarketRepository owns the catalog of synced skills surfaced to the
// marketplace UI. Writes are exclusively driven by SkillImporter.doSync —
// other paths must not mutate SkillRegistry.SkillCount unless they also
// update this table consistently.
type SkillMarketRepository interface {
	ListSkillMarketItems(ctx context.Context, orgID *int64, query string, category string) ([]*SkillMarketItem, error)
	GetSkillMarketItem(ctx context.Context, id int64) (*SkillMarketItem, error)
	FindSkillMarketItemBySlug(ctx context.Context, registryID int64, slug string) (*SkillMarketItem, error)
	CreateSkillMarketItem(ctx context.Context, item *SkillMarketItem) error
	UpdateSkillMarketItem(ctx context.Context, item *SkillMarketItem) error
	DeactivateSkillMarketItemsNotIn(ctx context.Context, registryID int64, slugs []string) error
}

// SkillRegistryOverrideRepository owns the per-org enable/disable bits
// stacked on top of platform-level registries.
type SkillRegistryOverrideRepository interface {
	SetSkillRegistryOverride(ctx context.Context, orgID int64, registryID int64, isDisabled bool) error
	ListSkillRegistryOverrides(ctx context.Context, orgID int64) ([]*SkillRegistryOverride, error)
}

// McpMarketRepository owns the MCP server catalog populated by the
// upstream registry syncer.
type McpMarketRepository interface {
	ListMcpMarketItems(ctx context.Context, query string, category string, limit, offset int) ([]*McpMarketItem, int64, error)
	GetMcpMarketItem(ctx context.Context, id int64) (*McpMarketItem, error)
	FindMcpMarketItemByRegistryName(ctx context.Context, registryName string) (*McpMarketItem, error)
	UpsertMcpMarketItem(ctx context.Context, item *McpMarketItem) error
	BatchUpsertMcpMarketItems(ctx context.Context, items []*McpMarketItem) error
	DeactivateMcpMarketItemsNotIn(ctx context.Context, sourceType string, registryNames []string) (int64, error)
}

// InstalledMcpRepository owns per-repository installations of MCP servers.
type InstalledMcpRepository interface {
	ListInstalledMcpServers(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*InstalledMcpServer, error)
	GetInstalledMcpServer(ctx context.Context, id int64) (*InstalledMcpServer, error)
	CreateInstalledMcpServer(ctx context.Context, server *InstalledMcpServer) error
	UpdateInstalledMcpServer(ctx context.Context, server *InstalledMcpServer) error
	DeleteInstalledMcpServer(ctx context.Context, id int64) error
	GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64) ([]*InstalledMcpServer, error)
}

// InstalledSkillRepository owns per-repository installations of skills.
type InstalledSkillRepository interface {
	ListInstalledSkills(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*InstalledSkill, error)
	GetInstalledSkill(ctx context.Context, id int64) (*InstalledSkill, error)
	CreateInstalledSkill(ctx context.Context, skill *InstalledSkill) error
	UpdateInstalledSkill(ctx context.Context, skill *InstalledSkill) error
	DeleteInstalledSkill(ctx context.Context, id int64) error
	GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64) ([]*InstalledSkill, error)
}

// Repository is the union of the focused sub-interfaces. Callers should
// prefer to depend on the narrowest sub-interface they actually need —
// this aggregate exists for wiring (single concrete impl) and for tests
// that need to mock the full surface.
type Repository interface {
	SkillRegistryRepository
	SkillMarketRepository
	SkillRegistryOverrideRepository
	McpMarketRepository
	InstalledMcpRepository
	InstalledSkillRepository
}
