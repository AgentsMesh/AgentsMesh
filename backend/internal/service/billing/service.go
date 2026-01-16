package billing

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/subscription"
	"gorm.io/gorm"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrPlanNotFound         = errors.New("plan not found")
	ErrQuotaExceeded        = errors.New("quota exceeded")
	ErrInvalidPlan          = errors.New("invalid plan")
)

// Service handles billing operations
type Service struct {
	db             *gorm.DB
	stripeEnabled  bool
}

// NewService creates a new billing service
func NewService(db *gorm.DB, stripeKey string) *Service {
	if stripeKey != "" {
		stripe.Key = stripeKey
	}
	return &Service{
		db:            db,
		stripeEnabled: stripeKey != "",
	}
}

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

// GetSubscription returns subscription for an organization
func (s *Service) GetSubscription(ctx context.Context, orgID int64) (*billing.Subscription, error) {
	var sub billing.Subscription
	if err := s.db.WithContext(ctx).Preload("Plan").Where("organization_id = ?", orgID).First(&sub).Error; err != nil {
		return nil, ErrSubscriptionNotFound
	}
	return &sub, nil
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

	if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
		return nil, err
	}

	sub.Plan = plan
	return sub, nil
}

// UpdateSubscription updates subscription plan
func (s *Service) UpdateSubscription(ctx context.Context, orgID int64, planName string) (*billing.Subscription, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, err
	}

	sub.PlanID = plan.ID

	// Update Stripe subscription if enabled
	if s.stripeEnabled && sub.StripeSubscriptionID != nil {
		// In a real implementation, update Stripe subscription here
	}

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return nil, err
	}

	sub.Plan = plan
	return sub, nil
}

// CancelSubscription cancels a subscription
func (s *Service) CancelSubscription(ctx context.Context, orgID int64) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	sub.Status = billing.SubscriptionStatusCanceled

	// Cancel Stripe subscription if enabled
	if s.stripeEnabled && sub.StripeSubscriptionID != nil {
		_, err := subscription.Cancel(*sub.StripeSubscriptionID, nil)
		if err != nil {
			return err
		}
	}

	return s.db.WithContext(ctx).Save(sub).Error
}

// CreateStripeCustomer creates a Stripe customer for an organization
func (s *Service) CreateStripeCustomer(ctx context.Context, orgID int64, email, name string) (string, error) {
	if !s.stripeEnabled {
		return "", nil
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Metadata: map[string]string{
			"organization_id": string(rune(orgID)),
		},
	}

	c, err := customer.New(params)
	if err != nil {
		return "", err
	}

	// Update subscription with Stripe customer ID
	s.db.WithContext(ctx).Model(&billing.Subscription{}).
		Where("organization_id = ?", orgID).
		Update("stripe_customer_id", c.ID)

	return c.ID, nil
}

// RecordUsage records usage for an organization
func (s *Service) RecordUsage(ctx context.Context, orgID int64, usageType string, quantity float64, metadata billing.UsageMetadata) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	record := &billing.UsageRecord{
		OrganizationID: orgID,
		UsageType:      usageType,
		Quantity:       quantity,
		PeriodStart:    sub.CurrentPeriodStart,
		PeriodEnd:      sub.CurrentPeriodEnd,
		Metadata:       metadata,
	}

	return s.db.WithContext(ctx).Create(record).Error
}

// GetUsage returns usage for an organization in current period
func (s *Service) GetUsage(ctx context.Context, orgID int64, usageType string) (float64, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return 0, err
	}

	var total float64
	if err := s.db.WithContext(ctx).Model(&billing.UsageRecord{}).
		Where("organization_id = ? AND usage_type = ? AND period_start >= ? AND period_end <= ?",
			orgID, usageType, sub.CurrentPeriodStart, sub.CurrentPeriodEnd).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

// GetUsageHistory returns usage history for an organization
func (s *Service) GetUsageHistory(ctx context.Context, orgID int64, usageType string, months int) ([]*billing.UsageRecord, error) {
	since := time.Now().AddDate(0, -months, 0)

	var records []*billing.UsageRecord
	query := s.db.WithContext(ctx).Where("organization_id = ? AND period_start >= ?", orgID, since)

	if usageType != "" {
		query = query.Where("usage_type = ?", usageType)
	}

	if err := query.Order("period_start DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

// CheckQuota checks if organization has quota available
func (s *Service) CheckQuota(ctx context.Context, orgID int64, resource string, requestedAmount int) error {
	sub, err := s.GetSubscription(ctx, orgID)

	var plan *billing.SubscriptionPlan
	if err != nil {
		// No subscription found, use Free plan as default
		if errors.Is(err, ErrSubscriptionNotFound) {
			plan, _ = s.GetPlan(ctx, "free")
			if plan == nil {
				// No Free plan in database, allow by default
				return nil
			}
		} else {
			return err
		}
	} else {
		plan = sub.Plan
		if plan == nil {
			plan, _ = s.GetPlanByID(ctx, sub.PlanID)
		}
	}

	if plan == nil {
		return ErrPlanNotFound
	}

	// Check custom quotas first (only if subscription exists)
	if sub != nil && sub.CustomQuotas != nil {
		if customLimit, ok := sub.CustomQuotas[resource]; ok {
			if limit, ok := customLimit.(float64); ok && int(limit) != -1 {
				current, _ := s.getCurrentResourceCount(ctx, orgID, resource)
				if current+requestedAmount > int(limit) {
					return ErrQuotaExceeded
				}
				return nil
			}
		}
	}

	// Check plan limits
	var limit int
	switch resource {
	case "users":
		limit = plan.MaxUsers
	case "runners":
		limit = plan.MaxRunners
	case "concurrent_pods":
		limit = plan.MaxConcurrentPods
	case "repositories":
		limit = plan.MaxRepositories
	case "pod_minutes":
		limit = plan.IncludedPodMinutes
	default:
		return nil
	}

	// -1 means unlimited
	if limit == -1 {
		return nil
	}

	current, _ := s.getCurrentResourceCount(ctx, orgID, resource)
	if current+requestedAmount > limit {
		return ErrQuotaExceeded
	}

	return nil
}

func (s *Service) getCurrentResourceCount(ctx context.Context, orgID int64, resource string) (int, error) {
	var count int64

	switch resource {
	case "users":
		s.db.WithContext(ctx).Table("organization_members").Where("organization_id = ?", orgID).Count(&count)
	case "runners":
		s.db.WithContext(ctx).Table("runners").Where("organization_id = ?", orgID).Count(&count)
	case "concurrent_pods":
		// Count active pods (running or initializing)
		s.db.WithContext(ctx).Table("pods").
			Where("organization_id = ? AND status IN ?", orgID, []string{"running", "initializing"}).
			Count(&count)
	case "repositories":
		s.db.WithContext(ctx).Table("repositories").Where("organization_id = ?", orgID).Count(&count)
	case "pod_minutes":
		usage, _ := s.GetUsage(ctx, orgID, billing.UsageTypePodMinutes)
		return int(usage), nil
	}

	return int(count), nil
}

// GetCurrentConcurrentPods returns the number of currently active pods for an organization
func (s *Service) GetCurrentConcurrentPods(ctx context.Context, orgID int64) (int, error) {
	return s.getCurrentResourceCount(ctx, orgID, "concurrent_pods")
}

// GetPlanByID returns a plan by ID
func (s *Service) GetPlanByID(ctx context.Context, planID int64) (*billing.SubscriptionPlan, error) {
	var plan billing.SubscriptionPlan
	if err := s.db.WithContext(ctx).First(&plan, planID).Error; err != nil {
		return nil, ErrPlanNotFound
	}
	return &plan, nil
}

// RenewSubscription renews subscription for next period
func (s *Service) RenewSubscription(ctx context.Context, orgID int64) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	// Set new period
	sub.CurrentPeriodStart = sub.CurrentPeriodEnd
	if sub.BillingCycle == billing.BillingCycleYearly {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(1, 0, 0)
	} else {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(0, 1, 0)
	}

	return s.db.WithContext(ctx).Save(sub).Error
}

// SetCustomQuota sets a custom quota for an organization
func (s *Service) SetCustomQuota(ctx context.Context, orgID int64, resource string, limit int) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	if sub.CustomQuotas == nil {
		sub.CustomQuotas = make(billing.CustomQuotas)
	}

	sub.CustomQuotas[resource] = limit

	return s.db.WithContext(ctx).Save(sub).Error
}

// GetBillingOverview returns billing overview for an organization
func (s *Service) GetBillingOverview(ctx context.Context, orgID int64) (*BillingOverview, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	plan := sub.Plan
	if plan == nil {
		plan, _ = s.GetPlanByID(ctx, sub.PlanID)
	}

	// Get current usage
	podMinutes, _ := s.GetUsage(ctx, orgID, billing.UsageTypePodMinutes)

	// Count resources
	var userCount, runnerCount, repoCount, concurrentPodCount int64
	s.db.WithContext(ctx).Table("organization_members").Where("organization_id = ?", orgID).Count(&userCount)
	s.db.WithContext(ctx).Table("runners").Where("organization_id = ?", orgID).Count(&runnerCount)
	s.db.WithContext(ctx).Table("repositories").Where("organization_id = ?", orgID).Count(&repoCount)
	s.db.WithContext(ctx).Table("pods").
		Where("organization_id = ? AND status IN ?", orgID, []string{"running", "initializing"}).
		Count(&concurrentPodCount)

	return &BillingOverview{
		Plan:              plan,
		Status:            sub.Status,
		BillingCycle:      sub.BillingCycle,
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		Usage: UsageOverview{
			PodMinutes:         podMinutes,
			IncludedPodMinutes: float64(plan.IncludedPodMinutes),
			Users:              int(userCount),
			MaxUsers:           plan.MaxUsers,
			Runners:            int(runnerCount),
			MaxRunners:         plan.MaxRunners,
			ConcurrentPods:     int(concurrentPodCount),
			MaxConcurrentPods:  plan.MaxConcurrentPods,
			Repositories:       int(repoCount),
			MaxRepositories:    plan.MaxRepositories,
		},
	}, nil
}

// BillingOverview represents billing overview
type BillingOverview struct {
	Plan               *billing.SubscriptionPlan `json:"plan"`
	Status             string                    `json:"status"`
	BillingCycle       string                    `json:"billing_cycle"`
	CurrentPeriodStart time.Time                 `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                 `json:"current_period_end"`
	Usage              UsageOverview             `json:"usage"`
}

// UsageOverview represents usage overview
type UsageOverview struct {
	PodMinutes         float64 `json:"pod_minutes"`
	IncludedPodMinutes float64 `json:"included_pod_minutes"`
	Users              int     `json:"users"`
	MaxUsers           int     `json:"max_users"`
	Runners            int     `json:"runners"`
	MaxRunners         int     `json:"max_runners"`
	ConcurrentPods     int     `json:"concurrent_pods"`
	MaxConcurrentPods  int     `json:"max_concurrent_pods"`
	Repositories       int     `json:"repositories"`
	MaxRepositories    int     `json:"max_repositories"`
}
