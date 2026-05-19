pub trait AuthTokenStore: Send + Sync {
    fn get_token(&self) -> Option<String>;
    fn get_refresh_token(&self) -> Option<String>;
    fn set_tokens(&self, token: String, refresh_token: String, expires_in_secs: Option<i64>);
    fn clear_tokens(&self);
    fn get_current_org_slug(&self) -> Option<String>;
}
