mod command;
mod connection;
pub mod dispatch;
mod dispatch_snapshot;
mod driver;
pub mod error;
pub mod pool;
pub mod retry;
pub mod types;

pub use error::RelayError;
pub use pool::RelayConnectionPool;
pub use types::{
    AcpCallback, ConnectionHandle, DisconnectCallback, GenerationAcpCallback,
    GenerationDisconnectCallback, GenerationStatusCallback, OutputCallback, RelayStatus,
    RelayStatusInfo, StatusCallback,
};

// LCOV_EXCL_START: test-only code
#[cfg(test)]
mod dispatch_snapshot_contract_tests;
#[cfg(test)]
mod dispatch_snapshot_state_tests;
#[cfg(test)]
mod dispatch_tests;
#[cfg(test)]
mod integration_tests;
// LCOV_EXCL_STOP
