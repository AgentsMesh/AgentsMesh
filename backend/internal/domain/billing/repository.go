package billing

import (
	"context"
	"time"
)

type BillingRepository interface {
	GetSubscriptionByOrgID(ctx context.Context, orgID int64) (*Subscription, error)
	CreateSubscription(ctx context.Context, sub *Subscription) error
	SaveSubscription(ctx context.Context, sub *Subscription) error
	UpdateSubscriptionFields(ctx context.Context, subID int64, updates map[string]interface{}) error
	UpdateSubscriptionFieldsByOrg(ctx context.Context, orgID int64, updates map[string]interface{}) error
	FindSubscriptionByProviderID(ctx context.Context, provider, subscriptionID string) (*Subscription, error)
	FindSubscriptionByLSCustomerID(ctx context.Context, customerID string) (*Subscription, error)
	AddSeats(ctx context.Context, orgID int64, additionalSeats int) error

	GetPlanByName(ctx context.Context, name string) (*SubscriptionPlan, error)
	GetPlanByID(ctx context.Context, id int64) (*SubscriptionPlan, error)
	ListActivePlans(ctx context.Context) ([]*SubscriptionPlan, error)

	GetPlanPrice(ctx context.Context, planID int64, currency string) (*PlanPrice, error)
	ListPlanPrices(ctx context.Context, planID int64) ([]PlanPrice, error)
	FindPlanByVariantID(ctx context.Context, variantID string) (*SubscriptionPlan, error)

	CreatePaymentOrder(ctx context.Context, order *PaymentOrder) error
	GetPaymentOrderByNo(ctx context.Context, orderNo string) (*PaymentOrder, error)
	GetPaymentOrderByExternalNo(ctx context.Context, externalNo string) (*PaymentOrder, error)
	UpdatePaymentOrderStatus(ctx context.Context, orderNo string, updates map[string]interface{}) error

	CreatePaymentTransaction(ctx context.Context, tx *PaymentTransaction) error

	CreateInvoice(ctx context.Context, invoice *Invoice) error
	ListInvoicesByOrg(ctx context.Context, orgID int64, limit, offset int) ([]*Invoice, error)

	CreateUsageRecord(ctx context.Context, record *UsageRecord) error
	SumUsageByPeriod(ctx context.Context, orgID int64, usageType string, periodStart, periodEnd time.Time) (float64, error)
	ListUsageHistory(ctx context.Context, orgID int64, usageType string, since time.Time) ([]*UsageRecord, error)

	CreateWebhookEvent(ctx context.Context, event *WebhookEvent) error
	DeleteWebhookEvent(ctx context.Context, eventID, provider string) error

	CountOrgMembers(ctx context.Context, orgID int64) (int64, error)
	CountRunners(ctx context.Context, orgID int64) (int64, error)
	CountActivePods(ctx context.Context, orgID int64) (int64, error)
	CountRepositories(ctx context.Context, orgID int64) (int64, error)
	CountPendingInvitations(ctx context.Context, orgID int64) (int64, error)

	SyncOrganizationSubscription(ctx context.Context, orgID int64, updates map[string]interface{}) error

	Scoped(rawTx interface{}) BillingRepository
}
