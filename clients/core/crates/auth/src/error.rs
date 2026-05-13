use agentsmesh_types::ServiceError;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AuthError {
    #[error("not authenticated")]
    NotAuthenticated,

    #[error("http error: {0}")]
    Http(#[from] reqwest::Error),

    #[error("invalid response: {0}")]
    InvalidResponse(String),

    #[error("server error: {status} - {message}")]
    Server {
        status: u16,
        message: String,
        code: Option<String>,
    },
}

impl From<AuthError> for ServiceError {
    fn from(e: AuthError) -> Self {
        match e {
            AuthError::NotAuthenticated => ServiceError::AuthExpired,
            AuthError::Http(http) => ServiceError::Network {
                message: http.to_string(),
            },
            AuthError::InvalidResponse(msg) => ServiceError::Http {
                status: 0,
                code: None,
                message: msg,
            },
            AuthError::Server {
                status,
                message,
                code,
            } => {
                if status == 401 {
                    return ServiceError::AuthExpired;
                }
                if status == 404 {
                    return ServiceError::ResourceNotFound {
                        resource: code.unwrap_or_else(|| "resource".into()),
                        id: None,
                    };
                }
                ServiceError::Http {
                    status,
                    code,
                    message,
                }
            }
        }
    }
}

#[derive(serde::Deserialize)]
pub(crate) struct ServerErrorBody {
    pub message: Option<String>,
    pub code: Option<String>,
}
