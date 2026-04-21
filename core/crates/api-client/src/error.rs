use thiserror::Error;

#[derive(Debug, Error)]
pub enum ApiError {
    #[error("HTTP {status}: {}", Self::describe_http(status_text, server_message.as_deref(), code.as_deref()))]
    Http {
        status: u16,
        status_text: String,
        code: Option<String>,
        server_message: Option<String>,
        data: Option<serde_json::Value>,
    },

    #[error("auth expired")]
    AuthExpired,

    #[error("network error: {0}")]
    Network(#[from] reqwest::Error),

    #[error("json error: {0}")]
    Json(#[from] serde_json::Error),
}

impl ApiError {
    pub fn has_code(&self, code: &str) -> bool {
        matches!(self, ApiError::Http { code: Some(c), .. } if c == code)
    }

    pub fn status(&self) -> Option<u16> {
        match self {
            ApiError::Http { status, .. } => Some(*status),
            _ => None,
        }
    }

    fn describe_http(status_text: &str, server_message: Option<&str>, code: Option<&str>) -> String {
        match (server_message, code) {
            (Some(msg), Some(c)) => format!("{msg} [{c}]"),
            (Some(msg), None) => msg.to_string(),
            (None, Some(c)) => format!("{status_text} [{c}]"),
            (None, None) => status_text.to_string(),
        }
    }
}
