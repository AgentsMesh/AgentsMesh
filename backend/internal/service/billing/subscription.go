package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// GetSubscription returns subscription for an organization
func (s *Service) GetSubscription(ctx context.Context, orgID int64) (*billing.Subscription, error) {
	sub, err := s.repo.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

// CreateSubscription creates a new subscription
func (s *Service) CreateSubscription(ctx context.Context, orgID int64, planName string) (*billing.Subscription, error) {
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // 1 month

	sub := &billing.Subscription{
		OrganizationID:     orgID,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	sub.Plan = plan
	return sub, nil
}

// CreateTrialSubscription creates a trial subscription for a new organization.
// NOTE: This uses the service's own repo. If the org was created in a
// transaction that hasn't committed yet, use CreateTrialSubscriptionTx instead.
func (s *Service) CreateTrialSubscription(ctx context.Context, orgID int64, planName string, trialDays int) (*billing.Subscription, error) {
	return s.createTrialSubscription(ctx, s.repo, orgID, planName, trialDays)
}

// CreateTrialSubscriptionTx creates a trial subscription using the provided transaction handle.
// This ensures the subscription insert can see the org record created in the same transaction.
// The rawTx parameter must be the underlying DB transaction type (e.g. *gorm.DB).
func (s *Service) CreateTrialSubscriptionTx(ctx context.Context, rawTx interface{}, orgID int64, planName string, trialDays int) (*billing.Subscription, error) {
	txRepo := s.repo.Scoped(rawTx)
	return s.createTrialSubscription(ctx, txRepo, orgID, planName, trialDays)
}

func (s *Service) createTrialSubscription(ctx context.Context, repo billing.BillingRepository, orgID int64, planName string, trialDays int) (*billing.Subscription, error) {
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	if trialDays <= 0 {
		trialDays = billing.DefaultTrialDays
	}

	now := time.Now()
	periodEnd := now.AddDate(0, 0, trialDays)

	sub := &billing.Subscription{
		OrganizationID:     orgID,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusTrialing,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		SeatCount:          1,
	}

	if err := repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	sub.Plan = plan
	return sub, nil
}

// AdminCreateSubscription creates a new active subscription for an organization that doesn't have one.
func (s *Service) AdminCreateSubscription(ctx context.Context, orgID int64, planName string, months int) (*billing.Subscription, error) {
	_, err := s.GetSubscription(ctx, orgID)
	if err == nil {
		return nil, ErrSubscriptionAlreadyExists
	}

	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	if months <= 0 {
		months = 1
	}

	now := time.Now()
	periodEnd := now.AddDate(0, months, 0)

	sub := &billing.Subscription{
		OrganizationID:     orgID,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		SeatCount:          1,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	// Sync organization table redundant fields
	s.syncOrganizationSubscription(ctx, orgID, &plan.Name, strPtr(billing.SubscriptionStatusActive))

	sub.Plan = plan
	return sub, nil
}

// AdminUpdatePlan directly changes the subscription plan without payment checks or downgrade delays.
func (s *Service) AdminUpdatePlan(ctx context.Context, orgID int64, planName string) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	newPlan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	slog.Info("admin updating subscription plan",
		"org_id", orgID, "plan_name", planName,
		"old_plan_id", sub.PlanID, "new_plan_id", newPlan.ID, "new_plan_name", newPlan.Name)

	if err := s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"plan_id":           newPlan.ID,
		"downgrade_to_plan": nil,
	}); err != nil {
		return nil, err
	}

	// Sync organization table redundant fields
	s.syncOrganizationSubscription(ctx, orgID, &newPlan.Name, nil)

	sub.PlanID = newPlan.ID
	sub.DowngradeToPlan = nil
	sub.Plan = newPlan
	return sub, nil
}

// AdminRenew extends a subscription by the specified number of months.
func (s *Service) AdminRenew(ctx context.Context, orgID int64, months int) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	start := sub.CurrentPeriodEnd
	now := time.Now()
	if now.After(start) {
		start = now
	}
	end := start.AddDate(0, months, 0)

	if err := s.repo.UpdateSubscriptionFields(ctx, sub.ID, map[string]interface{}{
		"status":               billing.SubscriptionStatusActive,
		"current_period_start": start,
		"current_period_end":   end,
		"frozen_at":            nil,
		"canceled_at":          nil,
		"cancel_at_period_end": false,
	}); err != nil {
		return nil, err
	}

	// Sync organization table
	s.syncOrganizationSubscription(ctx, orgID, nil, strPtr(billing.SubscriptionStatusActive))

	// Reload to get fresh data
	return s.GetSubscription(ctx, orgID)
}

// AdminCancelSubscription cancels a subscription without calling external payment APIs.
func (s *Service) AdminCancelSubscription(ctx context.Context, orgID int64) error {
	now := time.Now()

	if err := s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"status":      billing.SubscriptionStatusCanceled,
		"canceled_at": now,
	}); err != nil {
		return err
	}

	// Sync organization table
	s.syncOrganizationSubscription(ctx, orgID, nil, strPtr(billing.SubscriptionStatusCanceled))
	return nil
}

