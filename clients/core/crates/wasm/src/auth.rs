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
}

#[wasm_bindgen]
pub struct WasmAuthManager {
    manager: Arc<AuthManager>,
    base_url: String,
}

impl WasmAuthManager {
    pub(crate) fn token_store_arc(&self) -> Arc<dyn AuthTokenStore> {
        self.manager.clone()
    }
}

#[wasm_bindgen]
impl WasmAuthManager {
    #[wasm_bindgen(constructor)]
    pub fn new(base_url: String) -> Self {
        let storage: Arc<dyn PersistentStorage> = Arc::new(WebLocalStorage);
        Self {
            manager: Arc::new(AuthManager::new(base_url.clone(), storage)),
            base_url,
        }
    }

    pub fn new_with_storage(base_url: String, storage: JsStorageBackend) -> Self {
        let storage: Arc<dyn PersistentStorage> = Arc::new(storage);
        Self {
            manager: Arc::new(AuthManager::new(base_url.clone(), storage)),
            base_url,
        }
    }

    #[wasm_bindgen(getter)]
    pub fn base_url(&self) -> String { self.base_url.clone() }

    pub async fn login(&self, email: String, password: String) -> Result<String, String> {
        let session = self.manager.login(&email, &password).await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&session).map_err(agentsmesh_services::wire)
    }

    pub async fn logout(&self) -> Result<(), String> {
        self.manager.logout().await.map_err(agentsmesh_services::wire)
    }

    pub async fn refresh_token(&self) -> Result<String, String> {
        let tokens = self.manager.refresh_token().await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&tokens).map_err(agentsmesh_services::wire)
    }

    pub async fn bootstrap(&self) -> Result<String, String> {
        let result = self.manager.bootstrap().await;
        serde_json::to_string(&result).map_err(agentsmesh_services::wire)
    }

    pub async fn fetch_organizations(&self) -> Result<String, String> {
        let orgs = self.manager.fetch_organizations().await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&orgs).map_err(agentsmesh_services::wire)
    }

    pub fn switch_org(&self, slug: &str) -> Result<(), String> {
        self.manager.switch_org(slug).map_err(agentsmesh_services::wire)
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

    pub fn apply_session(&self, session_json: &str) -> Result<(), String> {
        let session: agentsmesh_types::AuthSession = serde_json::from_str(session_json)
            .map_err(agentsmesh_services::wire)?;
        self.manager.apply_session(&session);
        Ok(())
    }

    pub fn set_organizations(&self, orgs_json: &str) -> Result<(), String> {
        let orgs: Vec<agentsmesh_types::Organization> = serde_json::from_str(orgs_json)
            .map_err(agentsmesh_services::wire)?;
        self.manager.replace_organizations(orgs);
        Ok(())
    }

    pub fn set_current_org(&self, org_json: &str) -> Result<(), String> {
        if org_json.is_empty() {
            self.manager.set_current_org(None);
        } else {
            let org: agentsmesh_types::Organization = serde_json::from_str(org_json)
                .map_err(agentsmesh_services::wire)?;
            self.manager.set_current_org(Some(org));
        }
        Ok(())
    }

    pub fn clear_session(&self) {
        self.manager.clear();
    }

    pub fn get_token(&self) -> Option<String> { self.manager.get_token() }
    pub fn get_refresh_token(&self) -> Option<String> { self.manager.get_refresh_token() }
}
