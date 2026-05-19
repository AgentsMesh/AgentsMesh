package middleware

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	ID             int64           `gorm:"primaryKey" json:"id"`
	OrganizationID *int64          `gorm:"index" json:"organization_id,omitempty"`
	ActorID        *int64          `gorm:"index" json:"actor_id,omitempty"`
	ActorType      string          `gorm:"size:50;not null" json:"actor_type"` // user, system, runner
	Action         string          `gorm:"size:100;not null;index" json:"action"`
	ResourceType   string          `gorm:"size:50;not null" json:"resource_type"`
	ResourceID     *int64          `json:"resource_id,omitempty"`
	Details        json.RawMessage `gorm:"type:jsonb" json:"details,omitempty"`
	IPAddress      *string         `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent      *string         `gorm:"type:text" json:"user_agent,omitempty"`
	StatusCode     int             `json:"status_code"`
	Duration       int64           `json:"duration_ms"`
	CreatedAt      time.Time       `gorm:"not null;default:now();index" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type AuditConfig struct {
	DB              *gorm.DB
	SkipPaths       []string
	SkipMethods     []string
	CaptureBody     bool
	MaxBodySize     int64
	SensitiveFields []string
}

func DefaultAuditConfig(db *gorm.DB) *AuditConfig {
	return &AuditConfig{
		DB:          db,
		SkipPaths:   []string{"/health", "/metrics", "/api/v1/ws"},
		SkipMethods: []string{"GET", "HEAD", "OPTIONS"},
		CaptureBody: true,
		MaxBodySize: 10 * 1024, // 10KB
		SensitiveFields: []string{
			"password", "token", "secret", "api_key", "access_token",
			"refresh_token", "client_secret", "private_key",
		},
	}
}

type LogActionOptions struct {
	OrganizationID int64
	ActorID        int64
	ActorType      string // user, system, runner
	ResourceType   string
	ResourceID     int64
	StatusCode     int
	Details        map[string]interface{}
	IPAddress      string
	UserAgent      string
}

type AuditLogFilter struct {
	OrganizationID int64
	ActorID        int64
	Action         string
	ResourceType   string
	ResourceID     int64
	StartTime      time.Time
	EndTime        time.Time
	Limit          int
	Offset         int
}
