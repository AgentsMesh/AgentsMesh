package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"gorm.io/gorm"
)

// Compile-time interface check.
var _ billing.BillingRepository = (*billingRepository)(nil)

type billingRepository struct {
	db *gorm.DB
}

// NewBillingRepository creates a new BillingRepository backed by GORM.
func NewBillingRepository(db *gorm.DB) billing.BillingRepository {
	return &billingRepository{db: db}
}

// Scoped returns a new repository scoped to the given raw DB handle.
func (r *billingRepository) Scoped(rawTx interface{}) billing.BillingRepository {
	if tx, ok := rawTx.(*gorm.DB); ok {
		return &billingRepository{db: tx}
	}
	return r
}

// --- Subscription ---

func (r *billingRepository) GetSubscriptionByOrgID(ctx context.Context, orgID int64) (*billing.Subscription, error) {
	var sub billing.Subscription
	if err := r.db.WithContext(ctx).Preload("Plan").Where("organization_id = ?", orgID).First(&sub).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *billingRepository) CreateSubscription(ctx context.Context, sub *billing.Subscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *billingRepository) SaveSubscription(ctx context.Context, sub *billing.Subscription) error {
	return r.db.WithContext(ctx).Save(sub).Error
}

func (r *billingRepository) UpdateSubscriptionFields(ctx context.Context, subID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&billing.Subscription{}).Where("id = ?", subID).Updates(updates).Error
}

func (r *billingRepository) UpdateSubscriptionFieldsByOrg(ctx context.Context, orgID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&billing.Subscription{}).Where("organization_id = ?", orgID).Updates(updates).Error
}

func (r *billingRepository) FindSubscriptionByProviderID(ctx context.Context, provider, subscriptionID string) (*billing.Subscription, error) {
	var sub billing.Subscription
	var err error
	switch provider {
	case billing.PaymentProviderLemonSqueezy:
		err = r.db.WithContext(ctx).Where("lemonsqueezy_subscription_id = ?", subscriptionID).First(&sub).Error
	default:
		err = r.db.WithContext(ctx).Where("stripe_subscription_id = ?", subscriptionID).First(&sub).Error
	}
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *billingRepository) FindSubscriptionByLSCustomerID(ctx context.Context, customerID string) (*billing.Subscription, error) {
	var sub billing.Subscription
	if err := r.db.WithContext(ctx).Where("lemonsqueezy_customer_id = ?", customerID).First(&sub).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *billingRepository) AddSeats(ctx context.Context, orgID int64, additionalSeats int) error {
	return r.db.WithContext(ctx).Model(&billing.Subscription{}).
		Where("organization_id = ?", orgID).
		Update("seat_count", gorm.Expr("seat_count + ?", additionalSeats)).Error
}

// --- Plan ---

func (r *billingRepository) GetPlanByName(ctx context.Context, name string) (*billing.SubscriptionPlan, error) {
	var plan billing.SubscriptionPlan
	if err := r.db.WithContext(ctx).Where("name = ? AND is_active = ?", name, true).First(&plan).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

func (r *billingRepository) GetPlanByID(ctx context.Context, id int64) (*billing.SubscriptionPlan, error) {
	var plan billing.SubscriptionPlan
	if err := r.db.WithContext(ctx).First(&plan, id).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

func (r *billingRepository) ListActivePlans(ctx context.Context) ([]*billing.SubscriptionPlan, error) {
	var plans []*billing.SubscriptionPlan
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("price_per_seat_monthly ASC").Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

// --- PlanPrice ---

func (r *billingRepository) GetPlanPrice(ctx context.Context, planID int64, currency string) (*billing.PlanPrice, error) {
	var price billing.PlanPrice
	if err := r.db.WithContext(ctx).Preload("Plan").
		Where("plan_id = ? AND currency = ?", planID, currency).
		First(&price).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &price, nil
}

func (r *billingRepository) ListPlanPrices(ctx context.Context, planID int64) ([]billing.PlanPrice, error) {
	var prices []billing.PlanPrice
	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Find(&prices).Error; err != nil {
		return nil, err
	}
	return prices, nil
}

func (r *billingRepository) FindPlanByVariantID(ctx context.Context, variantID string) (*billing.SubscriptionPlan, error) {
	var price billing.PlanPrice
	if err := r.db.WithContext(ctx).Preload("Plan").
		Where("lemonsqueezy_variant_id_monthly = ? OR lemonsqueezy_variant_id_yearly = ?", variantID, variantID).
		First(&price).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return price.Plan, nil
}

// --- Cross-domain Counts ---

func (r *billingRepository) CountOrgMembers(ctx context.Context, orgID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("organization_members").Where("organization_id = ?", orgID).Count(&count).Error
	return count, err
}

func (r *billingRepository) CountRunners(ctx context.Context, orgID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("runners").Where("organization_id = ?", orgID).Count(&count).Error
	return count, err
}

func (r *billingRepository) CountActivePods(ctx context.Context, orgID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("pods").
		Where("organization_id = ? AND status IN ?", orgID, agentpod.ActiveStatuses()).
		Count(&count).Error
	return count, err
}

func (r *billingRepository) CountRepositories(ctx context.Context, orgID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("repositories").Where("organization_id = ?", orgID).Count(&count).Error
	return count, err
}

func (r *billingRepository) CountPendingInvitations(ctx context.Context, orgID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("invitations").
		Where("organization_id = ? AND accepted_at IS NULL AND expires_at > ?", orgID, time.Now()).
		Count(&count).Error
	return count, err
}

// --- Organization Sync ---

func (r *billingRepository) SyncOrganizationSubscription(ctx context.Context, orgID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Table("organizations").Where("id = ?", orgID).Updates(updates).Error
}
