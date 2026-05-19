package user

import (
	"time"
)

type GitCredential struct {
	ID     int64 `gorm:"primaryKey" json:"id"`
	UserID int64 `gorm:"not null;index" json:"user_id"`

	Name           string `gorm:"size:100;not null" json:"name"`
	CredentialType string `gorm:"size:20;not null" json:"credential_type"` // runner_local, oauth, pat, ssh_key

	RepositoryProviderID *int64 `gorm:"index" json:"repository_provider_id,omitempty"`

	PATEncrypted *string `gorm:"type:text;column:pat_encrypted" json:"-"`

	PublicKey           *string `gorm:"type:text" json:"public_key,omitempty"`
	PrivateKeyEncrypted *string `gorm:"type:text" json:"-"`
	Fingerprint         *string `gorm:"size:255" json:"fingerprint,omitempty"`

	HostPattern *string `gorm:"size:255" json:"host_pattern,omitempty"` // e.g., github.com, *, etc.

	IsDefault bool `gorm:"not null;default:false" json:"is_default"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	User               *User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RepositoryProvider *RepositoryProvider `gorm:"foreignKey:RepositoryProviderID" json:"repository_provider,omitempty"`
}

func (GitCredential) TableName() string {
	return "user_git_credentials"
}

const (
	CredentialTypeRunnerLocal = "runner_local"
	CredentialTypeOAuth       = "oauth"
	CredentialTypePAT         = "pat"
	CredentialTypeSSHKey      = "ssh_key"
)

func ValidCredentialTypes() []string {
	return []string{CredentialTypeRunnerLocal, CredentialTypeOAuth, CredentialTypePAT, CredentialTypeSSHKey}
}

func IsValidCredentialType(credentialType string) bool {
	for _, t := range ValidCredentialTypes() {
		if t == credentialType {
			return true
		}
	}
	return false
}

type GitCredentialResponse struct {
	ID                   int64   `json:"id"`
	Name                 string  `json:"name"`
	CredentialType       string  `json:"credential_type"`
	RepositoryProviderID *int64  `json:"repository_provider_id,omitempty"`
	ProviderName         *string `json:"provider_name,omitempty"` // Populated from RepositoryProvider
	PublicKey            *string `json:"public_key,omitempty"`
	Fingerprint          *string `json:"fingerprint,omitempty"`
	HostPattern          *string `json:"host_pattern,omitempty"`
	IsDefault            bool    `json:"is_default"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

func (c *GitCredential) ToResponse() *GitCredentialResponse {
	resp := &GitCredentialResponse{
		ID:                   c.ID,
		Name:                 c.Name,
		CredentialType:       c.CredentialType,
		RepositoryProviderID: c.RepositoryProviderID,
		PublicKey:            c.PublicKey,
		Fingerprint:          c.Fingerprint,
		HostPattern:          c.HostPattern,
		IsDefault:            c.IsDefault,
		CreatedAt:            c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            c.UpdatedAt.Format(time.RFC3339),
	}

	if c.RepositoryProvider != nil {
		resp.ProviderName = &c.RepositoryProvider.Name
	}

	return resp
}

func RunnerLocalCredentialResponse() *GitCredentialResponse {
	return &GitCredentialResponse{
		ID:             0,
		Name:           "Runner Local",
		CredentialType: CredentialTypeRunnerLocal,
		IsDefault:      false, // Will be set based on user preference
		CreatedAt:      "",
		UpdatedAt:      "",
	}
}
