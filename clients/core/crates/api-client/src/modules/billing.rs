use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::*;

impl ApiClient {
    pub async fn get_billing_overview(&self) -> Result<BillingOverview, ApiError> {
        self.get_resource(&self.org_path("/billing/overview"), "overview").await
    }

    pub async fn get_billing_subscription(&self) -> Result<Subscription, ApiError> {
        self.get_resource(&self.org_path("/billing/subscription"), "subscription").await
    }

    pub async fn create_billing_subscription(
        &self,
        data: &CreateSubscriptionRequest,
    ) -> Result<Subscription, ApiError> {
        self.post_resource(&self.org_path("/billing/subscription"), data, "subscription").await
    }

    pub async fn update_billing_subscription(
        &self,
        data: &UpdateSubscriptionRequest,
    ) -> Result<Subscription, ApiError> {
        self.put_resource(&self.org_path("/billing/subscription"), data, "subscription").await
    }

    pub async fn cancel_billing_subscription(&self) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path("/billing/subscription")).await
    }

    pub async fn list_billing_plans(&self) -> Result<PlanListResponse, ApiError> {
        self.get(&self.org_path("/billing/plans")).await
    }

    pub async fn get_billing_usage(
        &self,
        usage_type: Option<&str>,
    ) -> Result<BillingUsageResponse, ApiError> {
        let mut path = self.org_path("/billing/usage");
        if let Some(t) = usage_type {
            path = format!("{path}?type={t}");
        }
        self.get(&path).await
    }

    pub async fn check_billing_quota(
        &self,
        resource: &str,
        amount: Option<u32>,
    ) -> Result<QuotaCheckResponse, ApiError> {
        let mut path = self.org_path(&format!("/billing/quota/check?resource={resource}"));
        if let Some(a) = amount {
            path = format!("{path}&amount={a}");
        }
        self.get(&path).await
    }

    pub async fn create_billing_checkout(
        &self,
        data: &CheckoutRequest,
    ) -> Result<CheckoutStatus, ApiError> {
        self.post(&self.org_path("/billing/checkout"), data).await
    }

    pub async fn get_billing_checkout_status(
        &self,
        order_no: &str,
    ) -> Result<CheckoutStatus, ApiError> {
        self.get(&self.org_path(&format!("/billing/checkout/{order_no}")))
            .await
    }

    pub async fn request_cancel_subscription(
        &self,
        data: &CancelSubscriptionRequest,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path("/billing/subscription/cancel"),
            data,
        )
        .await
    }

    pub async fn reactivate_subscription(&self) -> Result<Subscription, ApiError> {
        self.post(
            &self.org_path("/billing/subscription/reactivate"),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn upgrade_subscription(
        &self,
        data: &UpgradeSubscriptionRequest,
    ) -> Result<Subscription, ApiError> {
        self.post(
            &self.org_path("/billing/subscription/upgrade"),
            data,
        )
        .await
    }

    pub async fn change_billing_cycle(
        &self,
        data: &ChangeBillingCycleRequest,
    ) -> Result<Subscription, ApiError> {
        self.post(
            &self.org_path("/billing/subscription/change-cycle"),
            data,
        )
        .await
    }

    pub async fn update_auto_renew(
        &self,
        data: &UpdateAutoRenewRequest,
    ) -> Result<Subscription, ApiError> {
        self.put(
            &self.org_path("/billing/subscription/auto-renew"),
            data,
        )
        .await
    }

    pub async fn get_seat_usage(&self) -> Result<SeatUsage, ApiError> {
        self.get(&self.org_path("/billing/seats")).await
    }

    pub async fn purchase_seats(
        &self,
        data: &PurchaseSeatsRequest,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(&self.org_path("/billing/seats/purchase"), data)
            .await
    }

    pub async fn list_billing_invoices(
        &self,
        limit: Option<u32>,
        offset: Option<u32>,
    ) -> Result<InvoiceListResponse, ApiError> {
        let mut path = self.org_path("/billing/invoices");
        let mut params = Vec::new();
        if let Some(l) = limit {
            params.push(format!("limit={l}"));
        }
        if let Some(o) = offset {
            params.push(format!("offset={o}"));
        }
        if !params.is_empty() {
            path = format!("{path}?{}", params.join("&"));
        }
        self.get(&path).await
    }

    pub async fn get_customer_portal(
        &self,
        data: &CustomerPortalRequest,
    ) -> Result<CustomerPortalResponse, ApiError> {
        self.post(&self.org_path("/billing/customer-portal"), data)
            .await
    }

    pub async fn get_billing_deployment_info(&self) -> Result<DeploymentInfo, ApiError> {
        self.get(&self.org_path("/billing/deployment")).await
    }
}
