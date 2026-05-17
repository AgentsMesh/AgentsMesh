use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_auth::AuthManager;
use agentsmesh_services::BlockstoreService;
use agentsmesh_state::blockstore_state::BlockstoreState;

use crate::callbacks::StorageCallback;
use crate::dto::{BootstrapResultDto, OrganizationDto, UserDto};
use crate::error::CoreError;
use crate::storage_bridge::StorageBridge;

#[derive(uniffi::Object)]
pub struct AgentsMeshCore {
    pub(crate) auth: Arc<AuthManager>,
    pub(crate) api: Arc<ApiClient>,
    /// Local SSOT for the blockstore — mirrors what desktop/web have.
    /// Mutations + load_subtree go through this; sync flat-map readers
    /// (`blocks_json`, `refs_json`, ...) read from its in-process state
    /// without a backend round-trip, which is what the WebView-embedded
    /// React DocumentView expects.
    pub(crate) blockstore: Arc<BlockstoreService>,
}

#[uniffi::export]
impl AgentsMeshCore {
    #[uniffi::constructor]
    pub fn new(base_url: String, storage: Box<dyn StorageCallback>) -> Self {
        let bridge = Arc::new(StorageBridge::new(Arc::from(storage)));
        let auth = Arc::new(AuthManager::new(base_url.clone(), bridge));
        let api = Arc::new(ApiClient::new(base_url, auth.clone()));
        let blockstore = Arc::new(BlockstoreService::new(api.clone(), BlockstoreState::new()));
        Self { auth, api, blockstore }
    }

    pub fn is_authenticated(&self) -> bool {
        self.auth.is_authenticated()
    }

    /// Strongly-typed current-user accessor for Swift/Kotlin.
    pub fn get_current_user(&self) -> Option<UserDto> {
        self.auth.current_user().map(UserDto::from)
    }

    /// Strongly-typed current-org accessor for Swift/Kotlin.
    pub fn get_current_org(&self) -> Option<OrganizationDto> {
        self.auth.get_current_org().map(OrganizationDto::from)
    }

    /// Strongly-typed organization list accessor for Swift/Kotlin.
    pub fn get_organizations(&self) -> Vec<OrganizationDto> {
        self.auth
            .get_organizations()
            .into_iter()
            .map(OrganizationDto::from)
            .collect()
    }

    // ── Legacy JSON accessors ──
    // Retained for WASM/node-bridge parity until those frontends migrate off.

    pub fn get_current_user_json(&self) -> Option<String> {
        self.auth
            .current_user()
            .and_then(|u| serde_json::to_string(&u).ok())
    }

    pub fn get_current_org_json(&self) -> Option<String> {
        self.auth
            .get_current_org()
            .and_then(|o| serde_json::to_string(&o).ok())
    }

    pub fn get_organizations_json(&self) -> Result<String, CoreError> {
        let orgs = self.auth.get_organizations();
        Ok(serde_json::to_string(&orgs)?)
    }
}

#[uniffi::export(async_runtime = "tokio")]
impl AgentsMeshCore {
    /// Hydrate the auth state by reading storage, refreshing the token if
    /// near expiry, and validating identity against the server. Returns
    /// a strongly-typed `BootstrapResultDto` enum — Swift/Kotlin pattern-
    /// match on `.anonymous` / `.authenticated(user, currentOrg)` /
    /// `.anonymousAfterCleanup(reason)` to drive UI without parsing JSON.
    pub async fn bootstrap(&self) -> BootstrapResultDto {
        BootstrapResultDto::from(self.auth.bootstrap().await)
    }
}

/// Bootstrap the global tracing subscriber. Pass `log_dir = None` for
/// stderr-only output (tests, early boot before sandbox path is known);
/// otherwise we create the directory and roll daily files inside it.
/// Idempotent — Swift can call this from every app launch without guarding.
#[uniffi::export]
pub fn init_logger(log_dir: Option<String>, level: String) -> Result<(), CoreError> {
    let cfg = match log_dir {
        Some(p) => agentsmesh_logging::LogConfig::file(p, level),
        None => agentsmesh_logging::LogConfig::console(level),
    };
    agentsmesh_logging::init(cfg).map_err(|e| CoreError::Unknown {
        message: e.to_string(),
    })?;
    agentsmesh_logging::install_panic_hook();
    Ok(())
}

/// Host-side log entrypoint for Swift/Kotlin callers. Re-emits through the
/// same `tracing` subscriber the Rust workspace uses, so iOS-side events
/// land in the same rolling file as Rust `tracing::*` calls.
#[uniffi::export]
pub fn log_event(level: String, target: String, msg: String) {
    agentsmesh_logging::log_event(&level, &target, &msg);
}
