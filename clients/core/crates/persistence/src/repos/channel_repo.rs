use std::sync::Arc;

use agentsmesh_types::Channel;

use crate::backend::StorageBackend;
use crate::error::Result;

pub struct ChannelRepo {
    backend: Arc<dyn StorageBackend>,
}

impl ChannelRepo {
    pub fn new(backend: Arc<dyn StorageBackend>) -> Self {
        Self { backend }
    }

    pub fn get(&self, id: i64) -> Result<Option<Channel>> {
        match self.backend.get_raw("channels", &id.to_string())? {
            Some(data) => Ok(Some(serde_json::from_slice(&data)?)),
            None => Ok(None),
        }
    }

    pub fn save(&self, channel: &Channel) -> Result<()> {
        let data = serde_json::to_vec(channel)?;
        self.backend
            .put_raw("channels", &channel.id.to_string(), &[], &data)
    }

    pub fn delete(&self, id: i64) -> Result<()> {
        self.backend.delete_raw("channels", &id.to_string())
    }

    pub fn list_all(&self) -> Result<Vec<Channel>> {
        super::deserialize_rows(self.backend.list_raw("channels")?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::backend::InMemoryBackend;

    fn make_repo() -> ChannelRepo {
        ChannelRepo::new(Arc::new(InMemoryBackend::new()))
    }

    fn make_channel(id: i64, name: &str) -> Channel {
        Channel {
            id,
            name: name.into(),
            description: None,
            is_archived: false,
            visibility: None,
            is_member: false,
            member_count: None,
            agent_count: None,
            organization_id: None,
            document: None,
            repository_id: None,
            ticket_id: None,
            ticket_slug: None,
            created_by_pod: None,
            created_by_user_id: None,
            created_at: None,
            updated_at: None,
        }
    }

    #[test]
    fn crud_roundtrip() {
        let repo = make_repo();
        repo.save(&make_channel(1, "general")).unwrap();
        let loaded = repo.get(1).unwrap().unwrap();
        assert_eq!(loaded.name, "general");
        repo.delete(1).unwrap();
        assert!(repo.get(1).unwrap().is_none());
    }

    #[test]
    fn list_all() {
        let repo = make_repo();
        repo.save(&make_channel(1, "a")).unwrap();
        repo.save(&make_channel(2, "b")).unwrap();
        assert_eq!(repo.list_all().unwrap().len(), 2);
    }

    #[test]
    fn get_nonexistent_returns_none() {
        let repo = make_repo();
        assert!(repo.get(999).unwrap().is_none());
    }

    #[test]
    fn delete_nonexistent_is_noop() {
        let repo = make_repo();
        repo.delete(999).unwrap();
    }

    #[test]
    fn save_overwrites_existing() {
        let repo = make_repo();
        repo.save(&make_channel(1, "old")).unwrap();
        repo.save(&make_channel(1, "new")).unwrap();
        let loaded = repo.get(1).unwrap().unwrap();
        assert_eq!(loaded.name, "new");
        assert_eq!(repo.list_all().unwrap().len(), 1);
    }

    #[test]
    fn list_empty_table() {
        let repo = make_repo();
        assert!(repo.list_all().unwrap().is_empty());
    }
}
