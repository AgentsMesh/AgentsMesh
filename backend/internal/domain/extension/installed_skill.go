package extension

import "time"

// Install source constants
const (
	InstallSourceMarket = "market"
	InstallSourceGitHub = "github"
	InstallSourceUpload = "upload"
)

// InstalledSkill represents a Skill installed for a specific repository
type InstalledSkill struct {
	ID             int64   `gorm:"primaryKey" json:"id"`
	OrganizationID int64   `gorm:"not null" json:"organization_id"`
	RepositoryID   int64   `gorm:"not null" json:"repository_id"`
	MarketItemID   *int64  `json:"market_item_id,omitempty"`
	Scope          string  `gorm:"size:20;not null" json:"scope"` // org / user
	InstalledBy    *int64  `json:"installed_by,omitempty"`
	Slug           string  `gorm:"size:100;not null" json:"slug"`
	InstallSource  string  `gorm:"size:20;not null" json:"install_source"` // market / github / upload
	SourceURL      string  `gorm:"size:500" json:"source_url,omitempty"`
	ContentSha     string  `gorm:"size:64" json:"content_sha,omitempty"`
	StorageKey     string  `gorm:"size:500" json:"storage_key,omitempty"`
	PackageSize    int64   `json:"package_size"`
	PinnedVersion  *int    `json:"pinned_version,omitempty"`
	IsEnabled      bool    `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt      time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	MarketItem *SkillMarketItem `gorm:"foreignKey:MarketItemID" json:"market_item,omitempty"`
}

func (InstalledSkill) TableName() string { return "installed_skills" }

// GetEffectiveSha returns the content SHA to use (from market item if tracking latest, or own SHA)
func (s *InstalledSkill) GetEffectiveSha() string {
	if s.InstallSource == InstallSourceMarket && s.PinnedVersion == nil && s.MarketItem != nil {
		return s.MarketItem.ContentSha
	}
	return s.ContentSha
}

// GetEffectiveStorageKey returns the storage key to use
func (s *InstalledSkill) GetEffectiveStorageKey() string {
	if s.InstallSource == InstallSourceMarket && s.PinnedVersion == nil && s.MarketItem != nil {
		return s.MarketItem.StorageKey
	}
	return s.StorageKey
}

// GetEffectivePackageSize returns the package size to use
func (s *InstalledSkill) GetEffectivePackageSize() int64 {
	if s.InstallSource == InstallSourceMarket && s.PinnedVersion == nil && s.MarketItem != nil {
		return s.MarketItem.PackageSize
	}
	return s.PackageSize
}
