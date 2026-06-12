use prost::Message;
use reqwest::header::{HeaderName, HeaderValue};

use crate::client::ApiClient;
use crate::error::ApiError;

/// Connect-RPC client helper. Binary in, binary out — `application/proto` is
/// the only content type the wasm/Rust client speaks (conventions §2.5).
///
/// `procedure` is the full path the Connect router expects, e.g.
/// `/proto.extension.v1.SkillRegistryService/ListSkillRegistries`.
///
/// Auth token comes from the same `AuthTokenStore` the REST request path uses
/// (no parallel token store). 401 → ApiError::AuthExpired so the wasm bridge
/// can re-prompt login; on success the body is the response's prost-encoded
/// bytes, decoded into `Res::Default + Res::decode`.
///
/// No JSON branch exists. There is no `application/json` fallback for the
/// client — conventions §2.5 § "Forbidden". curl-debug paths flip the
/// content-type on the server side, not here.
pub async fn connect_call<Req, Res>(
    client: &ApiClient,
    procedure: &str,
    body: &Req,
) -> Result<Res, ApiError>
where
    Req: Message,
    Res: Message + Default,
{
    let url = format!("{}{}", client.base_url, procedure);
    let payload = body.encode_to_vec();
    tracing::debug!(target: "api", procedure, bytes = payload.len(), "connect_call →");

    let mut builder = client
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

    if let Some(token) = client.auth_store.get_token() {
        let bearer = format!("Bearer {token}");
        // The token is user-controlled (in tests) but reqwest validates
        // header values; we fall back to silently omitting the header when
        // it can't be encoded so the call still hits the 401 path instead
        // of panicking. Production tokens are always ASCII JWTs.
        if let Ok(v) = HeaderValue::from_str(&bearer) {
            builder = builder.header(HeaderName::from_static("authorization"), v);
        }
    }

    let resp = builder.send().await?;
    let status = resp.status();

    if !status.is_success() {
        let status_code = status.as_u16();
        if status_code == 401 {
            tracing::warn!(target: "api", procedure, "connect_call ← 401 auth expired");
            return Err(ApiError::AuthExpired);
        }
        let status_text = status
            .canonical_reason()
            .unwrap_or("Unknown")
            .to_string();
        let body_bytes = resp.bytes().await.ok();
        let body_text = body_bytes
            .as_ref()
            .and_then(|b| std::str::from_utf8(b).ok())
            .filter(|s| !s.is_empty());
        let (code, server_message) = parse_connect_error(body_text);
        tracing::warn!(
            target: "api",
            procedure,
            status = status_code,
            code = code.as_deref().unwrap_or(""),
            message = server_message.as_deref().unwrap_or(""),
            "connect_call ← error"
        );
        return Err(ApiError::Http {
            status: status_code,
            status_text,
            code,
            server_message,
            data: None,
            url: Some(url),
        });
    }

    let resp_bytes = resp.bytes().await?;
    tracing::debug!(target: "api", procedure, status = status.as_u16(), bytes = resp_bytes.len(), "connect_call ← ok");
    Res::decode(resp_bytes).map_err(|e| ApiError::Decode(format!("prost decode: {e}")))
}

// Connect encodes unary errors as JSON `{"code":"...","message":"..."}` even
// on the proto lane — the protocol's error envelope is always JSON. Carrying
// the code through ApiError::Http lets clients map it to i18n instead of
// regex-matching messages; non-Connect bodies (proxy HTML) pass through raw.
fn parse_connect_error(body: Option<&str>) -> (Option<String>, Option<String>) {
    let Some(text) = body else {
        return (None, None);
    };
    if let Ok(v) = serde_json::from_str::<serde_json::Value>(text) {
        let code = v.get("code").and_then(|c| c.as_str()).map(String::from);
        let message = v.get("message").and_then(|m| m.as_str()).map(String::from);
        if code.is_some() || message.is_some() {
            return (code, message.or_else(|| Some(text.to_string())));
        }
    }
    (None, Some(text.to_string()))
}

#[cfg(test)]
mod tests {
    use super::parse_connect_error;

    #[test]
    fn connect_envelope_yields_code_and_message() {
        let (code, msg) =
            parse_connect_error(Some(r#"{"code":"already_exists","message":"source pod already resumed"}"#));
        assert_eq!(code.as_deref(), Some("already_exists"));
        assert_eq!(msg.as_deref(), Some("source pod already resumed"));
    }

    #[test]
    fn non_json_body_passes_through_raw() {
        let (code, msg) = parse_connect_error(Some("<html>502 Bad Gateway</html>"));
        assert_eq!(code, None);
        assert_eq!(msg.as_deref(), Some("<html>502 Bad Gateway</html>"));
    }

    #[test]
    fn json_without_envelope_fields_passes_through_raw() {
        let (code, msg) = parse_connect_error(Some(r#"{"error":"boom"}"#));
        assert_eq!(code, None);
        assert_eq!(msg.as_deref(), Some(r#"{"error":"boom"}"#));
    }

    #[test]
    fn empty_body_yields_nothing() {
        assert_eq!(parse_connect_error(None), (None, None));
    }
}
