package admin

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

type PromoCodeListFilter struct {
	Type     *promocode.PromoCodeType
	PlanName *string
	IsActive *bool
	Search   *string
	Page     int
	PageSize int
}

type PromoCodeListResult struct {
	Data       []*promocode.PromoCode `json:"data"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

type PromoCodeUpdateInput struct {
	Name           *string
	Description    *string
	MaxUses        *int
	MaxUsesPerOrg  *int
	ExpiresAt      *time.Time
	ClearExpiresAt bool
}

type RedemptionWithDetails struct {
	ID             int64                      `json:"id"`
	PromoCodeID    int64                      `json:"promo_code_id"`
	OrganizationID int64                      `json:"organization_id"`
	UserID         int64                      `json:"user_id"`
	PlanName       string                     `json:"plan_name"`
	DurationMonths int                        `json:"duration_months"`
	NewPeriodEnd   time.Time                  `json:"new_period_end"`
	IPAddress      *string                    `json:"ip_address,omitempty"`
	CreatedAt      time.Time                  `json:"created_at"`
	User           *user.User                 `json:"user,omitempty"`
	Organization   *organization.Organization `json:"organization,omitempty"`
}

type RedemptionListResult struct {
	Data       []*RedemptionWithDetails `json:"data"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

func (s *Service) createPromoCodeAuditLog(ctx context.Context, adminUserID int64, action admin.AuditAction, targetID int64, oldData, newData interface{}) {
	_ = s.LogActionFromContext(ctx, adminUserID, action, admin.AuditTargetPromoCode, targetID, oldData, newData, "", "")
}
