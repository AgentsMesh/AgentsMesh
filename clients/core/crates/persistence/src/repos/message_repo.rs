use std::sync::Arc;

use agentsmesh_types::ChannelMessage;
use crate::backend::StorageBackend;
use crate::error::Result;

pub struct MessageRepo {
    backend: Arc<dyn StorageBackend>,
}

impl MessageRepo {
    pub fn new(backend: Arc<dyn StorageBackend>) -> Self {
        Self { backend }
    }

    pub fn save_message(&self, msg: &ChannelMessage) -> Result<()> {
        let data = serde_json::to_vec(msg)?;
        let channel_id = msg.channel_id.to_string();
        let msg_id = msg.id.to_string();
        let indexed = vec![
            ("channel_id".to_string(), channel_id),
            ("msg_id".to_string(), msg_id),
        ];
        self.backend.put_raw("channel_messages", &msg.id.to_string(), 
            &indexed.iter().map(|(k,v)| (k.as_str(), v.as_str())).collect::<Vec<_>>(), &data)
    }

    pub fn get_by_channel(&self, channel_id: i64, limit: usize, before_id: Option<i64>) -> Result<Vec<ChannelMessage>> {
        let rows = self.backend.query_raw("channel_messages", "channel_id", &channel_id.to_string())?;
        let mut msgs: Vec<ChannelMessage> = rows.iter()
            .filter_map(|(_, data)| serde_json::from_slice(data).ok())
            .collect();
        msgs.sort_by(|a, b| b.id.cmp(&a.id));
        if let Some(bid) = before_id {
            msgs.retain(|m| m.id < bid);
        }
        msgs.truncate(limit);
        Ok(msgs)
    }

    pub fn delete_message(&self, message_id: i64) -> Result<()> {
        self.backend.delete_raw("channel_messages", &message_id.to_string())
    }

    pub fn clear_channel(&self, channel_id: i64) -> Result<()> {
        let rows = self.backend.query_raw("channel_messages", "channel_id", &channel_id.to_string())?;
        for (id, _) in rows {
            self.backend.delete_raw("channel_messages", &id)?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::backend::InMemoryBackend;

    fn make_msg(id: i64, channel_id: i64, content: &str) -> ChannelMessage {
        ChannelMessage {
            id, channel_id, body: content.into(),
            content: None, mentions: None, reply_to: None,
            sender_user: None, sender_user_id: None, sender_pod: None, sender_pod_info: None,
            message_type: None, pod_key: None, metadata: None, edited_at: None, is_deleted: None, created_at: None,
        }
    }

    #[test]
    fn save_and_get_by_channel() {
        let backend = Arc::new(InMemoryBackend::new());
        backend.migrate().unwrap();
        let repo = MessageRepo::new(backend);

        repo.save_message(&make_msg(1, 10, "hello")).unwrap();
        repo.save_message(&make_msg(2, 10, "world")).unwrap();
        repo.save_message(&make_msg(3, 20, "other channel")).unwrap();

        let msgs = repo.get_by_channel(10, 50, None).unwrap();
        assert_eq!(msgs.len(), 2);
        assert_eq!(msgs[0].id, 2); // newest first
    }

    #[test]
    fn get_by_channel_with_limit() {
        let backend = Arc::new(InMemoryBackend::new());
        backend.migrate().unwrap();
        let repo = MessageRepo::new(backend);

        for i in 1..=10 {
            repo.save_message(&make_msg(i, 10, &format!("msg {i}"))).unwrap();
        }

        let msgs = repo.get_by_channel(10, 3, None).unwrap();
        assert_eq!(msgs.len(), 3);
    }

    #[test]
    fn delete_message() {
        let backend = Arc::new(InMemoryBackend::new());
        backend.migrate().unwrap();
        let repo = MessageRepo::new(backend);

        repo.save_message(&make_msg(1, 10, "hi")).unwrap();
        repo.delete_message(1).unwrap();
        assert!(repo.get_by_channel(10, 50, None).unwrap().is_empty());
    }

    #[test]
    fn empty_channel_returns_empty() {
        let backend = Arc::new(InMemoryBackend::new());
        backend.migrate().unwrap();
        let repo = MessageRepo::new(backend);
        assert!(repo.get_by_channel(999, 50, None).unwrap().is_empty());
    }

    #[test]
    fn clear_channel_removes_all() {
        let backend = Arc::new(InMemoryBackend::new());
        backend.migrate().unwrap();
        let repo = MessageRepo::new(backend);

        repo.save_message(&make_msg(1, 10, "a")).unwrap();
        repo.save_message(&make_msg(2, 10, "b")).unwrap();
        repo.clear_channel(10).unwrap();
        assert!(repo.get_by_channel(10, 50, None).unwrap().is_empty());
    }

    #[test]
    fn delete_nonexistent_message_is_noop() {
        let backend = Arc::new(InMemoryBackend::new());
        backend.migrate().unwrap();
        let repo = MessageRepo::new(backend);
        repo.delete_message(999).unwrap();
    }

    #[test]
    fn messages_sorted_newest_first() {
        let backend = Arc::new(InMemoryBackend::new());
        backend.migrate().unwrap();
        let repo = MessageRepo::new(backend);

        for i in 1..=5 {
            repo.save_message(&make_msg(i, 1, &format!("msg{i}"))).unwrap();
        }
        let msgs = repo.get_by_channel(1, 50, None).unwrap();
        for w in msgs.windows(2) {
            assert!(w[0].id > w[1].id);
        }
    }

    #[test]
    fn get_by_channel_with_before_id() {
        let backend = Arc::new(InMemoryBackend::new());
        backend.migrate().unwrap();
        let repo = MessageRepo::new(backend);

        for i in 1..=10 {
            repo.save_message(&make_msg(i, 1, &format!("msg{i}"))).unwrap();
        }
        let msgs = repo.get_by_channel(1, 50, Some(6)).unwrap();
        assert_eq!(msgs.len(), 5);
        assert_eq!(msgs[0].id, 5); // newest first among those < 6
        assert_eq!(msgs[4].id, 1);

        let msgs = repo.get_by_channel(1, 2, Some(6)).unwrap();
        assert_eq!(msgs.len(), 2);
        assert_eq!(msgs[0].id, 5);
        assert_eq!(msgs[1].id, 4);
    }
}
