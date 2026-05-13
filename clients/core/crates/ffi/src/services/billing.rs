use agentsmesh_types::proto_billing_v1 as billing_proto;

use crate::core::AgentsMeshCore;
use crate::dto::{
    invoice_list_from_proto, plan_list_from_proto, BillingOverviewDto,
    ChangeBillingCycleRequestDto, CreateCheckoutRequestDto, CreateCheckoutResponseDto,
    CreateSubscriptionRequestDto, CheckoutStatusDto, DeploymentInfoDto, InvoiceListResponseDto,
    PlanListResponseDto, PublicPricingResponseDto, SeatUsageDto, SubscriptionDto,
    UpdateSubscriptionRequestDto, UpgradeSubscriptionRequestDto,
};
use crate::error::CoreError;

#[uniffi::export(async_runtime = "tokio")]
impl AgentsMeshCore {
    pub async fn get_billing_overview(&self) -> Result<BillingOverviewDto, CoreError> {
        let req = billing_proto::GetOverviewRequest { org_slug: self.org_slug()? };
        let resp = self.api.get_billing_overview_connect(&req).await?;
        Ok(resp.into())
    }

    pub async fn list_billing_plans(&self) -> Result<PlanListResponseDto, CoreError> {
        let req = billing_proto::ListPlansRequest { org_slug: self.org_slug()? };
        let resp = self.api.list_billing_plans_connect(&req).await?;
        Ok(plan_list_from_proto(resp))
    }

    pub async fn get_billing_subscription(&self) -> Result<SubscriptionDto, CoreError> {
        let req = billing_proto::GetSubscriptionRequest { org_slug: self.org_slug()? };
        let sub = self.api.get_billing_subscription_connect(&req).await?;
        Ok(sub.into())
    }

    pub async fn create_billing_subscription(
        &self,
        req: CreateSubscriptionRequestDto,
    ) -> Result<SubscriptionDto, CoreError> {
        let proto_req = billing_proto::CreateSubscriptionRequest {
            org_slug: self.org_slug()?,
            plan_name: req.plan_name,
            billing_cycle: req.billing_cycle,
        };
        let sub = self.api.create_billing_subscription_connect(&proto_req).await?;
        Ok(sub.into())
    }

    pub async fn update_billing_subscription(
        &self,
        req: UpdateSubscriptionRequestDto,
    ) -> Result<SubscriptionDto, CoreError> {
        let proto_req = billing_proto::UpdateSubscriptionRequest {
            org_slug: self.org_slug()?,
            plan_name: req.plan_name,
        };
        let sub = self.api.update_billing_subscription_connect(&proto_req).await?;
        Ok(sub.into())
    }

    pub async fn cancel_billing_subscription(&self) -> Result<(), CoreError> {
        let req = billing_proto::CancelSubscriptionRequest { org_slug: self.org_slug()? };
        self.api.cancel_billing_subscription_connect(&req).await?;
        Ok(())
    }

    pub async fn request_cancel_billing_subscription(
        &self,
        immediate: bool,
    ) -> Result<(), CoreError> {
        let req = billing_proto::RequestCancelSubscriptionRequest {
            org_slug: self.org_slug()?,
            immediate,
        };
        self.api.request_cancel_subscription_connect(&req).await?;
        Ok(())
    }

    pub async fn reactivate_billing_subscription(&self) -> Result<SubscriptionDto, CoreError> {
        let req = billing_proto::ReactivateSubscriptionRequest { org_slug: self.org_slug()? };
        let sub = self.api.reactivate_subscription_connect(&req).await?;
        Ok(sub.into())
    }

    pub async fn upgrade_billing_subscription(
        &self,
        req: UpgradeSubscriptionRequestDto,
    ) -> Result<SubscriptionDto, CoreError> {
        let proto_req = billing_proto::UpgradeSubscriptionRequest {
            org_slug: self.org_slug()?,
            plan_name: req.plan_name,
        };
        let sub = self.api.upgrade_subscription_connect(&proto_req).await?;
        Ok(sub.into())
    }

    pub async fn change_billing_cycle(
        &self,
        req: ChangeBillingCycleRequestDto,
    ) -> Result<(), CoreError> {
        let proto_req = billing_proto::ChangeBillingCycleRequest {
            org_slug: self.org_slug()?,
            billing_cycle: req.billing_cycle,
        };
        self.api.change_billing_cycle_connect(&proto_req).await?;
        Ok(())
    }

    pub async fn update_billing_auto_renew(
        &self,
        auto_renew: bool,
    ) -> Result<SubscriptionDto, CoreError> {
        let req = billing_proto::UpdateAutoRenewRequest {
            org_slug: self.org_slug()?,
            auto_renew,
        };
        let sub = self.api.update_auto_renew_connect(&req).await?;
        Ok(sub.into())
    }

    pub async fn get_billing_seat_usage(&self) -> Result<SeatUsageDto, CoreError> {
        let req = billing_proto::GetSeatUsageRequest { org_slug: self.org_slug()? };
        let usage = self.api.get_seat_usage_connect(&req).await?;
        Ok(usage.into())
    }

    pub async fn purchase_billing_seats(&self, seats: i32) -> Result<(), CoreError> {
        let req = billing_proto::PurchaseSeatsRequest {
            org_slug: self.org_slug()?,
            seats,
        };
        self.api.purchase_seats_connect(&req).await?;
        Ok(())
    }

    pub async fn list_billing_invoices(
        &self,
        offset: Option<i32>,
        limit: Option<i32>,
    ) -> Result<InvoiceListResponseDto, CoreError> {
        let req = billing_proto::ListInvoicesRequest {
            org_slug: self.org_slug()?,
            offset,
            limit,
        };
        let resp = self.api.list_billing_invoices_connect(&req).await?;
        Ok(invoice_list_from_proto(resp))
    }

    pub async fn create_billing_checkout(
        &self,
        req: CreateCheckoutRequestDto,
    ) -> Result<CreateCheckoutResponseDto, CoreError> {
        let proto_req = billing_proto::CreateCheckoutRequest {
            org_slug: self.org_slug()?,
            order_type: req.order_type,
            plan_name: req.plan_name,
            billing_cycle: req.billing_cycle,
            seats: req.seats,
            provider: req.provider,
            success_url: req.success_url,
            cancel_url: req.cancel_url,
        };
        let resp = self.api.create_billing_checkout_connect(&proto_req).await?;
        Ok(resp.into())
    }

    pub async fn get_billing_checkout_status(
        &self,
        order_no: String,
    ) -> Result<CheckoutStatusDto, CoreError> {
        let req = billing_proto::GetCheckoutStatusRequest {
            org_slug: self.org_slug()?,
            order_no,
        };
        let status = self.api.get_billing_checkout_status_connect(&req).await?;
        Ok(status.into())
    }

    pub async fn get_billing_deployment_info(&self) -> Result<DeploymentInfoDto, CoreError> {
        let req = billing_proto::GetDeploymentInfoRequest { org_slug: self.org_slug()? };
        let info = self.api.get_billing_deployment_info_connect(&req).await?;
        Ok(info.into())
    }

    pub async fn get_public_billing_pricing(
        &self,
        currency: Option<String>,
    ) -> Result<PublicPricingResponseDto, CoreError> {
        let req = billing_proto::GetPublicPricingRequest { currency };
        let resp = self.api.get_public_pricing_connect(&req).await?;
        Ok(resp.into())
    }

    pub async fn get_public_billing_deployment_info(
        &self,
    ) -> Result<DeploymentInfoDto, CoreError> {
        let req = billing_proto::GetPublicDeploymentInfoRequest {};
        let info = self.api.get_public_deployment_info_connect(&req).await?;
        Ok(info.into())
    }
}
