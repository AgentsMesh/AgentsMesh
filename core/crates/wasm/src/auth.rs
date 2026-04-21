use std::sync::Arc;

use agentsmesh_api_client::AuthTokenStore;
use agentsmesh_auth::{AuthManager, PersistentStorage};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
extern "C" {
    pub type JsStorageBackend;

    #[wasm_bindgen(method, structural)]
    fn get(this: &JsStorageBackend, key: &str) -> Option<String>;

    #[wasm_bindgen(method, structural)]
    fn set(this: &JsStorageBackend, key: &str, value: &str);

    #[wasm_bindgen(method, structural)]
    fn remove(this: &JsStorageBackend, key: &str);
}

unsafe impl Send for JsStorageBackend {}
unsafe impl Sync for JsStorageBackend {}

impl PersistentStorage for JsStorageBackend {
    fn get(&self, key: &str) -> Option<String> { JsStorageBackend::get(self, key) }
    fn set(&self, key: &str, value: &str) { JsStorageBackend::set(self, key, value); }
    fn remove(&self, key: &str) { JsStorageBackend::remove(self, key); }
    fn clear(&self) { self.remove("agentsmesh-auth"); }
}

struct WebLocalStorage;

impl PersistentStorage for WebLocalStorage {
    fn get(&self, key: &str) -> Option<String> {
        web_sys::window()?.local_storage().ok()??.get_item(key).ok()?
    }
    fn set(&self, key: &str, value: &str) {
        if let Some(s) = web_sys::window().and_then(|w| w.local_storage().ok()).flatten() {
            let _ = s.set_item(key, value);
        }
    }
    fn remove(&self, key: &str) {
        if let Some(s) = web_sys::window().and_then(|w| w.local_storage().ok()).flatten() {
            let _ = s.remove_item(key);
        }
    }
    fn clear(&self) { self.remove("agentsmesh-auth"); }
}

#[wasm_bindgen]
pub struct WasmAuthManager {
    manager: AuthManager,
    base_url: String,
}

#[wasm_bindgen]
impl WasmAuthManager {
    #[wasm_bindgen(constructor)]
    pub fn new(base_url: String) -> Self {
        let storage: Arc<dyn PersistentStorage> = Arc::new(WebLocalStorage);
        Self { manager: AuthManager::new(base_url.clone(), storage), base_url }
    }

    pub fn new_with_storage(base_url: String, storage: JsStorageBackend) -> Self {
        let storage: Arc<dyn PersistentStorage> = Arc::new(storage);
        Self { manager: AuthManager::new(base_url.clone(), storage), base_url }
    }

    #[wasm_bindgen(getter)]
    pub fn base_url(&self) -> String { self.base_url.clone() }

    pub async fn login(&self, email: String, password: String) -> Result<String, String> {
        let session = self.manager.login(&email, &password).await.map_err(|e| e.to_string())?;
        serde_json::to_string(&session).map_err(|e| e.to_string())
    }

    pub async fn logout(&self) -> Result<(), String> {
        self.manager.logout().await.map_err(|e| e.to_string())
    }

    pub async fn refresh_token(&self) -> Result<String, String> {
        let tokens = self.manager.refresh_token().await.map_err(|e| e.to_string())?;
        serde_json::to_string(&tokens).map_err(|e| e.to_string())
    }

    pub fn restore_session(&self) -> Result<bool, String> {
        self.manager.restore_session().map_err(|e| e.to_string())
    }

    pub async fn fetch_organizations(&self) -> Result<String, String> {
        let orgs = self.manager.fetch_organizations().await.map_err(|e| e.to_string())?;
        serde_json::to_string(&orgs).map_err(|e| e.to_string())
    }

    pub fn switch_org(&self, slug: &str) -> Result<(), String> {
        self.manager.switch_org(slug).map_err(|e| e.to_string())
    }

    pub fn is_authenticated(&self) -> bool { self.manager.is_authenticated() }

    pub fn get_current_user_json(&self) -> JsValue {
        match self.manager.current_user() {
            Some(u) => JsValue::from_str(&serde_json::to_string(&u).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    pub fn get_current_org_json(&self) -> JsValue {
        match self.manager.get_current_org() {
            Some(o) => JsValue::from_str(&serde_json::to_string(&o).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    pub fn get_organizations_json(&self) -> String {
        serde_json::to_string(&self.manager.get_organizations()).unwrap_or_default()
    }

    /// Apply an already-obtained AuthSession (SSO / register callback path).
    /// Writes token + refresh_token + user into Rust AuthState and persists.
    pub fn apply_session(&self, session_json: &str) -> Result<(), String> {
        let session: agentsmesh_types::AuthSession = serde_json::from_str(session_json)
            .map_err(|e| e.to_string())?;
        self.manager.apply_session(&session);
        Ok(())
    }

    /// Replace the organizations list (e.g. after a refetch outside fetch_organizations).
    /// Also promotes the first org to current_org if none is set.
    pub fn set_organizations(&self, orgs_json: &str) -> Result<(), String> {
        let orgs: Vec<agentsmesh_types::Organization> = serde_json::from_str(orgs_json)
            .map_err(|e| e.to_string())?;
        self.manager.replace_organizations(orgs);
        Ok(())
    }

    /// Set or clear current organization. Empty json string clears it.
    pub fn set_current_org(&self, org_json: &str) -> Result<(), String> {
        if org_json.is_empty() {
            self.manager.set_current_org_direct(None);
        } else {
            let org: agentsmesh_types::Organization = serde_json::from_str(org_json)
                .map_err(|e| e.to_string())?;
            self.manager.set_current_org_direct(Some(org));
        }
        Ok(())
    }

    /// Clear all session data (logout without API call). Useful for test reset.
    pub fn clear_session(&self) {
        self.manager.clear();
    }

    pub fn get_token(&self) -> Option<String> { AuthTokenStore::get_token(&self.manager) }
    pub fn get_refresh_token(&self) -> Option<String> { AuthTokenStore::get_refresh_token(&self.manager) }
}
