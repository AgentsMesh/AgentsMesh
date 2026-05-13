use prost::Message;
use reqwest::header::{HeaderName, HeaderValue};

use crate::error::{AuthError, ServerErrorBody};
use crate::manager::AuthManager;

// AuthManager-internal Connect-RPC helper. Mirrors
// `api-client::connect_call` (conventions §2.5: binary wire,
// application/proto) but lives here because AuthManager owns its own
// `reqwest::Client` + `base_url` and does not flow through `ApiClient`.
// Splitting auth's wire layer from the rest of the API surface is
// deliberate: AuthManager runs BEFORE the API client exists (the token
// it produces is what ApiClient consumes) so a circular ApiClient
// dependency is impossible.
//
// `procedure` is the full Connect router path, e.g.
// `/proto.auth.v1.AuthService/Login`. `bearer` is the Authorization
// header value when the RPC needs auth (Logout); None for the public
// surface (Login / Register / Refresh / OAuth / Verify / Forgot- and
// ResetPassword).
pub(crate) async fn connect_call<Req, Res>(
    manager: &AuthManager,
    procedure: &str,
    body: &Req,
    bearer: Option<&str>,
) -> Result<Res, AuthError>
where
    Req: Message,
    Res: Message + Default,
{
    let url = format!("{}{}", manager.base_url(), procedure);
    let payload = body.encode_to_vec();

    let mut builder = manager
        .http
        .post(&url)
        .header(
            HeaderName::from_static("content-type"),
            HeaderValue::from_static("application/proto"),
        )
        .header(
            HeaderName::from_static("connect-protocol-version"),
            HeaderValue::from_static("1"),
        )
        .body(payload);

    if let Some(b) = bearer {
        if let Ok(v) = HeaderValue::from_str(b) {
            builder = builder.header(HeaderName::from_static("authorization"), v);
        }
    }

    let resp = builder.send().await?;
    let status = resp.status();

    if !status.is_success() {
        let status_code = status.as_u16();
        // Connect error body shape: `{"code":"...","message":"..."}`
        // (server emits JSON regardless of request content-type because
        // it carries the typed Connect error code). We reuse
        // `ServerErrorBody` so the AuthError::Server variant looks the
        // same as the legacy REST path — bootstrap_tests + other callers
        // still match on `crate::AuthError::Server { status: 4xx, .. }`.
        let body_text = resp.text().await.unwrap_or_default();
        let (message, code) =
            match serde_json::from_str::<ServerErrorBody>(&body_text) {
                Ok(b) => (
                    b.message.unwrap_or_else(|| "unknown error".into()),
                    b.code,
                ),
                Err(_) => {
                    if body_text.is_empty() {
                        ("failed to parse error response".into(), None)
                    } else {
                        (body_text, None)
                    }
                }
            };
        return Err(AuthError::Server {
            status: status_code,
            message,
            code,
        });
    }

    let resp_bytes = resp.bytes().await?;
    Res::decode(resp_bytes)
        .map_err(|e| AuthError::InvalidResponse(format!("proto decode: {e}")))
}
