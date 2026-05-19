package admin

import (
	"encoding/json"
	"net"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

type AuditAction string

const (
	AuditActionUserView        AuditAction = "user.view"
	AuditActionUserUpdate      AuditAction = "user.update"
	AuditActionUserDisable     AuditAction = "user.disable"
	AuditActionUserEnable      AuditAction = "user.enable"
	AuditActionUserGrantAdmin    AuditAction = "user.grant_admin"
	AuditActionUserRevokeAdmin   AuditAction = "user.revoke_admin"
	AuditActionUserVerifyEmail   AuditAction = "user.verify_email"
	AuditActionUserUnverifyEmail AuditAction = "user.unverify_email"

	AuditActionOrgView   AuditAction = "organization.view"
	AuditActionOrgUpdate AuditAction = "organization.update"
	AuditActionOrgDelete AuditAction = "organization.delete"

	AuditActionSubView     AuditAction = "subscription.view"
	AuditActionSubUpdate   AuditAction = "subscription.update"
	AuditActionSubExtend   AuditAction = "subscription.extend"
	AuditActionSubCancel   AuditAction = "subscription.cancel"
	AuditActionSubFreeze   AuditAction = "subscription.freeze"
	AuditActionSubUnfreeze AuditAction = "subscription.unfreeze"
	AuditActionSubRenew    AuditAction = "subscription.renew"
	AuditActionSubQuota    AuditAction = "subscription.set_quota"

	AuditActionRunnerView    AuditAction = "runner.view"
	AuditActionRunnerDisable AuditAction = "runner.disable"
	AuditActionRunnerEnable  AuditAction = "runner.enable"
	AuditActionRunnerDelete  AuditAction = "runner.delete"

	AuditActionPromoCodeCreate     AuditAction = "promo_code.create"
	AuditActionPromoCodeUpdate     AuditAction = "promo_code.update"
	AuditActionPromoCodeDelete     AuditAction = "promo_code.delete"
	AuditActionPromoCodeActivate   AuditAction = "promo_code.activate"
	AuditActionPromoCodeDeactivate AuditAction = "promo_code.deactivate"

	AuditActionCreate     AuditAction = "create"
	AuditActionUpdate     AuditAction = "update"
	AuditActionDelete     AuditAction = "delete"
	AuditActionActivate   AuditAction = "activate"
	AuditActionDeactivate AuditAction = "deactivate"

	AuditActionConfigUpdate AuditAction = "config.update"

	AuditActionSupportTicketReply  AuditAction = "support_ticket.reply"
	AuditActionSupportTicketStatus AuditAction = "support_ticket.status"
	AuditActionSupportTicketAssign AuditAction = "support_ticket.assign"
)

type TargetType string

const (
	TargetTypeUser         TargetType = "user"
	TargetTypeOrganization TargetType = "organization"
	TargetTypeSubscription TargetType = "subscription"
	TargetTypeRunner       TargetType = "runner"
	TargetTypePromoCode    TargetType = "promo_code"
	TargetTypeConfig       TargetType = "config"
	TargetTypeSupportTicket TargetType = "support_ticket"
	TargetTypeSSOConfig     TargetType = "sso_config"

	AuditTargetUser          = TargetTypeUser
	AuditTargetOrganization  = TargetTypeOrganization
	AuditTargetSubscription  = TargetTypeSubscription
	AuditTargetRunner        = TargetTypeRunner
	AuditTargetPromoCode     = TargetTypePromoCode
	AuditTargetConfig        = TargetTypeConfig
	AuditTargetSupportTicket = TargetTypeSupportTicket
	AuditTargetSSOConfig     = TargetTypeSSOConfig
)

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

	AdminUser *user.User `gorm:"foreignKey:AdminUserID" json:"admin_user,omitempty"`
}

func (AuditLog) TableName() string {
	return "system_admin_audit_logs"
}

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

type AuditLogListResponse struct {
	Data       []AuditLog `json:"data"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}
