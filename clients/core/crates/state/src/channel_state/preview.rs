use super::{ChannelState, PREVIEW_MAX_CHARS};
use crate::channel_types::{ChannelMessage, MessagePreview, SenderPodInfo};

impl ChannelState {
    pub fn get_last_message(&self, channel_id: i64) -> Option<&MessagePreview> {
        self.channels
            .iter()
            .find(|c| c.id == channel_id)
            .and_then(|c| c.last_message.as_ref())
    }

    /// Write a preview onto the matching Channel's `last_message` /
    /// `last_activity_at` fields (the proto-schema-derived state). Sidebar
    /// renders read directly from `Channel.last_message` — no side-channel
    /// HashMap needed.
    pub fn set_last_message(&mut self, channel_id: i64, preview: MessagePreview) {
        let ts = preview.timestamp.clone();
        if let Some(c) = self.channels.iter_mut().find(|c| c.id == channel_id) {
            c.last_activity_at = Some(ts.clone());
            c.last_message = Some(preview);
            if let Some(repo) = &self.channel_repo {
                let _ = repo.save(c);
            }
        }
        if self.current_channel.as_ref().is_some_and(|c| c.id == channel_id) {
            let preview_clone = self
                .channels
                .iter()
                .find(|c| c.id == channel_id)
                .and_then(|c| c.last_message.clone());
            if let Some(c) = self.current_channel.as_mut() {
                c.last_activity_at = Some(ts);
                c.last_message = preview_clone;
            }
        }
    }

    pub fn make_preview(msg: &ChannelMessage) -> MessagePreview {
        let sender = msg
            .sender_user
            .as_ref()
            .map(|u| u.name.as_deref().unwrap_or(&u.username).to_string())
            .or_else(|| msg.sender_pod_info.as_ref().map(pod_sender_name))
            .unwrap_or_default();

        let preview = match msg.message_type.as_deref() {
            Some("code") => "[Code]".to_string(),
            Some("command") => "[Command]".to_string(),
            _ => truncate_str(&msg.body, PREVIEW_MAX_CHARS),
        };

        MessagePreview {
            message_id: msg.id,
            sender_name: sender,
            content_preview: preview,
            message_type: msg.message_type.clone(),
            timestamp: msg.created_at.clone().unwrap_or_default(),
        }
    }
}

/// Pod sender priority matches `clients/web/src/lib/pod-display-name.ts`:
/// alias (user-defined) > agent.name > pod_key prefix. Whitespace-only
/// alias/agent values fall through to the next tier.
fn pod_sender_name(p: &SenderPodInfo) -> String {
    if let Some(alias) = p.alias.as_deref().map(str::trim).filter(|s| !s.is_empty()) {
        return alias.to_string();
    }
    if let Some(name) = p
        .agent
        .as_ref()
        .map(|a| a.name.trim())
        .filter(|s| !s.is_empty())
    {
        return name.to_string();
    }
    p.pod_key.clone()
}

fn truncate_str(s: &str, max_chars: usize) -> String {
    let chars: Vec<char> = s.chars().take(max_chars + 1).collect();
    if chars.len() > max_chars {
        let mut result: String = chars[..max_chars].iter().collect();
        result.push('…');
        result
    } else {
        s.to_string()
    }
}
