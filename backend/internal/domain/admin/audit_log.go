package admin

import (
	"encoding/json"
	"net"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// AuditAction represents types of admin actions
type AuditAction string

const (
	// User actions
	AuditActionUserView        AuditAction = "user.view"
	AuditActionUserUpdate      AuditAction = "user.update"
	AuditActionUserDisable     AuditAction = "user.disable"
	AuditActionUserEnable      AuditAction = "user.enable"
	AuditActionUserGrantAdmin  AuditAction = "user.grant_admin"
	AuditActionUserRevokeAdmin AuditAction = "user.revoke_admin"

	// Organization actions
	AuditActionOrgView   AuditAction = "organization.view"
	AuditActionOrgUpdate AuditAction = "organization.update"
	AuditActionOrgDelete AuditAction = "organization.delete"

	// Subscription actions
	AuditActionSubView     AuditAction = "subscription.view"
	AuditActionSubUpdate   AuditAction = "subscription.update"
	AuditActionSubExtend   AuditAction = "subscription.extend"
	AuditActionSubCancel   AuditAction = "subscription.cancel"
	AuditActionSubFreeze   AuditAction = "subscription.freeze"
	AuditActionSubUnfreeze AuditAction = "subscription.unfreeze"
	AuditActionSubRenew    AuditAction = "subscription.renew"
	AuditActionSubQuota    AuditAction = "subscription.set_quota"

	// Runner actions
	AuditActionRunnerView    AuditAction = "runner.view"
	AuditActionRunnerDisable AuditAction = "runner.disable"
	AuditActionRunnerEnable  AuditAction = "runner.enable"
	AuditActionRunnerDelete  AuditAction = "runner.delete"

	// Promo code actions
	AuditActionPromoCodeCreate     AuditAction = "promo_code.create"
	AuditActionPromoCodeUpdate     AuditAction = "promo_code.update"
	AuditActionPromoCodeDelete     AuditAction = "promo_code.delete"
	AuditActionPromoCodeActivate   AuditAction = "promo_code.activate"
	AuditActionPromoCodeDeactivate AuditAction = "promo_code.deactivate"

	// Generic actions (for new entity types)
	AuditActionCreate     AuditAction = "create"
	AuditActionUpdate     AuditAction = "update"
	AuditActionDelete     AuditAction = "delete"
	AuditActionActivate   AuditAction = "activate"
	AuditActionDeactivate AuditAction = "deactivate"

	// Config actions
	AuditActionConfigUpdate AuditAction = "config.update"
)

// TargetType represents the type of entity being acted upon
type TargetType string

const (
	TargetTypeUser         TargetType = "user"
	TargetTypeOrganization TargetType = "organization"
	TargetTypeSubscription TargetType = "subscription"
	TargetTypeRunner       TargetType = "runner"
	TargetTypePromoCode    TargetType = "promo_code"
	TargetTypeConfig       TargetType = "config"

	// Aliases for convenience
	AuditTargetUser         = TargetTypeUser
	AuditTargetOrganization = TargetTypeOrganization
	AuditTargetSubscription = TargetTypeSubscription
	AuditTargetRunner       = TargetTypeRunner
	AuditTargetPromoCode    = TargetTypePromoCode
	AuditTargetConfig       = TargetTypeConfig
)

// AuditLog represents a system admin audit log entry
type AuditLog struct {
	ID          int64       `gorm:"primaryKey" json:"id"`
	AdminUserID int64       `gorm:"not null;index" json:"admin_user_id"`
	Action      AuditAction `gorm:"size:100;not null" json:"action"`
	TargetType  TargetType  `gorm:"size:50;not null" json:"target_type"`
	TargetID    int64       `gorm:"not null" json:"target_id"`
	OldData     *string     `gorm:"type:jsonb" json:"old_data,omitempty"`
	NewData     *string     `gorm:"type:jsonb" json:"new_data,omitempty"`
	IPAddress   *string     `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent   *string     `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt   time.Time   `gorm:"not null;default:now()" json:"created_at"`

	// Associations (for eager loading)
	AdminUser *user.User `gorm:"foreignKey:AdminUserID" json:"admin_user,omitempty"`
}

func (AuditLog) TableName() string {
	return "system_admin_audit_logs"
}

// AuditLogEntry is used for creating new audit log entries
type AuditLogEntry struct {
	AdminUserID int64
	Action      AuditAction
	TargetType  TargetType
	TargetID    int64
	OldData     interface{}
	NewData     interface{}
	IPAddress   net.IP
	UserAgent   string
}

// ToAuditLog converts AuditLogEntry to AuditLog for database storage
func (e *AuditLogEntry) ToAuditLog() (*AuditLog, error) {
	log := &AuditLog{
		AdminUserID: e.AdminUserID,
		Action:      e.Action,
		TargetType:  e.TargetType,
		TargetID:    e.TargetID,
		CreatedAt:   time.Now(),
	}

	if e.OldData != nil {
		data, err := json.Marshal(e.OldData)
		if err != nil {
			return nil, err
		}
		str := string(data)
		log.OldData = &str
	}

	if e.NewData != nil {
		data, err := json.Marshal(e.NewData)
		if err != nil {
			return nil, err
		}
		str := string(data)
		log.NewData = &str
	}

	if e.IPAddress != nil {
		str := e.IPAddress.String()
		log.IPAddress = &str
	}

	if e.UserAgent != "" {
		log.UserAgent = &e.UserAgent
	}

	return log, nil
}

// AuditLogQuery represents query parameters for audit logs
type AuditLogQuery struct {
	AdminUserID *int64
	Action      *AuditAction
	TargetType  *TargetType
	TargetID    *int64
	StartTime   *time.Time
	EndTime     *time.Time
	Page        int
	PageSize    int
}

// AuditLogListResponse represents paginated audit log response
type AuditLogListResponse struct {
	Data       []AuditLog `json:"data"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}
