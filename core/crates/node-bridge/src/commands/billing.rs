use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn billing_get_overview(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_overview().await.map_err(err)
    }

    #[napi]
    pub async fn billing_get_subscription(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_subscription().await.map_err(err)
    }

    #[napi]
    pub async fn billing_create_subscription(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.create_subscription(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_cancel_subscription(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.cancel_subscription().await.map_err(err)
    }

    #[napi]
    pub async fn billing_update_subscription(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.update_subscription(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_list_plans(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.list_plans().await.map_err(err)
    }

    #[napi]
    pub async fn billing_get_usage(&self, usage_type: Option<String>) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_usage(usage_type).await.map_err(err)
    }

    #[napi]
    pub async fn billing_check_quota(&self, resource: String, amount: Option<u32>) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.check_quota(&resource, amount).await.map_err(err)
    }

    #[napi]
    pub async fn billing_create_checkout(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.create_checkout(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_get_checkout_status(&self, order_no: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_checkout_status(&order_no).await.map_err(err)
    }

    #[napi]
    pub async fn billing_request_cancel(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.request_cancel(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_reactivate(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.reactivate().await.map_err(err)
    }

    #[napi]
    pub async fn billing_upgrade(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.upgrade(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_change_cycle(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.change_cycle(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_update_auto_renew(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.update_auto_renew(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_get_seat_usage(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_seat_usage().await.map_err(err)
    }

    #[napi]
    pub async fn billing_purchase_seats(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.purchase_seats(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_list_invoices(&self, limit: Option<u32>, offset: Option<u32>) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.list_invoices(limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn billing_get_customer_portal(&self, json: String) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_customer_portal(&json).await.map_err(err)
    }

    #[napi]
    pub async fn billing_get_deployment_info(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_deployment_info().await.map_err(err)
    }

    #[napi]
    pub async fn billing_get_public_pricing(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_public_pricing().await.map_err(err)
    }

    #[napi]
    pub async fn billing_get_public_deployment_info(&self) -> napi::Result<String> {
        let svc = self.billing.lock().await;
            svc.get_public_deployment_info().await.map_err(err)
    }

}
