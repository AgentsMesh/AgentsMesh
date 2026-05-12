// Hand-maintained `prost::Message` mirrors of `proto/billing/v1/billing.proto`.
// Tag numbers match the .proto byte-for-byte; `tools/validate_prost_tags`
// runs at build time to catch drift (watch list §8). NO `Serialize` /
// `Deserialize` derives on these structs — binary wire only (conventions
// §2.5, §3).
//
// PR #334 lesson: every UI field (especially the public-pricing card's
// `price_monthly`, `price_yearly`, currency, max_*) must have a numbered
// prost tag matching the .proto. A drifted tag is silently dropped on the
// wire; the round-trip test at the bottom of this file proves byte-for-byte
// equality after encode/decode for each message.

// -------- Entities --------

#[derive(Clone, PartialEq, prost::Message)]
pub struct SubscriptionPlan {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub name: String,
    #[prost(string, tag = "3")]
    pub display_name: String,
    #[prost(double, tag = "4")]
    pub price_per_seat_monthly: f64,
    #[prost(double, tag = "5")]
    pub price_per_seat_yearly: f64,
    #[prost(int32, tag = "6")]
    pub included_pod_minutes: i32,
    #[prost(double, tag = "7")]
    pub price_per_extra_minute: f64,
    #[prost(int32, tag = "8")]
    pub max_users: i32,
    #[prost(int32, tag = "9")]
    pub max_runners: i32,
    #[prost(int32, tag = "10")]
    pub max_concurrent_pods: i32,
    #[prost(int32, tag = "11")]
    pub max_repositories: i32,
    #[prost(bool, tag = "12")]
    pub is_active: bool,
    #[prost(string, tag = "13")]
    pub created_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct Subscription {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub organization_id: i64,
    #[prost(int64, tag = "3")]
    pub plan_id: i64,
    #[prost(string, tag = "4")]
    pub status: String,
    #[prost(string, tag = "5")]
    pub billing_cycle: String,
    #[prost(string, tag = "6")]
    pub current_period_start: String,
    #[prost(string, tag = "7")]
    pub current_period_end: String,
    #[prost(message, optional, tag = "8")]
    pub plan: Option<SubscriptionPlan>,
    #[prost(string, optional, tag = "9")]
    pub payment_provider: Option<String>,
    #[prost(string, optional, tag = "10")]
    pub payment_method: Option<String>,
    #[prost(bool, tag = "11")]
    pub auto_renew: bool,
    #[prost(int32, tag = "12")]
    pub seat_count: i32,
    #[prost(string, optional, tag = "13")]
    pub stripe_customer_id: Option<String>,
    #[prost(string, optional, tag = "14")]
    pub stripe_subscription_id: Option<String>,
    #[prost(string, optional, tag = "15")]
    pub lemonsqueezy_customer_id: Option<String>,
    #[prost(string, optional, tag = "16")]
    pub lemonsqueezy_subscription_id: Option<String>,
    #[prost(string, optional, tag = "17")]
    pub canceled_at: Option<String>,
    #[prost(bool, tag = "18")]
    pub cancel_at_period_end: bool,
    #[prost(string, optional, tag = "19")]
    pub frozen_at: Option<String>,
    #[prost(string, optional, tag = "20")]
    pub downgrade_to_plan: Option<String>,
    #[prost(string, optional, tag = "21")]
    pub next_billing_cycle: Option<String>,
    #[prost(string, tag = "22")]
    pub created_at: String,
    #[prost(string, tag = "23")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UsageOverview {
    #[prost(double, tag = "1")]
    pub pod_minutes: f64,
    #[prost(double, tag = "2")]
    pub included_pod_minutes: f64,
    #[prost(int32, tag = "3")]
    pub users: i32,
    #[prost(int32, tag = "4")]
    pub max_users: i32,
    #[prost(int32, tag = "5")]
    pub runners: i32,
    #[prost(int32, tag = "6")]
    pub max_runners: i32,
    #[prost(int32, tag = "7")]
    pub concurrent_pods: i32,
    #[prost(int32, tag = "8")]
    pub max_concurrent_pods: i32,
    #[prost(int32, tag = "9")]
    pub repositories: i32,
    #[prost(int32, tag = "10")]
    pub max_repositories: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct BillingOverview {
    #[prost(message, optional, tag = "1")]
    pub plan: Option<SubscriptionPlan>,
    #[prost(string, tag = "2")]
    pub status: String,
    #[prost(string, tag = "3")]
    pub billing_cycle: String,
    #[prost(string, tag = "4")]
    pub current_period_start: String,
    #[prost(string, tag = "5")]
    pub current_period_end: String,
    #[prost(bool, tag = "6")]
    pub cancel_at_period_end: bool,
    #[prost(message, optional, tag = "7")]
    pub usage: Option<UsageOverview>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct Invoice {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub organization_id: i64,
    #[prost(int64, optional, tag = "3")]
    pub payment_order_id: Option<i64>,
    #[prost(string, tag = "4")]
    pub invoice_no: String,
    #[prost(string, tag = "5")]
    pub status: String,
    #[prost(string, tag = "6")]
    pub currency: String,
    #[prost(double, tag = "7")]
    pub subtotal: f64,
    #[prost(double, tag = "8")]
    pub tax_amount: f64,
    #[prost(double, tag = "9")]
    pub total: f64,
    #[prost(string, tag = "10")]
    pub period_start: String,
    #[prost(string, tag = "11")]
    pub period_end: String,
    #[prost(string, optional, tag = "12")]
    pub pdf_url: Option<String>,
    #[prost(string, optional, tag = "13")]
    pub issued_at: Option<String>,
    #[prost(string, optional, tag = "14")]
    pub due_at: Option<String>,
    #[prost(string, optional, tag = "15")]
    pub paid_at: Option<String>,
    #[prost(string, tag = "16")]
    pub created_at: String,
    #[prost(string, tag = "17")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct SeatUsage {
    #[prost(int32, tag = "1")]
    pub total_seats: i32,
    #[prost(int32, tag = "2")]
    pub used_seats: i32,
    #[prost(int32, tag = "3")]
    pub available_seats: i32,
    #[prost(int32, tag = "4")]
    pub max_seats: i32,
    #[prost(bool, tag = "5")]
    pub can_add_seats: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DeploymentInfo {
    #[prost(string, tag = "1")]
    pub deployment_type: String,
    #[prost(string, repeated, tag = "2")]
    pub available_providers: Vec<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CheckoutStatus {
    #[prost(string, tag = "1")]
    pub order_no: String,
    #[prost(string, tag = "2")]
    pub status: String,
    #[prost(string, tag = "3")]
    pub order_type: String,
    #[prost(double, tag = "4")]
    pub amount: f64,
    #[prost(string, tag = "5")]
    pub currency: String,
    #[prost(string, tag = "6")]
    pub created_at: String,
    #[prost(string, optional, tag = "7")]
    pub paid_at: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct PublicPlanPricing {
    #[prost(string, tag = "1")]
    pub name: String,
    #[prost(string, tag = "2")]
    pub display_name: String,
    #[prost(double, tag = "3")]
    pub price_monthly: f64,
    #[prost(double, tag = "4")]
    pub price_yearly: f64,
    #[prost(int32, tag = "5")]
    pub max_users: i32,
    #[prost(int32, tag = "6")]
    pub max_runners: i32,
    #[prost(int32, tag = "7")]
    pub max_repositories: i32,
    #[prost(int32, tag = "8")]
    pub max_concurrent_pods: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct PublicPricingResponse {
    #[prost(string, tag = "1")]
    pub deployment_type: String,
    #[prost(string, tag = "2")]
    pub currency: String,
    #[prost(message, repeated, tag = "3")]
    pub plans: Vec<PublicPlanPricing>,
}

// -------- Requests / Responses --------

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetOverviewRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListPlansRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListPlansResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<SubscriptionPlan>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetSubscriptionRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateSubscriptionRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub plan_name: String,
    #[prost(string, optional, tag = "3")]
    pub billing_cycle: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateSubscriptionRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub plan_name: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CancelSubscriptionRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CancelSubscriptionResponse {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RequestCancelSubscriptionRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(bool, tag = "2")]
    pub immediate: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RequestCancelSubscriptionResponse {
    #[prost(string, optional, tag = "1")]
    pub current_period_end: Option<String>,
    #[prost(bool, tag = "2")]
    pub immediate: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ReactivateSubscriptionRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpgradeSubscriptionRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub plan_name: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChangeBillingCycleRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub billing_cycle: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChangeBillingCycleResponse {
    #[prost(string, tag = "1")]
    pub current_cycle: String,
    #[prost(string, tag = "2")]
    pub next_cycle: String,
    #[prost(string, tag = "3")]
    pub effective_date: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateAutoRenewRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(bool, tag = "2")]
    pub auto_renew: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetSeatUsageRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct PurchaseSeatsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int32, tag = "2")]
    pub seats: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct PurchaseSeatsResponse {
    #[prost(message, optional, tag = "1")]
    pub seats: Option<SeatUsage>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListInvoicesRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int32, optional, tag = "2")]
    pub offset: Option<i32>,
    #[prost(int32, optional, tag = "3")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListInvoicesResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<Invoice>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateCheckoutRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub order_type: String,
    #[prost(string, optional, tag = "3")]
    pub plan_name: Option<String>,
    #[prost(string, optional, tag = "4")]
    pub billing_cycle: Option<String>,
    #[prost(int32, optional, tag = "5")]
    pub seats: Option<i32>,
    #[prost(string, optional, tag = "6")]
    pub provider: Option<String>,
    #[prost(string, tag = "7")]
    pub success_url: String,
    #[prost(string, tag = "8")]
    pub cancel_url: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateCheckoutResponse {
    #[prost(string, tag = "1")]
    pub order_no: String,
    #[prost(string, tag = "2")]
    pub session_id: String,
    #[prost(string, tag = "3")]
    pub session_url: String,
    #[prost(string, optional, tag = "4")]
    pub qr_code_url: Option<String>,
    #[prost(string, tag = "5")]
    pub expires_at: String,
    #[prost(string, tag = "6")]
    pub provider: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetCheckoutStatusRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub order_no: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetDeploymentInfoRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetPublicPricingRequest {
    #[prost(string, optional, tag = "1")]
    pub currency: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetPublicDeploymentInfoRequest {}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    fn sample_plan() -> SubscriptionPlan {
        SubscriptionPlan {
            id: 7,
            name: "based".into(),
            display_name: "Based".into(),
            price_per_seat_monthly: 9.9,
            price_per_seat_yearly: 99.0,
            included_pod_minutes: 100,
            price_per_extra_minute: 0.05,
            max_users: 1,
            max_runners: 1,
            max_concurrent_pods: 5,
            max_repositories: 5,
            is_active: true,
            created_at: "2026-05-01T00:00:00Z".into(),
        }
    }

    #[test]
    fn plan_round_trip_preserves_every_field() {
        let original = sample_plan();
        let bytes = original.encode_to_vec();
        let decoded = SubscriptionPlan::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn public_pricing_round_trip_preserves_pr_334_fields() {
        // PR #334 regression: PublicPlanPricing dropped `price_monthly`,
        // `price_yearly`, `currency`. Pin them with non-default values so a
        // transposed tag pair surfaces as a field-value swap.
        let original = PublicPricingResponse {
            deployment_type: "global".into(),
            currency: "USD".into(),
            plans: vec![PublicPlanPricing {
                name: "based".into(),
                display_name: "Based".into(),
                price_monthly: 9.9,
                price_yearly: 99.0,
                max_users: 1,
                max_runners: 1,
                max_repositories: 5,
                max_concurrent_pods: 5,
            }],
        };
        let bytes = original.encode_to_vec();
        let decoded = PublicPricingResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.plans.len(), 1);
        assert_eq!(decoded.plans[0].price_monthly, 9.9);
        assert_eq!(decoded.plans[0].price_yearly, 99.0);
        assert_eq!(decoded.currency, "USD");
    }

    #[test]
    fn subscription_round_trip_preserves_provider_ids() {
        let original = Subscription {
            id: 1,
            organization_id: 42,
            plan_id: 7,
            status: "active".into(),
            billing_cycle: "monthly".into(),
            current_period_start: "2026-05-01T00:00:00Z".into(),
            current_period_end: "2026-06-01T00:00:00Z".into(),
            plan: Some(sample_plan()),
            payment_provider: Some("stripe".into()),
            payment_method: Some("card".into()),
            auto_renew: true,
            seat_count: 3,
            stripe_customer_id: Some("cus_123".into()),
            stripe_subscription_id: Some("sub_456".into()),
            lemonsqueezy_customer_id: None,
            lemonsqueezy_subscription_id: None,
            canceled_at: None,
            cancel_at_period_end: false,
            frozen_at: None,
            downgrade_to_plan: None,
            next_billing_cycle: None,
            created_at: "2026-05-01T00:00:00Z".into(),
            updated_at: "2026-05-10T00:00:00Z".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = Subscription::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.stripe_customer_id.as_deref(), Some("cus_123"));
        assert_eq!(decoded.seat_count, 3);
    }

    #[test]
    fn list_response_round_trip_preserves_envelope() {
        let original = ListPlansResponse {
            items: vec![sample_plan()],
            total: 1,
            limit: 20,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListPlansResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn optional_offset_zero_distinguishable_from_absent() {
        let with_zero = ListInvoicesRequest {
            org_slug: "acme".into(),
            offset: Some(0),
            limit: None,
        };
        let absent = ListInvoicesRequest {
            org_slug: "acme".into(),
            offset: None,
            limit: None,
        };
        let zero_bytes = with_zero.encode_to_vec();
        let absent_bytes = absent.encode_to_vec();
        assert_ne!(zero_bytes, absent_bytes);
        let r1 = ListInvoicesRequest::decode(&*zero_bytes).unwrap();
        let r2 = ListInvoicesRequest::decode(&*absent_bytes).unwrap();
        assert_eq!(r1.offset, Some(0));
        assert_eq!(r2.offset, None);
    }

    #[test]
    fn cancel_response_is_empty_message() {
        let resp = CancelSubscriptionResponse {};
        let bytes = resp.encode_to_vec();
        assert!(bytes.is_empty());
        assert_eq!(resp, CancelSubscriptionResponse::decode(&*bytes).unwrap());
    }

    #[test]
    fn checkout_request_optional_fields() {
        let original = CreateCheckoutRequest {
            org_slug: "acme".into(),
            order_type: "subscription".into(),
            plan_name: Some("pro".into()),
            billing_cycle: Some("monthly".into()),
            seats: Some(5),
            provider: Some("stripe".into()),
            success_url: "https://example.com/success".into(),
            cancel_url: "https://example.com/cancel".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = CreateCheckoutRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn overview_round_trip_with_usage_nested() {
        let original = BillingOverview {
            plan: Some(sample_plan()),
            status: "active".into(),
            billing_cycle: "monthly".into(),
            current_period_start: "2026-05-01T00:00:00Z".into(),
            current_period_end: "2026-06-01T00:00:00Z".into(),
            cancel_at_period_end: false,
            usage: Some(UsageOverview {
                pod_minutes: 42.5,
                included_pod_minutes: 100.0,
                users: 3,
                max_users: 5,
                runners: 1,
                max_runners: 2,
                concurrent_pods: 0,
                max_concurrent_pods: 5,
                repositories: 4,
                max_repositories: 10,
            }),
        };
        let bytes = original.encode_to_vec();
        let decoded = BillingOverview::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }
}
