package billing

import "time"

type License struct {
	ID int64 `gorm:"primaryKey" json:"id"`

	LicenseKey string `gorm:"size:255;not null;uniqueIndex" json:"license_key"`

	OrganizationName string `gorm:"size:255;not null" json:"organization_name"`
	ContactEmail     string `gorm:"size:255;not null" json:"contact_email"`

	PlanName          string   `gorm:"size:50;not null" json:"plan_name"`
	MaxUsers          int      `gorm:"not null;default:-1" json:"max_users"`
	MaxRunners        int      `gorm:"not null;default:-1" json:"max_runners"`
	MaxRepositories   int      `gorm:"not null;default:-1" json:"max_repositories"`
	MaxConcurrentPods int      `gorm:"not null;default:-1" json:"max_concurrent_pods"`
	Features          Features `gorm:"type:jsonb;default:'{}'" json:"features,omitempty"`

	IssuedAt  time.Time  `gorm:"not null;default:now()" json:"issued_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	Signature            string  `gorm:"type:text;not null" json:"signature"`
	PublicKeyFingerprint *string `gorm:"size:64" json:"public_key_fingerprint,omitempty"`

	IsActive         bool       `gorm:"not null;default:true" json:"is_active"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason *string    `gorm:"type:text" json:"revocation_reason,omitempty"`

	ActivatedAt    *time.Time `json:"activated_at,omitempty"`
	ActivatedOrgID *int64     `json:"activated_org_id,omitempty"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (License) TableName() string {
	return "licenses"
}

func (l *License) IsValid() bool {
	if !l.IsActive {
		return false
	}
	if l.RevokedAt != nil {
		return false
	}
	if l.ExpiresAt != nil && time.Now().After(*l.ExpiresAt) {
		return false
	}
	return true
}

func (l *License) IsActivated() bool {
	return l.ActivatedAt != nil && l.ActivatedOrgID != nil
}

func (l *License) DaysUntilExpiry() int {
	if l.ExpiresAt == nil {
		return -1
	}
	duration := time.Until(*l.ExpiresAt)
	return int(duration.Hours() / 24)
}

type LicenseStatus struct {
	IsActive         bool       `json:"is_active"`
	LicenseKey       string     `json:"license_key,omitempty"`
	OrganizationName string     `json:"organization_name,omitempty"`
	Plan             string     `json:"plan,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	MaxUsers         int        `json:"max_users,omitempty"`
	MaxRunners       int        `json:"max_runners,omitempty"`
	MaxRepositories  int        `json:"max_repositories,omitempty"`
	MaxPodMinutes    int        `json:"max_pod_minutes,omitempty"` // -1 for unlimited
	Features         []string   `json:"features,omitempty"`
	Message          string     `json:"message,omitempty"`
}
