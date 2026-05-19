use agentsmesh_types::{Organization, User};
use serde::{Deserialize, Serialize};

pub(crate) const LEGACY_STORAGE_KEY: &str = "agentsmesh-auth";
pub(crate) const NAMESPACE_PREFIX: &str = "agentsmesh-auth";
pub(crate) const SCHEMA_VERSION: u32 = 1;

/// Only this struct hits disk / localStorage / Keychain. Crate-private:
/// the single supported way to read/write it is through `AuthManager`.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub(crate) struct PersistedSession {
    pub(crate) access_token: String,
    pub(crate) refresh_token: String,
    #[serde(default)]
    pub(crate) expires_at: i64,
    #[serde(default)]
    pub(crate) base_url: String,
    #[serde(default)]
    pub(crate) current_org_slug: Option<String>,
    #[serde(default)]
    pub(crate) schema_version: u32,
}

#[derive(Debug, Clone, Default)]
pub(crate) struct AuthState {
    pub(crate) session: Option<PersistedSession>,
    pub(crate) user: Option<User>,
    pub(crate) current_org: Option<Organization>,
    pub(crate) organizations: Vec<Organization>,
}

impl AuthState {
    pub(crate) fn apply_session(
        &mut self,
        session: &agentsmesh_types::AuthSession,
        base_url: &str,
        now_secs: i64,
    ) {
        let expires_at = now_secs + session.expires_in.unwrap_or(3600);
        let current_org_slug = self
            .session
            .as_ref()
            .and_then(|s| s.current_org_slug.clone())
            .or_else(|| self.current_org.as_ref().map(|o| o.slug.clone()));
        self.session = Some(PersistedSession {
            access_token: session.token.clone(),
            refresh_token: session.refresh_token.clone(),
            expires_at,
            base_url: base_url.to_string(),
            current_org_slug,
            schema_version: SCHEMA_VERSION,
        });
        self.user = Some(session.user.clone());
    }

    pub(crate) fn apply_tokens(
        &mut self,
        tokens: &agentsmesh_types::AuthTokens,
        base_url: &str,
        now_secs: i64,
    ) {
        let expires_at = now_secs + tokens.expires_in.unwrap_or(3600);
        match self.session.as_mut() {
            Some(s) => {
                s.access_token = tokens.token.clone();
                s.refresh_token = tokens.refresh_token.clone();
                s.expires_at = expires_at;
                if s.base_url.is_empty() {
                    s.base_url = base_url.to_string();
                }
                s.schema_version = SCHEMA_VERSION;
            }
            None => {
                self.session = Some(PersistedSession {
                    access_token: tokens.token.clone(),
                    refresh_token: tokens.refresh_token.clone(),
                    expires_at,
                    base_url: base_url.to_string(),
                    current_org_slug: None,
                    schema_version: SCHEMA_VERSION,
                });
            }
        }
    }

    pub(crate) fn restore_persisted(&mut self, persisted: PersistedSession) {
        self.session = Some(persisted);
    }

    pub(crate) fn set_user(&mut self, user: User) {
        self.user = Some(user);
    }

    pub(crate) fn clear(&mut self) {
        *self = Self::default();
    }

    pub(crate) fn token(&self) -> Option<&str> {
        self.session.as_ref().map(|s| s.access_token.as_str())
    }

    pub(crate) fn refresh_token(&self) -> Option<&str> {
        self.session.as_ref().map(|s| s.refresh_token.as_str())
    }
}

pub(crate) fn url_slug(base_url: &str) -> String {
    let trimmed = base_url.trim_end_matches('/');
    let (scheme, rest) = match trimmed.find("://") {
        Some(i) => (&trimmed[..i], &trimmed[i + 3..]),
        None => ("", trimmed),
    };
    let authority = rest.split(['/', '?', '#']).next().unwrap_or(rest);
    let normalized = if scheme.is_empty() {
        authority.to_lowercase()
    } else {
        format!("{}_{}", scheme.to_lowercase(), authority.to_lowercase())
    };
    normalized
        .chars()
        .map(|c| if c.is_ascii_alphanumeric() { c } else { '_' })
        .take(64)
        .collect()
}

pub(crate) fn session_storage_key(base_url: &str) -> String {
    format!("{}/{}/session", NAMESPACE_PREFIX, url_slug(base_url))
}
