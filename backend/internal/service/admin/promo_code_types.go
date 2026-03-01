package admin

import (
	"context"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/promocode"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
)

// PromoCodeListFilter represents filter options for listing promo codes
type PromoCodeListFilter struct {
	Type     *promocode.PromoCodeType
	PlanName *string
	IsActive *bool
	Search   *string
	Page     int
	PageSize int
}

// PromoCodeListResult represents the result of listing promo codes
type PromoCodeListResult struct {
	Data       []*promocode.PromoCode `json:"data"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

// PromoCodeUpdateInput represents the input for updating a promo code
type PromoCodeUpdateInput struct {
	Name           *string
	Description    *string
	MaxUses        *int
	MaxUsesPerOrg  *int
	ExpiresAt      *time.Time
	ClearExpiresAt bool
}

// RedemptionWithDetails represents a redemption with user and organization details
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

// RedemptionListResult represents the result of listing redemptions
type RedemptionListResult struct {
	Data       []*RedemptionWithDetails `json:"data"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

// createPromoCodeAuditLog is a helper to create audit log entries for promo code operations
func (s *Service) createPromoCodeAuditLog(ctx context.Context, adminUserID int64, action admin.AuditAction, targetID int64, oldData, newData interface{}) {
	// Best effort - don't fail the main operation if audit logging fails
	_ = s.LogActionFromContext(ctx, adminUserID, action, admin.AuditTargetPromoCode, targetID, oldData, newData, "", "")
}
