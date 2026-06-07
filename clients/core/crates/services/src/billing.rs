use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_billing_v1 as billing_proto;
use prost::Message;

pub struct BillingService {
    client: Arc<ApiClient>,
}

impl BillingService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Each `*_connect` method takes prost-encoded bytes and returns
    // prost-encoded bytes — matching the wasm bridge's `Result<Vec<u8>, String>`
    // surface (conventions §2.5). Caller (TS) encodes via
    // @bufbuild/protobuf .toBinary() and decodes via .fromBinary().

    pub async fn get_overview_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetOverviewRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_overview request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, "get overview");
        let resp = self.client.get_billing_overview_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_plans_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::ListPlansRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_plans request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, "list plans");
        let resp = self.client.list_billing_plans_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_subscription_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetSubscriptionRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_subscription request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, "get subscription");
        let resp = self.client.get_billing_subscription_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_subscription_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::CreateSubscriptionRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_subscription request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, plan_name = %req.plan_name, "create subscription");
        let resp = self.client.create_billing_subscription_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_subscription_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::UpdateSubscriptionRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_subscription request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, plan_name = %req.plan_name, "update subscription");
        let resp = self.client.update_billing_subscription_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn cancel_subscription_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::CancelSubscriptionRequest::decode(request_bytes)
            .map_err(|e| format!("decode cancel_subscription request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, "cancel subscription");
        let resp = self.client.cancel_billing_subscription_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn request_cancel_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::RequestCancelSubscriptionRequest::decode(request_bytes)
            .map_err(|e| format!("decode request_cancel request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, immediate = req.immediate, "request cancel subscription");
        let resp = self.client.request_cancel_subscription_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn reactivate_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::ReactivateSubscriptionRequest::decode(request_bytes)
            .map_err(|e| format!("decode reactivate request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, "reactivate subscription");
        let resp = self.client.reactivate_subscription_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn upgrade_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::UpgradeSubscriptionRequest::decode(request_bytes)
            .map_err(|e| format!("decode upgrade request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, plan_name = %req.plan_name, "upgrade subscription");
        let resp = self.client.upgrade_subscription_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn change_cycle_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::ChangeBillingCycleRequest::decode(request_bytes)
            .map_err(|e| format!("decode change_cycle request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, billing_cycle = %req.billing_cycle, "change billing cycle");
        let resp = self.client.change_billing_cycle_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_auto_renew_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::UpdateAutoRenewRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_auto_renew request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, auto_renew = req.auto_renew, "update auto renew");
        let resp = self.client.update_auto_renew_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_seat_usage_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetSeatUsageRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_seat_usage request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, "get seat usage");
        let resp = self.client.get_seat_usage_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn purchase_seats_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::PurchaseSeatsRequest::decode(request_bytes)
            .map_err(|e| format!("decode purchase_seats request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, seats = req.seats, "purchase seats");
        let resp = self.client.purchase_seats_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_invoices_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::ListInvoicesRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_invoices request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, "list invoices");
        let resp = self.client.list_billing_invoices_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_checkout_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::CreateCheckoutRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_checkout request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, order_type = %req.order_type, "create checkout");
        let resp = self.client.create_billing_checkout_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_checkout_status_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetCheckoutStatusRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_checkout_status request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, order_no = %req.order_no, "get checkout status");
        let resp = self.client.get_billing_checkout_status_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_deployment_info_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetDeploymentInfoRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_deployment_info request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, "get deployment info");
        let resp = self.client.get_billing_deployment_info_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_public_pricing_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetPublicPricingRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_public_pricing request: {e}"))?;
        tracing::debug!(target: "billing", "get public pricing");
        let resp = self.client.get_public_pricing_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_public_deployment_info_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetPublicDeploymentInfoRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_public_deployment_info request: {e}"))?;
        tracing::debug!(target: "billing", "get public deployment info");
        let resp = self.client.get_public_deployment_info_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    // -------- Usage / quota / customer portal — REST refugees --------

    pub async fn get_usage_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetUsageRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_usage request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, "get usage");
        let resp = self.client.get_usage_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_usage_history_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::GetUsageHistoryRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_usage_history request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, months = req.months, "get usage history");
        let resp = self
            .client
            .get_usage_history_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn check_quota_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::CheckQuotaRequest::decode(request_bytes)
            .map_err(|e| format!("decode check_quota request: {e}"))?;
        tracing::debug!(target: "billing", org_slug = %req.org_slug, resource = %req.resource, "check quota");
        let resp = self.client.check_quota_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn set_custom_quota_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::SetCustomQuotaRequest::decode(request_bytes)
            .map_err(|e| format!("decode set_custom_quota request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, resource = %req.resource, "set custom quota");
        let resp = self
            .client
            .set_custom_quota_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_customer_portal_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = billing_proto::CreateCustomerPortalRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_customer_portal request: {e}"))?;
        tracing::info!(target: "billing", org_slug = %req.org_slug, "create customer portal");
        let resp = self
            .client
            .create_customer_portal_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
