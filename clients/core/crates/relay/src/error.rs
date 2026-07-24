use agentsmesh_types::ServiceError;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum RelayError {
    #[error("connection error: {0}")]
    Connection(String),

    #[error("not connected: {0}")]
    NotConnected(String),

    #[error("send error: {0}")]
    Send(String),

    #[error("protocol error: {0}")]
    Protocol(#[from] agentsmesh_protocol::ProtocolError),
}

impl From<&RelayError> for ServiceError {
    fn from(e: &RelayError) -> Self {
        ServiceError::Network {
            message: e.to_string(),
        }
    }
}

impl From<RelayError> for ServiceError {
    fn from(e: RelayError) -> Self {
        ServiceError::from(&e)
    }
}

// LCOV_EXCL_START: test-only code
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn variants_keep_context_when_converted_to_service_errors() {
        let cases = [
            RelayError::Connection("dial failed".to_string()),
            RelayError::NotConnected("pod-1".to_string()),
            RelayError::Send("socket closed".to_string()),
            RelayError::Protocol(agentsmesh_protocol::ProtocolError::EmptyMessage),
        ];

        for error in cases {
            let expected = error.to_string();
            let borrowed = ServiceError::from(&error);
            assert_eq!(borrowed, ServiceError::Network { message: expected });
        }

        let owned = RelayError::NotConnected("pod-2".to_string());
        assert_eq!(
            ServiceError::from(owned),
            ServiceError::Network {
                message: "not connected: pod-2".to_string(),
            }
        );
    }
}
// LCOV_EXCL_STOP
