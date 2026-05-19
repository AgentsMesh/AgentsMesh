use std::sync::{Arc, RwLock};

use agentsmesh_types::{Organization, User};

use crate::error::AuthError;
use crate::state::{session_storage_key, AuthState};
use crate::storage::PersistentStorage;

pub struct AuthManager {
    pub(crate) state: Arc<RwLock<AuthState>>,
    pub(crate) storage: Arc<dyn PersistentStorage>,
    pub(crate) base_url: String,
    pub(crate) http: reqwest::Client,
    /// Serializes concurrent calls to `refresh_token()` within this
    /// process: bootstrap-triggered refresh and any other in-flight
    /// refresh on the same `AuthManager` must not race. Cross-process
    /// contention (e.g. two web tabs) is mediated by the server's
    /// one-shot refresh-token rotation: whichever side loses the race
    /// gets a 401 and lands on cleanup.
    pub(crate) refresh_lock: futures::lock::Mutex<()>,
}

impl AuthManager {
    pub fn new(base_url: String, storage: Arc<dyn PersistentStorage>) -> Self {
        Self {
            state: Arc::new(RwLock::new(AuthState::default())),
            storage,
            base_url: base_url.trim_end_matches('/').to_string(),
            http: reqwest::Client::new(),
            refresh_lock: futures::lock::Mutex::new(()),
        }
    }

    pub fn base_url(&self) -> &str {
        &self.base_url
    }

    pub(crate) fn session_key(&self) -> String {
        session_storage_key(&self.base_url)
    }

    pub fn is_authenticated(&self) -> bool {
        let state = self.read_state();
        match state.session.as_ref() {
            Some(s) => s.expires_at > now_unix_secs(),
            None => false,
        }
    }

    pub fn expires_at(&self) -> Option<i64> {
        self.read_state().session.as_ref().map(|s| s.expires_at)
    }

    pub fn current_user(&self) -> Option<User> {
        self.read_state().user.clone()
    }

    pub fn replace_organizations(&self, orgs: Vec<Organization>) {
        let promote = {
            let mut s = self.write_state();
            s.organizations = orgs.clone();
            if s.current_org.is_none() {
                orgs.into_iter().next()
            } else {
                None
            }
        };
        if let Some(first) = promote {
            self.set_current_org(Some(first));
        }
    }

    pub fn set_current_org(&self, org: Option<Organization>) {
        {
            let mut s = self.write_state();
            s.current_org = org.clone();
            if let Some(sess) = s.session.as_mut() {
                sess.current_org_slug = org.as_ref().map(|o| o.slug.clone());
            }
        }
        self.persist();
    }

    pub fn clear(&self) {
        self.reset_local();
    }

    pub(crate) fn reset_local(&self) {
        self.write_state().clear();
        self.storage.remove(&self.session_key());
    }

    pub(crate) fn persist(&self) {
        let state = self.read_state();
        let Some(session) = state.session.as_ref() else {
            return;
        };
        if let Ok(json) = serde_json::to_string(session) {
            self.storage.set(&self.session_key(), &json);
        }
    }

    pub(crate) fn bearer_header(&self) -> Result<String, AuthError> {
        let state = self.read_state();
        state
            .session
            .as_ref()
            .map(|s| format!("Bearer {}", s.access_token))
            .ok_or(AuthError::NotAuthenticated)
    }

    /// Poison-tolerant read guard: never recover from a panic that left
    /// the lock poisoned — reading the snapshot is always safe; the
    /// alternative (panic-propagate) surfaces as an opaque runtime error
    /// across the napi/wasm boundary.
    pub(crate) fn read_state(&self) -> std::sync::RwLockReadGuard<'_, AuthState> {
        self.state.read().unwrap_or_else(|e| e.into_inner())
    }

    pub(crate) fn write_state(&self) -> std::sync::RwLockWriteGuard<'_, AuthState> {
        self.state.write().unwrap_or_else(|e| e.into_inner())
    }
}

pub(crate) fn now_unix_secs() -> i64 {
    use web_time::{SystemTime, UNIX_EPOCH};
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}
