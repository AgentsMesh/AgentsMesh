use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub enum ServiceError {
    Http {
        status: u16,
        code: Option<String>,
        message: String,
    },
    AuthExpired,
    Network {
        message: String,
    },
    InvalidJson {
        message: String,
    },
    ResourceNotFound {
        resource: String,
        id: Option<String>,
    },
    Unknown {
        message: String,
    },
}

impl ServiceError {
    pub fn to_wire(&self) -> String {
        serde_json::to_string(self).unwrap_or_else(|_| {
            format!(
                r#"{{"kind":"unknown","message":"serialization failed: {}"}}"#,
                self.fallback_message().replace('"', "'")
            )
        })
    }

    pub fn unknown(message: impl Into<String>) -> Self {
        ServiceError::Unknown {
            message: message.into(),
        }
    }

    fn fallback_message(&self) -> String {
        match self {
            ServiceError::Http { message, .. } => message.clone(),
            ServiceError::AuthExpired => "auth expired".to_string(),
            ServiceError::Network { message } => message.clone(),
            ServiceError::InvalidJson { message } => message.clone(),
            ServiceError::ResourceNotFound { resource, .. } => {
                format!("{resource} not found")
            }
            ServiceError::Unknown { message } => message.clone(),
        }
    }
}

impl From<serde_json::Error> for ServiceError {
    fn from(e: serde_json::Error) -> Self {
        ServiceError::InvalidJson {
            message: e.to_string(),
        }
    }
}

impl From<String> for ServiceError {
    fn from(s: String) -> Self {
        if s.starts_with('{') {
            if let Ok(parsed) = serde_json::from_str::<ServiceError>(&s) {
                return parsed;
            }
        }
        ServiceError::Unknown { message: s }
    }
}

impl From<&str> for ServiceError {
    fn from(s: &str) -> Self {
        ServiceError::from(s.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn http_roundtrip() {
        let err = ServiceError::Http {
            status: 404,
            code: Some("RESOURCE_NOT_FOUND".into()),
            message: "Pod not found".into(),
        };
        let wire = err.to_wire();
        let parsed: ServiceError = serde_json::from_str(&wire).unwrap();
        assert_eq!(err, parsed);
    }

    #[test]
    fn resource_not_found_tag() {
        let err = ServiceError::ResourceNotFound {
            resource: "Pod".into(),
            id: Some("pk_123".into()),
        };
        let wire = err.to_wire();
        assert!(wire.contains(r#""kind":"resource_not_found""#));
        assert!(wire.contains(r#""resource":"Pod""#));
    }

    #[test]
    fn string_recovers_wire_format() {
        let original = ServiceError::AuthExpired;
        let wire = original.to_wire();
        let round = ServiceError::from(wire);
        assert_eq!(round, ServiceError::AuthExpired);
    }

    #[test]
    fn string_falls_back_to_unknown() {
        let err = ServiceError::from("random message".to_string());
        assert!(matches!(err, ServiceError::Unknown { .. }));
    }
}
