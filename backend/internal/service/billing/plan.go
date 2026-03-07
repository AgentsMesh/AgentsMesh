package billing

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// GetPlan returns a plan by name
func (s *Service) GetPlan(ctx context.Context, planName string) (*billing.SubscriptionPlan, error) {
	plan, err := s.repo.GetPlanByName(ctx, planName)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

// ListPlans returns all active plans
func (s *Service) ListPlans(ctx context.Context) ([]*billing.SubscriptionPlan, error) {
	return s.repo.ListActivePlans(ctx)
}

// GetPlanByID returns a plan by ID
func (s *Service) GetPlanByID(ctx context.Context, planID int64) (*billing.SubscriptionPlan, error) {
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

// GetPlanPrice returns price for a plan in specific currency
func (s *Service) GetPlanPrice(ctx context.Context, planName, currency string) (*billing.PlanPrice, error) {
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	price, err := s.repo.GetPlanPrice(ctx, plan.ID, currency)
	if err != nil {
		return nil, err
	}
	if price == nil {
		return nil, ErrPriceNotFound
	}

	price.Plan = plan
	return price, nil
}

// GetPlanPriceByID returns price for a plan ID in specific currency
func (s *Service) GetPlanPriceByID(ctx context.Context, planID int64, currency string) (*billing.PlanPrice, error) {
	price, err := s.repo.GetPlanPrice(ctx, planID, currency)
	if err != nil {
		return nil, err
	}
	if price == nil {
		return nil, ErrPriceNotFound
	}
	return price, nil
}

// GetPlanPrices returns all prices for a plan
func (s *Service) GetPlanPrices(ctx context.Context, planName string) ([]billing.PlanPrice, error) {
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	prices, err := s.repo.ListPlanPrices(ctx, plan.ID)
	if err != nil {
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
		price, err := s.repo.GetPlanPrice(ctx, plan.ID, currency)
		if err != nil || price == nil {
			// Skip plans without price for this currency
			continue
		}

		result = append(result, &PlanWithPrice{
			Plan:  plan,
			Price: price,
		})
	}

	return result, nil
}
