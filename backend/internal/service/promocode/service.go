package promocode

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
)

var (
	ErrPromoCodeNotFound      = errors.New("promo code not found")
	ErrPromoCodeInvalid       = errors.New("promo code is invalid or expired")
	ErrPromoCodeAlreadyUsed   = errors.New("promo code already used by this organization")
	ErrPromoCodeMaxUses       = errors.New("promo code has reached maximum uses")
	ErrInvalidPlan            = errors.New("invalid plan in promo code")
	ErrNotOwner               = errors.New("only organization owner can redeem promo codes")
	ErrPromoCodeAlreadyExists = errors.New("promo code already exists")
)

// Service handles promo code operations
type Service struct {
	repo     promocode.Repository
	billing  BillingProvider
}

// NewService creates a new promo code service
func NewService(repo promocode.Repository, billing BillingProvider) *Service {
	return &Service{
		repo:     repo,
		billing:  billing,
	}
}

// ValidateRequest represents a validate promo code request
type ValidateRequest struct {
	Code           string
	OrganizationID int64
}

// Error codes for promo code validation
const (
	ErrCodeNotFound      = "promo_code_not_found"
	ErrCodeExpired       = "promo_code_expired"
	ErrCodeDisabled      = "promo_code_disabled"
	ErrCodeMaxUsed       = "promo_code_max_used"
	ErrCodeInvalid       = "promo_code_invalid"
	ErrCodeAlreadyUsed   = "promo_code_already_used"
	ErrCodeNotOwner      = "promo_code_not_owner"
	ErrCodeRedeemSuccess = "promo_code_redeem_success"
)

// ValidateResponse represents a validate promo code response
type ValidateResponse struct {
	Valid           bool       `json:"valid"`
	Code            string     `json:"code"`
	PlanName        string     `json:"plan_name,omitempty"`
	PlanDisplayName string     `json:"plan_display_name,omitempty"`
	DurationMonths  int        `json:"duration_months,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	MessageCode     string     `json:"message_code,omitempty"`
}

// Validate validates a promo code
func (s *Service) Validate(ctx context.Context, req *ValidateRequest) (*ValidateResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))

	promoCode, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if promoCode == nil {
		return &ValidateResponse{Valid: false, Code: code, MessageCode: ErrCodeNotFound}, nil
	}

	// Check basic validity
	if !promoCode.IsValid() {
		messageCode := ErrCodeInvalid
		if promoCode.ExpiresAt != nil && time.Now().After(*promoCode.ExpiresAt) {
			messageCode = ErrCodeExpired
		}
		if promoCode.MaxUses != nil && promoCode.UsedCount >= *promoCode.MaxUses {
			messageCode = ErrCodeMaxUsed
		}
		if !promoCode.IsActive {
			messageCode = ErrCodeDisabled
		}
		return &ValidateResponse{Valid: false, Code: code, MessageCode: messageCode}, nil
	}

	// Check if organization already used this code
	count, err := s.repo.CountOrgRedemptionsForCode(ctx, req.OrganizationID, promoCode.ID)
	if err != nil {
		return nil, err
	}
	if count >= int64(promoCode.MaxUsesPerOrg) {
		return &ValidateResponse{Valid: false, Code: code, MessageCode: ErrCodeAlreadyUsed}, nil
	}

	// Get plan display name via billing provider
	plan, err := s.billing.GetPlanByName(ctx, promoCode.PlanName)
	if err != nil {
		return nil, ErrInvalidPlan
	}

	return &ValidateResponse{
		Valid:           true,
		Code:            promoCode.Code,
		PlanName:        promoCode.PlanName,
		PlanDisplayName: plan.DisplayName,
		DurationMonths:  promoCode.DurationMonths,
		ExpiresAt:       promoCode.ExpiresAt,
	}, nil
}

// RedeemRequest represents a redeem promo code request
type RedeemRequest struct {
	Code           string
	OrganizationID int64
	UserID         int64
	UserRole       string // owner, admin, member
	IPAddress      string
	UserAgent      string
}

// RedeemResponse represents a redeem promo code response
type RedeemResponse struct {
	Success        bool      `json:"success"`
	PlanName       string    `json:"plan_name,omitempty"`
	DurationMonths int       `json:"duration_months,omitempty"`
	NewPeriodEnd   time.Time `json:"new_period_end,omitempty"`
	MessageCode    string    `json:"message_code,omitempty"`
}

// Redeem redeems a promo code
func (s *Service) Redeem(ctx context.Context, req *RedeemRequest) (*RedeemResponse, error) {
	// Check if user is owner
	if req.UserRole != "owner" {
		return &RedeemResponse{Success: false, MessageCode: ErrCodeNotOwner}, nil
	}

	// Validate first
	validateResp, err := s.Validate(ctx, &ValidateRequest{
		Code:           req.Code,
		OrganizationID: req.OrganizationID,
	})
	if err != nil {
		return nil, err
	}
	if !validateResp.Valid {
		return &RedeemResponse{Success: false, MessageCode: validateResp.MessageCode}, nil
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	promoCode, _ := s.repo.GetByCode(ctx, code)

	// Get target plan via billing provider
	targetPlan, err := s.billing.GetActivePlanByName(ctx, promoCode.PlanName)
	if err != nil {
		return nil, ErrInvalidPlan
	}

	// Execute atomic redeem via repository
	var newPeriodEnd time.Time

	redemption := &promocode.Redemption{
		PromoCodeID:    promoCode.ID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		PlanName:       promoCode.PlanName,
		DurationMonths: promoCode.DurationMonths,
		IPAddress:      &req.IPAddress,
		UserAgent:      &req.UserAgent,
	}

	err = s.repo.RedeemAtomic(ctx, &promocode.RedeemAtomicParams{
		Redemption:  redemption,
		PromoCodeID: promoCode.ID,
		ApplyBilling: func(ctx context.Context, tx interface{}) error {
			result, err := s.billing.ApplyPromoSubscription(ctx, tx, &ApplySubscriptionRequest{
				OrganizationID: req.OrganizationID,
				PlanID:         targetPlan.ID,
				DurationMonths: promoCode.DurationMonths,
			})
			if err != nil {
				return err
			}
			// Backfill redemption fields from billing result (pointer, updated before repo.Create)
			newPeriodEnd = result.NewPeriodEnd
			redemption.PreviousPlanName = result.PreviousPlanName
			redemption.PreviousPeriodEnd = result.PreviousPeriodEnd
			redemption.NewPeriodEnd = newPeriodEnd
			return nil
		},
	})

	if err != nil {
		slog.Error("failed to redeem promo code", "code", code, "org_id", req.OrganizationID, "error", err)
		return nil, err
	}

	slog.Info("promo code redeemed", "code", code, "org_id", req.OrganizationID, "user_id", req.UserID, "plan", promoCode.PlanName)
	return &RedeemResponse{
		Success:        true,
		PlanName:       promoCode.PlanName,
		DurationMonths: promoCode.DurationMonths,
		NewPeriodEnd:   newPeriodEnd,
		MessageCode:    ErrCodeRedeemSuccess,
	}, nil
}

// GetRedemptionHistory gets redemption history for an organization
func (s *Service) GetRedemptionHistory(ctx context.Context, orgID int64) ([]*promocode.Redemption, error) {
	return s.repo.GetRedemptionsByOrg(ctx, orgID)
}
