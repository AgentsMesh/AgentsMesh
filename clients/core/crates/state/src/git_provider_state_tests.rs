use agentsmesh_types::{ProviderRepository, RepositoryProvider};

use crate::git_provider_state::GitProviderState;

fn make_provider(id: i64, ptype: &str) -> RepositoryProvider {
    RepositoryProvider {
        id,
        provider_type: ptype.to_string(),
        name: format!("provider-{id}"),
        base_url: None,
        has_client_id: None,
        has_bot_token: None,
        has_identity: None,
        is_default: None,
        is_active: None,
        created_at: None,
        updated_at: None,
    }
}

fn make_project(id: &str) -> ProviderRepository {
    ProviderRepository {
        id: Some(id.to_string()),
        name: format!("project-{id}"),
        slug: None,
        description: None,
        default_branch: None,
        visibility: None,
        http_clone_url: None,
        ssh_clone_url: None,
        web_url: None,
    }
}

#[test]
fn new_is_empty() {
    let s = GitProviderState::new();
    assert!(s.providers().is_empty());
    assert!(s.current_provider().is_none());
    assert!(s.available_projects().is_empty());
}

#[test]
fn set_and_get_providers() {
    let mut s = GitProviderState::new();
    s.set_providers(vec![make_provider(1, "github")]);
    assert_eq!(s.providers().len(), 1);
    assert_eq!(s.providers()[0].provider_type, "github");
}

#[test]
fn set_providers_replaces_previous() {
    let mut s = GitProviderState::new();
    s.set_providers(vec![make_provider(1, "github")]);
    s.set_providers(vec![make_provider(2, "gitlab"), make_provider(3, "gitee")]);
    assert_eq!(s.providers().len(), 2);
    assert_eq!(s.providers()[0].id, 2);
}

#[test]
fn set_current_provider() {
    let mut s = GitProviderState::new();
    s.set_current_provider(Some(make_provider(1, "github")));
    assert_eq!(s.current_provider().unwrap().id, 1);
    s.set_current_provider(None);
    assert!(s.current_provider().is_none());
}

#[test]
fn set_available_projects() {
    let mut s = GitProviderState::new();
    s.set_available_projects(vec![make_project("p1"), make_project("p2")]);
    assert_eq!(s.available_projects().len(), 2);
}

#[test]
fn add_provider() {
    let mut s = GitProviderState::new();
    s.add_provider(make_provider(1, "github"));
    s.add_provider(make_provider(2, "gitlab"));
    assert_eq!(s.providers().len(), 2);
}

#[test]
fn update_provider() {
    let mut s = GitProviderState::new();
    s.set_providers(vec![make_provider(1, "github")]);
    let mut updated = make_provider(1, "github");
    updated.name = "updated-name".to_string();
    s.update_provider("1", updated);
    assert_eq!(s.providers()[0].name, "updated-name");
}

#[test]
fn update_provider_nonexistent_is_noop() {
    let mut s = GitProviderState::new();
    s.set_providers(vec![make_provider(1, "github")]);
    s.update_provider("999", make_provider(999, "gitlab"));
    assert_eq!(s.providers()[0].name, "provider-1");
}

#[test]
fn remove_provider() {
    let mut s = GitProviderState::new();
    s.set_providers(vec![make_provider(1, "github"), make_provider(2, "gitlab")]);
    s.remove_provider("1");
    assert_eq!(s.providers().len(), 1);
    assert_eq!(s.providers()[0].id, 2);
}

#[test]
fn remove_provider_clears_current_if_same() {
    let mut s = GitProviderState::new();
    s.set_providers(vec![make_provider(1, "github")]);
    s.set_current_provider(Some(make_provider(1, "github")));
    s.remove_provider("1");
    assert!(s.current_provider().is_none());
}

#[test]
fn remove_provider_keeps_current_if_different() {
    let mut s = GitProviderState::new();
    s.set_providers(vec![make_provider(1, "github"), make_provider(2, "gitlab")]);
    s.set_current_provider(Some(make_provider(2, "gitlab")));
    s.remove_provider("1");
    assert_eq!(s.current_provider().unwrap().id, 2);
}

#[test]
fn remove_provider_nonexistent_is_noop() {
    let mut s = GitProviderState::new();
    s.set_providers(vec![make_provider(1, "github")]);
    s.remove_provider("999");
    assert_eq!(s.providers().len(), 1);
}
