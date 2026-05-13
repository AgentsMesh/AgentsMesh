use agentsmesh_types::proto_billing_v1 as billing_proto;

fn opt_str(s: String) -> Option<String> {
    if s.is_empty() { None } else { Some(s) }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct SubscriptionPlanDto {
    pub id: i64,
    pub name: String,
    pub display_name: String,
    pub price_per_seat_monthly: f64,
    pub price_per_seat_yearly: f64,
    pub included_pod_minutes: i32,
    pub price_per_extra_minute: f64,
    pub max_users: i32,
    pub max_runners: i32,
    pub max_concurrent_pods: i32,
    pub max_repositories: i32,
    pub is_active: bool,
    pub created_at: Option<String>,
}

impl From<billing_proto::SubscriptionPlan> for SubscriptionPlanDto {
    fn from(p: billing_proto::SubscriptionPlan) -> Self {
        Self {
            id: p.id,
            name: p.name,
            display_name: p.display_name,
            price_per_seat_monthly: p.price_per_seat_monthly,
            price_per_seat_yearly: p.price_per_seat_yearly,
            included_pod_minutes: p.included_pod_minutes,
            price_per_extra_minute: p.price_per_extra_minute,
            max_users: p.max_users,
            max_runners: p.max_runners,
            max_concurrent_pods: p.max_concurrent_pods,
            max_repositories: p.max_repositories,
            is_active: p.is_active,
            created_at: opt_str(p.created_at),
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct SubscriptionDto {
    pub id: i64,
    pub plan_name: Option<String>,
    pub status: Option<String>,
    pub billing_cycle: Option<String>,
    pub current_period_start: Option<String>,
    pub current_period_end: Option<String>,
    pub auto_renew: bool,
    pub seat_count: i32,
    pub cancel_at_period_end: bool,
}

impl From<billing_proto::Subscription> for SubscriptionDto {
    fn from(s: billing_proto::Subscription) -> Self {
        Self {
            id: s.id,
            plan_name: s.plan.as_ref().map(|p| p.name.clone()),
            status: opt_str(s.status),
            billing_cycle: opt_str(s.billing_cycle),
            current_period_start: opt_str(s.current_period_start),
            current_period_end: opt_str(s.current_period_end),
            auto_renew: s.auto_renew,
            seat_count: s.seat_count,
            cancel_at_period_end: s.cancel_at_period_end,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct UsageOverviewDto {
    pub pod_minutes: f64,
    pub included_pod_minutes: f64,
    pub users: i32,
    pub max_users: i32,
    pub runners: i32,
    pub max_runners: i32,
    pub concurrent_pods: i32,
    pub max_concurrent_pods: i32,
    pub repositories: i32,
    pub max_repositories: i32,
}

impl From<billing_proto::UsageOverview> for UsageOverviewDto {
    fn from(u: billing_proto::UsageOverview) -> Self {
        Self {
            pod_minutes: u.pod_minutes,
            included_pod_minutes: u.included_pod_minutes,
            users: u.users,
            max_users: u.max_users,
            runners: u.runners,
            max_runners: u.max_runners,
            concurrent_pods: u.concurrent_pods,
            max_concurrent_pods: u.max_concurrent_pods,
            repositories: u.repositories,
            max_repositories: u.max_repositories,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct BillingOverviewDto {
    pub plan: Option<SubscriptionPlanDto>,
    pub status: Option<String>,
    pub billing_cycle: Option<String>,
    pub current_period_start: Option<String>,
    pub current_period_end: Option<String>,
    pub cancel_at_period_end: bool,
    pub usage: Option<UsageOverviewDto>,
}

impl From<billing_proto::BillingOverview> for BillingOverviewDto {
    fn from(o: billing_proto::BillingOverview) -> Self {
        Self {
            plan: o.plan.map(SubscriptionPlanDto::from),
            status: opt_str(o.status),
            billing_cycle: opt_str(o.billing_cycle),
            current_period_start: opt_str(o.current_period_start),
            current_period_end: opt_str(o.current_period_end),
            cancel_at_period_end: o.cancel_at_period_end,
            usage: o.usage.map(UsageOverviewDto::from),
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PlanListResponseDto {
    pub plans: Vec<SubscriptionPlanDto>,
}

pub(crate) fn plan_list_from_proto(
    resp: billing_proto::ListPlansResponse,
) -> PlanListResponseDto {
    PlanListResponseDto {
        plans: resp.items.into_iter().map(SubscriptionPlanDto::from).collect(),
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct InvoiceDto {
    pub id: i64,
    pub invoice_no: String,
    pub status: Option<String>,
    pub currency: Option<String>,
    pub subtotal: f64,
    pub tax_amount: f64,
    pub total: f64,
    pub period_start: Option<String>,
    pub period_end: Option<String>,
    pub pdf_url: Option<String>,
    pub issued_at: Option<String>,
    pub due_at: Option<String>,
    pub paid_at: Option<String>,
}

impl From<billing_proto::Invoice> for InvoiceDto {
    fn from(i: billing_proto::Invoice) -> Self {
        Self {
            id: i.id,
            invoice_no: i.invoice_no,
            status: opt_str(i.status),
            currency: opt_str(i.currency),
            subtotal: i.subtotal,
            tax_amount: i.tax_amount,
            total: i.total,
            period_start: opt_str(i.period_start),
            period_end: opt_str(i.period_end),
            pdf_url: i.pdf_url,
            issued_at: i.issued_at,
            due_at: i.due_at,
            paid_at: i.paid_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct InvoiceListResponseDto {
    pub invoices: Vec<InvoiceDto>,
    pub total: i64,
}

pub(crate) fn invoice_list_from_proto(
    resp: billing_proto::ListInvoicesResponse,
) -> InvoiceListResponseDto {
    InvoiceListResponseDto {
        invoices: resp.items.into_iter().map(InvoiceDto::from).collect(),
        total: resp.total,
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct SeatUsageDto {
    pub total_seats: i32,
    pub used_seats: i32,
    pub available_seats: i32,
    pub max_seats: i32,
    pub can_add_seats: bool,
}

impl From<billing_proto::SeatUsage> for SeatUsageDto {
    fn from(s: billing_proto::SeatUsage) -> Self {
        Self {
            total_seats: s.total_seats,
            used_seats: s.used_seats,
            available_seats: s.available_seats,
            max_seats: s.max_seats,
            can_add_seats: s.can_add_seats,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct DeploymentInfoDto {
    pub deployment_type: String,
    pub available_providers: Vec<String>,
}

impl From<billing_proto::DeploymentInfo> for DeploymentInfoDto {
    fn from(d: billing_proto::DeploymentInfo) -> Self {
        Self {
            deployment_type: d.deployment_type,
            available_providers: d.available_providers,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CheckoutStatusDto {
    pub order_no: String,
    pub status: Option<String>,
    pub order_type: Option<String>,
    pub amount: f64,
    pub currency: Option<String>,
    pub created_at: Option<String>,
    pub paid_at: Option<String>,
}

impl From<billing_proto::CheckoutStatus> for CheckoutStatusDto {
    fn from(s: billing_proto::CheckoutStatus) -> Self {
        Self {
            order_no: s.order_no,
            status: opt_str(s.status),
            order_type: opt_str(s.order_type),
            amount: s.amount,
            currency: opt_str(s.currency),
            created_at: opt_str(s.created_at),
            paid_at: s.paid_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateCheckoutResponseDto {
    pub order_no: String,
    pub session_id: Option<String>,
    pub session_url: Option<String>,
    pub qr_code_url: Option<String>,
    pub expires_at: Option<String>,
    pub provider: Option<String>,
}

impl From<billing_proto::CreateCheckoutResponse> for CreateCheckoutResponseDto {
    fn from(r: billing_proto::CreateCheckoutResponse) -> Self {
        Self {
            order_no: r.order_no,
            session_id: opt_str(r.session_id),
            session_url: opt_str(r.session_url),
            qr_code_url: r.qr_code_url,
            expires_at: opt_str(r.expires_at),
            provider: opt_str(r.provider),
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateSubscriptionRequestDto {
    pub plan_name: String,
    pub billing_cycle: Option<String>,
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct UpdateSubscriptionRequestDto {
    pub plan_name: String,
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct UpgradeSubscriptionRequestDto {
    pub plan_name: String,
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct ChangeBillingCycleRequestDto {
    pub billing_cycle: String,
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateCheckoutRequestDto {
    pub order_type: String,
    pub plan_name: Option<String>,
    pub billing_cycle: Option<String>,
    pub seats: Option<i32>,
    pub provider: Option<String>,
    pub success_url: String,
    pub cancel_url: String,
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PublicPlanPricingDto {
    pub name: String,
    pub display_name: String,
    pub price_monthly: f64,
    pub price_yearly: f64,
    pub max_users: i32,
    pub max_runners: i32,
    pub max_repositories: i32,
    pub max_concurrent_pods: i32,
}

impl From<billing_proto::PublicPlanPricing> for PublicPlanPricingDto {
    fn from(p: billing_proto::PublicPlanPricing) -> Self {
        Self {
            name: p.name,
            display_name: p.display_name,
            price_monthly: p.price_monthly,
            price_yearly: p.price_yearly,
            max_users: p.max_users,
            max_runners: p.max_runners,
            max_repositories: p.max_repositories,
            max_concurrent_pods: p.max_concurrent_pods,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PublicPricingResponseDto {
    pub deployment_type: String,
    pub currency: String,
    pub plans: Vec<PublicPlanPricingDto>,
}

impl From<billing_proto::PublicPricingResponse> for PublicPricingResponseDto {
    fn from(r: billing_proto::PublicPricingResponse) -> Self {
        Self {
            deployment_type: r.deployment_type,
            currency: r.currency,
            plans: r.plans.into_iter().map(PublicPlanPricingDto::from).collect(),
        }
    }
}
