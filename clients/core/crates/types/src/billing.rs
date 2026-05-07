use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BillingOverview {
    pub subscription: Option<Subscription>,
    pub usage_summary: Option<serde_json::Value>,
    pub seats: Option<SeatUsage>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Subscription {
    pub plan_name: Option<String>,
    pub status: Option<String>,
    pub billing_cycle: Option<String>,
    pub current_period_start: Option<String>,
    pub current_period_end: Option<String>,
    pub auto_renew: Option<bool>,
    pub seats: Option<i64>,
    pub cancel_at_period_end: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Plan {
    pub name: String,
    pub display_name: Option<String>,
    pub description: Option<String>,
    pub features: Option<serde_json::Value>,
    pub is_active: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlanPrice {
    pub plan_name: String,
    pub currency: String,
    pub monthly_price: Option<f64>,
    pub annual_price: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BillingUsage {
    pub usage_type: Option<String>,
    pub current: Option<f64>,
    pub limit: Option<f64>,
    pub period_start: Option<String>,
    pub period_end: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SeatUsage {
    pub total: Option<i64>,
    pub used: Option<i64>,
    pub available: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Invoice {
    pub id: String,
    pub amount: Option<f64>,
    pub currency: Option<String>,
    pub status: Option<String>,
    pub invoice_url: Option<String>,
    pub created_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckoutStatus {
    pub order_no: String,
    pub status: Option<String>,
    pub payment_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeploymentInfo {
    pub billing_enabled: Option<bool>,
    pub payment_providers: Option<Vec<String>>,
}

// `GET /api/v1/config/pricing` returns its own pricing-card-shaped payload
// (deployment_type + currency + per-plan price/limits) — distinct from the
// admin `Plan` entity returned by `/billing/plans`. Keep these two DTOs
// independent so a field added to one doesn't silently get dropped by the
// other's deserialize→reserialize wasm relay (regression: #329, #334).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PublicPricingResponse {
    pub deployment_type: String,
    pub currency: String,
    pub plans: Vec<PublicPlanPricing>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PublicPlanPricing {
    pub name: String,
    pub display_name: String,
    pub price_monthly: f64,
    pub price_yearly: f64,
    pub max_users: i64,
    pub max_runners: i64,
    pub max_repositories: i64,
    pub max_concurrent_pods: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSubscriptionRequest {
    pub plan_name: String,
    pub billing_cycle: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateSubscriptionRequest {
    pub plan_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckoutRequest {
    pub order_type: Option<String>,
    pub plan_name: String,
    pub billing_cycle: Option<String>,
    pub seats: Option<i64>,
    pub provider: Option<String>,
    pub success_url: Option<String>,
    pub cancel_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CancelSubscriptionRequest {
    pub immediate: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpgradeSubscriptionRequest {
    pub plan_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChangeBillingCycleRequest {
    pub billing_cycle: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateAutoRenewRequest {
    pub auto_renew: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PurchaseSeatsRequest {
    pub seats: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CustomerPortalRequest {
    pub return_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CustomerPortalResponse {
    pub url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlanListResponse {
    pub plans: Vec<Plan>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlanPriceListResponse {
    pub prices: Vec<PlanPrice>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InvoiceListResponse {
    pub invoices: Vec<Invoice>,
    pub total: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuotaCheckResponse {
    pub allowed: bool,
    pub current: Option<f64>,
    pub limit: Option<f64>,
}

#[cfg(test)]
mod tests {
    use super::*;

    // Backend payload (from `v1.PublicPricingResponse` in
    // backend/internal/api/rest/v1/billing_plans.go). The wasm bridge
    // (services/billing.rs::get_public_pricing) does deserialize→reserialize,
    // so any field missing from the Rust DTO is silently dropped before the
    // JS layer sees it — that was the regression behind the "$undefined" /
    // "undefinedundefined" pricing cards.
    const BACKEND_JSON: &str = r#"{
        "deployment_type": "global",
        "currency": "USD",
        "plans": [
            {
                "name": "based",
                "display_name": "Based",
                "price_monthly": 9.9,
                "price_yearly": 99.0,
                "max_users": 1,
                "max_runners": 1,
                "max_repositories": 5,
                "max_concurrent_pods": 5
            }
        ]
    }"#;

    #[test]
    fn public_pricing_decodes_backend_payload() {
        let resp: PublicPricingResponse = serde_json::from_str(BACKEND_JSON).unwrap();
        assert_eq!(resp.deployment_type, "global");
        assert_eq!(resp.currency, "USD");
        assert_eq!(resp.plans.len(), 1);
        let p = &resp.plans[0];
        assert_eq!(p.name, "based");
        assert_eq!(p.display_name, "Based");
        assert_eq!(p.price_monthly, 9.9);
        assert_eq!(p.price_yearly, 99.0);
        assert_eq!(p.max_users, 1);
        assert_eq!(p.max_concurrent_pods, 5);
    }

    #[test]
    fn public_pricing_wasm_relay_preserves_all_fields() {
        // Simulates services/billing.rs:get_public_pricing — deserialize via
        // typed DTO, then re-serialize to the JSON string the JS layer parses.
        // If any field gets dropped here, JS sees `undefined` and renders
        // "undefinedundefined" again.
        let typed: PublicPricingResponse = serde_json::from_str(BACKEND_JSON).unwrap();
        let relayed = serde_json::to_string(&typed).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();

        assert_eq!(parsed["currency"], "USD");
        assert_eq!(parsed["deployment_type"], "global");
        let plan = &parsed["plans"][0];
        for key in [
            "name", "display_name",
            "price_monthly", "price_yearly",
            "max_users", "max_runners", "max_repositories", "max_concurrent_pods",
        ] {
            assert!(!plan[key].is_null(), "field `{key}` was dropped by wasm relay");
        }
    }
}
