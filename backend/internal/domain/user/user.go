package user

import (
	"time"
)

// User represents a user in the system (globally unique)
type User struct {
	ID           int64   `gorm:"primaryKey" json:"id"`
	Email        string  `gorm:"size:255;not null;uniqueIndex" json:"email"`
	Username     string  `gorm:"size:255;not null;uniqueIndex" json:"username"`
	Name         *string `gorm:"size:255" json:"name,omitempty"`
	AvatarURL    *string `gorm:"type:text" json:"avatar_url,omitempty"`
	PasswordHash *string `gorm:"size:255" json:"-"` // Never expose in JSON

	IsActive      bool       `gorm:"not null;default:true" json:"is_active"`
	IsSystemAdmin bool       `gorm:"not null;default:false" json:"is_system_admin"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`

	IsEmailVerified            bool       `gorm:"not null;default:false" json:"is_email_verified"`
	EmailVerificationToken     *string    `gorm:"size:255" json:"-"`
	EmailVerificationExpiresAt *time.Time `json:"-"`

	PasswordResetToken     *string    `gorm:"size:255" json:"-"`
	PasswordResetExpiresAt *time.Time `json:"-"`

	DefaultGitCredentialID *int64 `gorm:"index" json:"default_git_credential_id,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Identities           []Identity       `gorm:"foreignKey:UserID" json:"identities,omitempty"`
	DefaultGitCredential *GitCredential   `gorm:"foreignKey:DefaultGitCredentialID" json:"default_git_credential,omitempty"`
}

func (User) TableName() string {
	return "users"
}

type Identity struct {
	ID     int64 `gorm:"primaryKey" json:"id"`
	UserID int64 `gorm:"not null;index" json:"user_id"`

	Provider         string  `gorm:"size:50;not null" json:"provider"` // github, google, gitlab, gitee
	ProviderUserID   string  `gorm:"size:255;not null" json:"provider_user_id"`
	ProviderUsername *string `gorm:"size:255" json:"provider_username,omitempty"`

	AccessTokenEncrypted  *string    `gorm:"type:text" json:"-"`
	RefreshTokenEncrypted *string    `gorm:"type:text" json:"-"`
	TokenExpiresAt        *time.Time `json:"token_expires_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Identity) TableName() string {
	return "user_identities"
}

type UserWithOrgs struct {
	User
	Organizations []UserOrganization `json:"organizations,omitempty"`
}

type UserOrganization struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Role     string `json:"role"`
	LogoURL  string `json:"logo_url,omitempty"`
	JoinedAt string `json:"joined_at"`
}
