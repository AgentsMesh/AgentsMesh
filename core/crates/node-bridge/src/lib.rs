use std::path::PathBuf;
use std::sync::Arc;
use tokio::sync::Mutex;
use napi_derive::napi;

use agentsmesh_api_client::{ApiClient, AuthTokenStore};
use agentsmesh_auth::{AuthManager, storage::PersistentStorage};
use agentsmesh_services::*;
use agentsmesh_state::*;

mod file_storage;
use file_storage::FileStorage;

fn err(e: impl std::fmt::Display) -> napi::Error {
    napi::Error::from_reason(e.to_string())
}

#[napi]
pub struct AppState {
    auth: Arc<AuthManager>,
    pod: Arc<Mutex<PodService>>,
    runner: Arc<Mutex<RunnerService>>,
    ticket: Arc<Mutex<TicketService>>,
    channel: Arc<Mutex<ChannelService>>,
    loop_svc: Arc<Mutex<LoopService>>,
    autopilot: Arc<Mutex<AutopilotService>>,
    mesh: Arc<Mutex<MeshService>>,
    billing: Arc<Mutex<BillingService>>,
    extension: Arc<Mutex<ExtensionService>>,
    repository: Arc<Mutex<RepositoryService>>,
    invitation: Arc<Mutex<InvitationService>>,
    grant: Arc<Mutex<GrantService>>,
    apikey: Arc<Mutex<ApiKeyService>>,
    binding: Arc<Mutex<BindingService>>,
    message: Arc<Mutex<MessageService>>,
    notification: Arc<Mutex<NotificationService>>,
    org: Arc<Mutex<OrgApiService>>,
    user: Arc<Mutex<UserApiService>>,
    user_credential: Arc<Mutex<UserCredentialService>>,
    agent: Arc<Mutex<AgentService>>,
    file: Arc<Mutex<FileService>>,
    support_ticket: Arc<Mutex<SupportTicketService>>,
    ticket_relations: Arc<Mutex<TicketRelationsService>>,
    token_usage: Arc<Mutex<TokenUsageService>>,
    auth_api: Arc<Mutex<AuthApiService>>,
    blockstore: Arc<Mutex<BlockstoreService>>,
    client: Arc<ApiClient>,
}

#[napi]
impl AppState {
    #[napi(constructor)]
    pub fn new(base_url: String, storage_dir: String) -> napi::Result<Self> {
        let dir = PathBuf::from(storage_dir);
        let _ = std::fs::create_dir_all(&dir);
        let storage: Arc<dyn PersistentStorage> = Arc::new(FileStorage::new(dir));
        let auth = Arc::new(AuthManager::new(base_url.clone(), storage));
        let _ = auth.restore_session();
        let client = Arc::new(ApiClient::new(base_url, auth.clone()));
        let c = client.clone();
        Ok(Self {
            auth,
            pod: Arc::new(Mutex::new(PodService::new(c.clone(), pod_state::PodState::new()))),
            runner: Arc::new(Mutex::new(RunnerService::new(c.clone(), runner_state::RunnerState::new()))),
            ticket: Arc::new(Mutex::new(TicketService::new(c.clone(), ticket_state::TicketState::new()))),
            channel: Arc::new(Mutex::new(ChannelService::new(c.clone(), channel_state::ChannelState::new()))),
            loop_svc: Arc::new(Mutex::new(LoopService::new(c.clone(), loop_state::LoopState::new()))),
            autopilot: Arc::new(Mutex::new(AutopilotService::new(c.clone(), autopilot_state::AutopilotState::new()))),
            mesh: Arc::new(Mutex::new(MeshService::new(c.clone(), mesh_state::MeshState::new()))),
            billing: Arc::new(Mutex::new(BillingService::new(c.clone()))),
            extension: Arc::new(Mutex::new(ExtensionService::new(c.clone()))),
            repository: Arc::new(Mutex::new(RepositoryService::new(c.clone()))),
            invitation: Arc::new(Mutex::new(InvitationService::new(c.clone()))),
            grant: Arc::new(Mutex::new(GrantService::new(c.clone()))),
            apikey: Arc::new(Mutex::new(ApiKeyService::new(c.clone()))),
            binding: Arc::new(Mutex::new(BindingService::new(c.clone()))),
            message: Arc::new(Mutex::new(MessageService::new(c.clone()))),
            notification: Arc::new(Mutex::new(NotificationService::new(c.clone()))),
            org: Arc::new(Mutex::new(OrgApiService::new(c.clone()))),
            user: Arc::new(Mutex::new(UserApiService::new(c.clone()))),
            user_credential: Arc::new(Mutex::new(UserCredentialService::new(c.clone()))),
            agent: Arc::new(Mutex::new(AgentService::new(c.clone()))),
            file: Arc::new(Mutex::new(FileService::new(c.clone()))),
            support_ticket: Arc::new(Mutex::new(SupportTicketService::new(c.clone()))),
            ticket_relations: Arc::new(Mutex::new(TicketRelationsService::new(c.clone()))),
            token_usage: Arc::new(Mutex::new(TokenUsageService::new(c.clone()))),
            auth_api: Arc::new(Mutex::new(AuthApiService::new(c.clone()))),
            blockstore: Arc::new(Mutex::new(BlockstoreService::new(
                c.clone(),
                blockstore_state::BlockstoreState::new(),
            ))),
            client: c,
        })
    }

    // ===== Raw HTTP (legacy callers that bypass typed services) =====
    #[napi]
    pub async fn api_get(&self, endpoint: String) -> napi::Result<String> {
        let v: serde_json::Value = self.client.get(&endpoint).await.map_err(err)?;
        serde_json::to_string(&v).map_err(err)
    }

    #[napi]
    pub async fn api_post(&self, endpoint: String, body: String) -> napi::Result<String> {
        let payload: serde_json::Value = if body.is_empty() {
            serde_json::Value::Null
        } else {
            serde_json::from_str(&body).map_err(err)?
        };
        let v: serde_json::Value = self.client.post(&endpoint, &payload).await.map_err(err)?;
        serde_json::to_string(&v).map_err(err)
    }

    #[napi]
    pub async fn api_put(&self, endpoint: String, body: String) -> napi::Result<String> {
        let payload: serde_json::Value = if body.is_empty() {
            serde_json::Value::Null
        } else {
            serde_json::from_str(&body).map_err(err)?
        };
        let v: serde_json::Value = self.client.put(&endpoint, &payload).await.map_err(err)?;
        serde_json::to_string(&v).map_err(err)
    }

    #[napi]
    pub async fn api_patch(&self, endpoint: String, body: String) -> napi::Result<String> {
        let payload: serde_json::Value = if body.is_empty() {
            serde_json::Value::Null
        } else {
            serde_json::from_str(&body).map_err(err)?
        };
        let v: serde_json::Value = self.client.patch(&endpoint, &payload).await.map_err(err)?;
        serde_json::to_string(&v).map_err(err)
    }

    #[napi]
    pub async fn api_delete(&self, endpoint: String) -> napi::Result<String> {
        let v: serde_json::Value = self.client.delete(&endpoint).await.map_err(err)?;
        serde_json::to_string(&v).map_err(err)
    }

    #[napi]
    pub fn api_org_path(&self, path: String) -> String {
        self.client.org_path(&path)
    }

    // ===== Auth =====
    #[napi]
    pub async fn auth_login(&self, email: String, password: String) -> napi::Result<String> {
        let session = self.auth.login(&email, &password).await.map_err(err)?;
        serde_json::to_string(&session).map_err(err)
    }

    #[napi]
    pub async fn auth_logout(&self) -> napi::Result<()> {
        self.auth.logout().await.map_err(err)
    }

    #[napi]
    pub async fn auth_refresh_token(&self) -> napi::Result<String> {
        let tokens = self.auth.refresh_token().await.map_err(err)?;
        serde_json::to_string(&tokens).map_err(err)
    }

    #[napi]
    pub async fn auth_fetch_organizations(&self) -> napi::Result<String> {
        let orgs = self.auth.fetch_organizations().await.map_err(err)?;
        serde_json::to_string(&orgs).map_err(err)
    }

    #[napi]
    pub fn auth_is_authenticated(&self) -> bool {
        self.auth.is_authenticated()
    }

    #[napi]
    pub fn auth_restore_session(&self) -> napi::Result<bool> {
        self.auth.restore_session().map_err(err)
    }

    #[napi]
    pub fn auth_switch_org(&self, slug: String) -> napi::Result<()> {
        self.auth.switch_org(&slug).map_err(err)
    }

    #[napi]
    pub fn auth_get_token(&self) -> Option<String> {
        AuthTokenStore::get_token(self.auth.as_ref())
    }

    #[napi]
    pub fn auth_get_current_user_json(&self) -> Option<String> {
        self.auth.current_user().map(|u| serde_json::to_string(&u).unwrap_or_default())
    }

    #[napi]
    pub fn auth_get_current_org_json(&self) -> Option<String> {
        self.auth.get_current_org().map(|o| serde_json::to_string(&o).unwrap_or_default())
    }

    #[napi]
    pub fn auth_get_organizations_json(&self) -> String {
        serde_json::to_string(&self.auth.get_organizations()).unwrap_or_default()
    }

    #[napi]
    pub fn auth_apply_session(&self, session_json: String) -> napi::Result<()> {
        let session: agentsmesh_types::AuthSession = serde_json::from_str(&session_json).map_err(err)?;
        self.auth.apply_session(&session);
        Ok(())
    }

    #[napi]
    pub fn auth_set_organizations(&self, orgs_json: String) -> napi::Result<()> {
        let orgs: Vec<agentsmesh_types::Organization> = serde_json::from_str(&orgs_json).map_err(err)?;
        self.auth.replace_organizations(orgs);
        Ok(())
    }

    #[napi]
    pub fn auth_set_current_org(&self, org_json: String) -> napi::Result<()> {
        if org_json.is_empty() {
            self.auth.set_current_org_direct(None);
        } else {
            let org: agentsmesh_types::Organization = serde_json::from_str(&org_json).map_err(err)?;
            self.auth.set_current_org_direct(Some(org));
        }
        Ok(())
    }

    #[napi]
    pub fn auth_clear_session(&self) {
        self.auth.clear();
    }
}

mod commands;
