package billing

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// GetPlan returns a plan by name
func (s *Service) GetPlan(ctx context.Context, planName string) (*billing.SubscriptionPlan, error) {
	var plan billing.SubscriptionPlan
	if err := s.db.WithContext(ctx).Where("name = ? AND is_active = ?", planName, true).First(&plan).Error; err != nil {
		return nil, ErrPlanNotFound
	}
	return &plan, nil
}

// ListPlans returns all active plans
func (s *Service) ListPlans(ctx context.Context) ([]*billing.SubscriptionPlan, error) {
	var plans []*billing.SubscriptionPlan
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("price_per_seat_monthly ASC").Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

// GetPlanByID returns a plan by ID
func (s *Service) GetPlanByID(ctx context.Context, planID int64) (*billing.SubscriptionPlan, error) {
	var plan billing.SubscriptionPlan
	if err := s.db.WithContext(ctx).First(&plan, planID).Error; err != nil {
		return nil, ErrPlanNotFound
	}
	return &plan, nil
}

// GetPlanPrice returns price for a plan in specific currency
func (s *Service) GetPlanPrice(ctx context.Context, planName, currency string) (*billing.PlanPrice, error) {
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	var price billing.PlanPrice
	if err := s.db.WithContext(ctx).
		Where("plan_id = ? AND currency = ?", plan.ID, currency).
		First(&price).Error; err != nil {
		return nil, ErrPriceNotFound
	}

	price.Plan = plan
	return &price, nil
}

// GetPlanPriceByID returns price for a plan ID in specific currency
func (s *Service) GetPlanPriceByID(ctx context.Context, planID int64, currency string) (*billing.PlanPrice, error) {
	var price billing.PlanPrice
	if err := s.db.WithContext(ctx).
		Preload("Plan").
		Where("plan_id = ? AND currency = ?", planID, currency).
		First(&price).Error; err != nil {
		return nil, ErrPriceNotFound
	}
	return &price, nil
}

// GetPlanPrices returns all prices for a plan
func (s *Service) GetPlanPrices(ctx context.Context, planName string) ([]billing.PlanPrice, error) {
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	var prices []billing.PlanPrice
	if err := s.db.WithContext(ctx).
		Where("plan_id = ?", plan.ID).
		Find(&prices).Error; err != nil {
		return nil, err
	}

	// Attach plan reference
	for i := range prices {
		prices[i].Plan = plan
	}

	return prices, nil
}

// PlanWithPrice combines a plan with its price in a specific currency
type PlanWithPrice struct {
	Plan  *billing.SubscriptionPlan `json:"plan"`
	Price *billing.PlanPrice        `json:"price"`
}

// ListPlansWithPrices returns all active plans with prices for a specific currency
func (s *Service) ListPlansWithPrices(ctx context.Context, currency string) ([]*PlanWithPrice, error) {
	plans, err := s.ListPlans(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*PlanWithPrice, 0, len(plans))
	for _, plan := range plans {
		var price billing.PlanPrice
		if err := s.db.WithContext(ctx).
			Where("plan_id = ? AND currency = ?", plan.ID, currency).
			First(&price).Error; err != nil {
			// Skip plans without price for this currency
			continue
		}

		result = append(result, &PlanWithPrice{
			Plan:  plan,
			Price: &price,
		})
	}

	return result, nil
}
