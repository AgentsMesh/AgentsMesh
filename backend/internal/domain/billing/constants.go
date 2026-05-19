package billing

const (
	PlanBased      = "based" // Entry-level paid plan
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
	PlanOnPremise  = "onpremise"
)

const (
	CurrencyUSD = "USD"
	CurrencyCNY = "CNY"
)

const DefaultTrialDays = 30

const (
	SubscriptionStatusActive   = "active"
	SubscriptionStatusPastDue  = "past_due"
	SubscriptionStatusCanceled = "canceled"
	SubscriptionStatusTrialing = "trialing"
	SubscriptionStatusFrozen   = "frozen"
	SubscriptionStatusPaused   = "paused"
	SubscriptionStatusExpired  = "expired"
)

const (
	PaymentProviderStripe       = "stripe"
	PaymentProviderLemonSqueezy = "lemonsqueezy"
	PaymentProviderAlipay       = "alipay"
	PaymentProviderWeChat       = "wechat"
	PaymentProviderLicense      = "license"
)

const (
	PaymentMethodCard            = "card"
	PaymentMethodAlipayQR        = "alipay_qr"
	PaymentMethodAlipayAgreement = "alipay_agreement"
	PaymentMethodWeChatNative    = "wechat_native"
	PaymentMethodWeChatContract  = "wechat_contract"
)

const (
	BillingCycleMonthly = "monthly"
	BillingCycleYearly  = "yearly"
)

const (
	UsageTypePodMinutes  = "pod_minutes"
	UsageTypeStorageGB   = "storage_gb"
	UsageTypeAPIRequests = "api_requests"
)

const (
	OrderTypeSubscription = "subscription"
	OrderTypeSeatPurchase = "seat_purchase"
	OrderTypePlanUpgrade  = "plan_upgrade"
	OrderTypeRenewal      = "renewal"
)

const (
	OrderStatusPending    = "pending"
	OrderStatusProcessing = "processing"
	OrderStatusSucceeded  = "succeeded"
	OrderStatusFailed     = "failed"
	OrderStatusCanceled   = "canceled"
	OrderStatusRefunded   = "refunded"
)

const (
	TransactionTypePayment    = "payment"
	TransactionTypeRefund     = "refund"
	TransactionTypeChargeback = "chargeback"
)

const (
	TransactionStatusPending   = "pending"
	TransactionStatusSucceeded = "succeeded"
	TransactionStatusFailed    = "failed"
)

const (
	InvoiceStatusDraft  = "draft"
	InvoiceStatusIssued = "issued"
	InvoiceStatusPaid   = "paid"
	InvoiceStatusVoid   = "void"
)

const (
	WebhookEventCheckoutCompleted = "checkout.session.completed"

	WebhookEventInvoicePaid   = "invoice.paid"
	WebhookEventInvoiceFailed = "invoice.payment_failed"

	WebhookEventSubscriptionDeleted = "customer.subscription.deleted"
	WebhookEventSubscriptionUpdated = "customer.subscription.updated"
)
