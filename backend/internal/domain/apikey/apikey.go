package apikey

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type Scope string

const (
	ScopePodRead      Scope = "pods:read"
	ScopePodWrite     Scope = "pods:write"
	ScopeTicketRead   Scope = "tickets:read"
	ScopeTicketWrite  Scope = "tickets:write"
	ScopeChannelRead  Scope = "channels:read"
	ScopeChannelWrite Scope = "channels:write"
	ScopeRunnerRead   Scope = "runners:read"
	ScopeRepoRead     Scope = "repos:read"
	ScopeLoopRead     Scope = "loops:read"
	ScopeLoopWrite    Scope = "loops:write"
)

var AllScopes = map[Scope]bool{
	ScopePodRead:      true,
	ScopePodWrite:     true,
	ScopeTicketRead:   true,
	ScopeTicketWrite:  true,
	ScopeChannelRead:  true,
	ScopeChannelWrite: true,
	ScopeRunnerRead:   true,
	ScopeRepoRead:     true,
	ScopeLoopRead:     true,
	ScopeLoopWrite:    true,
}

func ValidateScope(s string) bool {
	return AllScopes[Scope(s)]
}

type Scopes []Scope

func (s *Scopes) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for Scan")
	}
	return json.Unmarshal(bytes, s)
}

func (s Scopes) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s Scopes) HasScope(scope Scope) bool {
	for _, sc := range s {
		if sc == scope {
			return true
		}
	}
	return false
}

func (s Scopes) ToStrings() []string {
	result := make([]string, len(s))
	for i, sc := range s {
		result[i] = string(sc)
	}
	return result
}

func ScopesFromStrings(ss []string) Scopes {
	result := make(Scopes, len(ss))
	for i, s := range ss {
		result[i] = Scope(s)
	}
	return result
}

type APIKey struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	OrganizationID int64      `gorm:"not null;index" json:"organization_id"`
	Name           string     `gorm:"size:255;not null" json:"name"`
	Slug           *string    `gorm:"size:100;column:slug" json:"slug,omitempty"`
	Description    *string    `gorm:"type:text" json:"description,omitempty"`
	KeyPrefix      string     `gorm:"size:12;not null" json:"key_prefix"`
	KeyHash        string     `gorm:"size:128;uniqueIndex;not null" json:"-"`
	Scopes         Scopes     `gorm:"type:jsonb;not null;default:'[]'" json:"scopes"`
	IsEnabled      bool       `gorm:"not null;default:true" json:"is_enabled"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	CreatedBy      int64      `gorm:"not null" json:"created_by"`
	CreatedAt      time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

func (APIKey) TableName() string {
	return "api_keys"
}

func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

func (k *APIKey) IsValid() bool {
	return k.IsEnabled && !k.IsExpired()
}
