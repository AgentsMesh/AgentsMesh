package extension

import (
	"context"
	"errors"
)

// Domain-level repository errors
var (
	// ErrDuplicateInstall is returned when attempting to install a skill or MCP server
	// that already exists with the same unique key (org + repo + scope + user + slug).
	ErrDuplicateInstall = errors.New("already installed with the same slug in this scope")
)

// Repository defines the data access interface for extension entities
type Repository interface {
	// Skill Registries
	ListSkillRegistries(ctx context.Context, orgID *int64) ([]*SkillRegistry, error)
	GetSkillRegistry(ctx context.Context, id int64) (*SkillRegistry, error)
	CreateSkillRegistry(ctx context.Context, registry *SkillRegistry) error
	UpdateSkillRegistry(ctx context.Context, registry *SkillRegistry) error
	DeleteSkillRegistry(ctx context.Context, id int64) error
	FindSkillRegistryByURL(ctx context.Context, orgID *int64, repoURL string) (*SkillRegistry, error)

	// Skill Market Items
	ListSkillMarketItems(ctx context.Context, orgID *int64, query string, category string) ([]*SkillMarketItem, error)
	GetSkillMarketItem(ctx context.Context, id int64) (*SkillMarketItem, error)
	FindSkillMarketItemBySlug(ctx context.Context, registryID int64, slug string) (*SkillMarketItem, error)
	CreateSkillMarketItem(ctx context.Context, item *SkillMarketItem) error
	UpdateSkillMarketItem(ctx context.Context, item *SkillMarketItem) error
	DeactivateSkillMarketItemsNotIn(ctx context.Context, registryID int64, slugs []string) error

	// MCP Market Items
	ListMcpMarketItems(ctx context.Context, query string, category string, limit, offset int) ([]*McpMarketItem, int64, error)
	GetMcpMarketItem(ctx context.Context, id int64) (*McpMarketItem, error)
	FindMcpMarketItemByRegistryName(ctx context.Context, registryName string) (*McpMarketItem, error)
	UpsertMcpMarketItem(ctx context.Context, item *McpMarketItem) error
	DeactivateMcpMarketItemsNotIn(ctx context.Context, sourceType string, registryNames []string) (int64, error)

	// Installed MCP Servers
	ListInstalledMcpServers(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*InstalledMcpServer, error)
	GetInstalledMcpServer(ctx context.Context, id int64) (*InstalledMcpServer, error)
	CreateInstalledMcpServer(ctx context.Context, server *InstalledMcpServer) error
	UpdateInstalledMcpServer(ctx context.Context, server *InstalledMcpServer) error
	DeleteInstalledMcpServer(ctx context.Context, id int64) error
	GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64) ([]*InstalledMcpServer, error)

	// Installed Skills
	ListInstalledSkills(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*InstalledSkill, error)
	GetInstalledSkill(ctx context.Context, id int64) (*InstalledSkill, error)
	CreateInstalledSkill(ctx context.Context, skill *InstalledSkill) error
	UpdateInstalledSkill(ctx context.Context, skill *InstalledSkill) error
	DeleteInstalledSkill(ctx context.Context, id int64) error
	GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64) ([]*InstalledSkill, error)

	// Skill Registry Overrides
	SetSkillRegistryOverride(ctx context.Context, orgID int64, registryID int64, isDisabled bool) error
	ListSkillRegistryOverrides(ctx context.Context, orgID int64) ([]*SkillRegistryOverride, error)
}
