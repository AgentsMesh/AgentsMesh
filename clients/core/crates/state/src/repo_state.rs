use std::sync::Arc;

use agentsmesh_persistence::StorageBackend;
use agentsmesh_types::proto_repository_v1::{Branch, Repository};

use crate::persist_helpers::JsonStore;

pub struct RepoState {
    repositories: Vec<Repository>,
    current_repo: Option<Repository>,
    branches: Vec<Branch>,
    store: Option<JsonStore>,
}

impl RepoState {
    pub fn new() -> Self {
        Self { repositories: Vec::new(), current_repo: None, branches: Vec::new(), store: None }
    }

    pub fn with_storage(backend: Arc<dyn StorageBackend>) -> Self {
        let store = JsonStore::new(backend, "repositories");
        let repositories = store.load_all();
        Self { repositories, current_repo: None, branches: Vec::new(), store: Some(store) }
    }

    pub fn repositories(&self) -> &[Repository] { &self.repositories }
    pub fn current_repo(&self) -> Option<&Repository> { self.current_repo.as_ref() }
    pub fn branches(&self) -> &[Branch] { &self.branches }

    pub fn set_repositories(&mut self, repos: Vec<Repository>) {
        tracing::debug!(target: "repo", count = repos.len(), "set repositories (baseline)");
        self.repositories = repos;
        if let Some(store) = &self.store { store.replace_all(&self.repositories, |r| r.id.to_string()); }
    }

    pub fn set_current_repo(&mut self, repo: Option<Repository>) { self.current_repo = repo; }
    pub fn set_branches(&mut self, branches: Vec<Branch>) { self.branches = branches; }

    pub fn add_repository(&mut self, repo: Repository) {
        tracing::info!(target: "repo", repo_id = repo.id, "add repository");
        if let Some(store) = &self.store {
            store.save(&repo.id.to_string(), &repo);
        }
        self.repositories.push(repo);
    }

    pub fn update_repository(&mut self, id: &str, repo: Repository) {
        tracing::info!(target: "repo", repo_id = id, "update repository");
        if let Some(r) = self.repositories.iter_mut().find(|r| r.id.to_string() == id) {
            *r = repo.clone();
            if let Some(store) = &self.store { store.save(id, r); }
        }
        if self.current_repo.as_ref().is_some_and(|r| r.id.to_string() == id) {
            self.current_repo = Some(repo);
        }
    }

    pub fn remove_repository(&mut self, id: &str) {
        tracing::info!(target: "repo", repo_id = id, "remove repository");
        self.repositories.retain(|r| r.id.to_string() != id);
        if self.current_repo.as_ref().is_some_and(|r| r.id.to_string() == id) {
            self.current_repo = None;
        }
        if let Some(store) = &self.store { store.delete(id); }
    }
}

impl Default for RepoState {
    fn default() -> Self { Self::new() }
}
