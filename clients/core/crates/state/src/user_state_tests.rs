use agentsmesh_types::{User, UserIdentity};

use crate::user_state::UserState;

fn make_user(id: i64, email: &str) -> User {
    User { id, email: email.to_string(), username: format!("user{id}"), name: None, avatar_url: None, is_email_verified: None }
}

fn make_identity(id: i64, provider: &str) -> UserIdentity {
    UserIdentity { id, provider: provider.to_string(), provider_user_id: None, provider_username: None, created_at: None }
}

#[test]
fn new_is_empty() {
    let s = UserState::new();
    assert!(s.profile().is_none());
    assert!(s.identities().is_empty());
}

#[test]
fn set_and_get_profile() {
    let mut s = UserState::new();
    s.set_profile(Some(make_user(1, "a@b.com")));
    let p = s.profile().unwrap();
    assert_eq!(p.email, "a@b.com");
}

#[test]
fn set_profile_to_none() {
    let mut s = UserState::new();
    s.set_profile(Some(make_user(1, "a@b.com")));
    s.set_profile(None);
    assert!(s.profile().is_none());
}

#[test]
fn set_profile_replaces_previous() {
    let mut s = UserState::new();
    let mut u = make_user(1, "a@b.com");
    u.name = Some("Old".to_string());
    s.set_profile(Some(u));
    let mut u2 = make_user(1, "a@b.com");
    u2.name = Some("New".to_string());
    s.set_profile(Some(u2));
    assert_eq!(s.profile().unwrap().name.as_deref(), Some("New"));
}

#[test]
fn add_identity() {
    let mut s = UserState::new();
    s.add_identity(make_identity(1, "github"));
    s.add_identity(make_identity(2, "gitlab"));
    assert_eq!(s.identities().len(), 2);
    assert_eq!(s.identities()[0].provider, "github");
}

#[test]
fn remove_identity() {
    let mut s = UserState::new();
    s.add_identity(make_identity(1, "github"));
    s.add_identity(make_identity(2, "gitlab"));
    s.remove_identity("1");
    assert_eq!(s.identities().len(), 1);
    assert_eq!(s.identities()[0].provider, "gitlab");
}

#[test]
fn remove_identity_nonexistent_is_noop() {
    let mut s = UserState::new();
    s.add_identity(make_identity(1, "github"));
    s.remove_identity("999");
    assert_eq!(s.identities().len(), 1);
}
