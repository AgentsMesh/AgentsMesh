use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_promocode_v1 as pc_proto;
use prost::Message;

pub struct PromoCodeService {
    client: Arc<ApiClient>,
}

impl PromoCodeService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // TS encodes via @bufbuild/protobuf .toBinary() → wasm bridge → here →
    // ApiClient.*_connect (binary in / binary out, conventions §2.5).

    pub async fn validate_promo_code_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pc_proto::ValidatePromoCodeRequest::decode(request_bytes)
            .map_err(|e| format!("decode validate_promo_code request: {e}"))?;
        tracing::debug!(target: "promocode", org_slug = %req.org_slug, "validate promo code");
        let resp = self.client.validate_promo_code_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn redeem_promo_code_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pc_proto::RedeemPromoCodeRequest::decode(request_bytes)
            .map_err(|e| format!("decode redeem_promo_code request: {e}"))?;
        tracing::info!(target: "promocode", org_slug = %req.org_slug, "redeem promo code");
        let resp = self.client.redeem_promo_code_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_redemption_history_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pc_proto::GetRedemptionHistoryRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_redemption_history request: {e}"))?;
        tracing::debug!(target: "promocode", org_slug = %req.org_slug, "get redemption history");
        let resp = self.client.get_redemption_history_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
