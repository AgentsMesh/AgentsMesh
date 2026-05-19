package user

import (
	"time"
)

type RepositoryProvider struct {
	ID     int64 `gorm:"primaryKey" json:"id"`
	UserID int64 `gorm:"not null;index" json:"user_id"`

	ProviderType string `gorm:"size:50;not null" json:"provider_type"` // github, gitlab, gitee
	Name         string `gorm:"size:100;not null" json:"name"`         // User-defined name
	BaseURL      string `gorm:"size:255;not null" json:"base_url"`     // https://github.com, https://gitlab.company.com

	IdentityID *int64    `gorm:"index" json:"identity_id,omitempty"`
	Identity   *Identity `gorm:"foreignKey:IdentityID" json:"identity,omitempty"`

	ClientID              *string `gorm:"size:255" json:"client_id,omitempty"`
	ClientSecretEncrypted *string `gorm:"type:text" json:"-"`

	BotTokenEncrypted *string `gorm:"type:text" json:"-"`

	IsDefault bool `gorm:"not null;default:false" json:"is_default"`
	IsActive  bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	User           *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	GitCredentials []*GitCredential `gorm:"foreignKey:RepositoryProviderID" json:"git_credentials,omitempty"`
}

func (RepositoryProvider) TableName() string {
	return "user_repository_providers"
}

type RepositoryProviderResponse struct {
	ID           int64  `json:"id"`
	ProviderType string `json:"provider_type"`
	Name         string `json:"name"`
	BaseURL      string `json:"base_url"`
	HasClientID  bool   `json:"has_client_id"`
	HasBotToken  bool   `json:"has_bot_token"`
	HasIdentity  bool   `json:"has_identity"`  // Has linked OAuth identity with access token
	IsDefault    bool   `json:"is_default"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (p *RepositoryProvider) ToResponse() *RepositoryProviderResponse {
	hasIdentity := p.IdentityID != nil &&
		p.Identity != nil &&
		p.Identity.AccessTokenEncrypted != nil &&
		*p.Identity.AccessTokenEncrypted != ""

	return &RepositoryProviderResponse{
		ID:           p.ID,
		ProviderType: p.ProviderType,
		Name:         p.Name,
		BaseURL:      p.BaseURL,
		HasClientID:  p.ClientID != nil && *p.ClientID != "",
		HasBotToken:  p.BotTokenEncrypted != nil && *p.BotTokenEncrypted != "",
		HasIdentity:  hasIdentity,
		IsDefault:    p.IsDefault,
		IsActive:     p.IsActive,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
	}
}

const (
	ProviderTypeGitHub = "github"
	ProviderTypeGitLab = "gitlab"
	ProviderTypeGitee  = "gitee"
)

func ValidProviderTypes() []string {
	return []string{ProviderTypeGitHub, ProviderTypeGitLab, ProviderTypeGitee}
}

func IsValidProviderType(providerType string) bool {
	for _, t := range ValidProviderTypes() {
		if t == providerType {
			return true
		}
	}
	return false
}
