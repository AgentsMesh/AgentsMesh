use std::collections::HashMap;
use crate::channel_state::{ChannelSortMode, ChannelState};
use crate::channel_types::{Channel, ChannelMessage, SenderAgentInfo, SenderPodInfo, User};

fn ch(id: i64, name: &str) -> Channel {
    Channel { id, name: name.into(), ..Default::default() }
}
fn msg(id: i64, channel: i64, content: &str) -> ChannelMessage {
    ChannelMessage { id, channel_id: channel, body: content.into(), ..Default::default() }
}
fn msg_with_sender(id: i64, ch: i64, content: &str, user_id: i64, username: &str) -> ChannelMessage {
    let mut m = msg(id, ch, content);
    m.sender_user_id = Some(user_id);
    m.sender_user = Some(user(user_id, username));
    m.created_at = Some(format!("2026-01-01T00:00:{:02}Z", id));
    m
}
fn msg_with_time(id: i64, ch: i64, content: &str, time: &str) -> ChannelMessage {
    let mut m = msg(id, ch, content);
    m.created_at = Some(time.into());
    m
}

// ── Basic operations (existing) ──

#[test] fn new_state() { let s = ChannelState::new(); assert!(s.get_channels().is_empty()); assert!(s.get_current_channel().is_none()); }
#[test] fn set_channels() { let mut s = ChannelState::new(); s.set_channels(vec![ch(1,"gen"), ch(2,"dev")]); assert_eq!(s.get_channels().len(), 2); }
#[test] fn set_current_channel() { let mut s = ChannelState::new(); s.set_channels(vec![ch(1,"gen")]); s.set_current_channel(Some(1)); assert_eq!(s.get_current_channel().unwrap().name, "gen"); s.set_current_channel(Some(99)); assert!(s.get_current_channel().is_none()); }
#[test] fn add_message() { let mut s = ChannelState::new(); s.add_message(1, msg(100,1,"hi")); assert_eq!(s.get_messages(1).unwrap().messages.len(), 1); }
#[test] fn add_message_dedup() { let mut s = ChannelState::new(); s.add_message(1, msg(100,1,"hi")); s.add_message(1, msg(100,1,"dup")); assert_eq!(s.get_messages(1).unwrap().messages.len(), 1); }
#[test] fn add_message_returns_true_for_new() { let mut s = ChannelState::new(); assert!(s.add_message(1, msg(100,1,"hi"))); assert!(!s.add_message(1, msg(100,1,"dup"))); }
#[test] fn update_message() { let mut s = ChannelState::new(); s.add_message(1, msg(100,1,"old")); s.update_message(1, msg(100,1,"new")); assert_eq!(s.get_messages(1).unwrap().messages[0].body, "new"); }
#[test] fn update_message_no_cache() { let mut s = ChannelState::new(); s.update_message(1, msg(100,1,"x")); assert!(s.get_messages(1).is_none()); }
#[test] fn remove_message() { let mut s = ChannelState::new(); s.add_message(1, msg(100,1,"a")); s.add_message(1, msg(101,1,"b")); s.remove_message(1, 100); assert_eq!(s.get_messages(1).unwrap().messages.len(), 1); }
#[test] fn set_messages() { let mut s = ChannelState::new(); s.set_messages(1, vec![msg(1,1,"a"), msg(2,1,"b")], true); let c = s.get_messages(1).unwrap(); assert_eq!(c.messages.len(), 2); assert!(c.has_more); }
#[test] fn unread_counts() { let mut s = ChannelState::new(); s.set_channels(vec![ch(1,"a")]); assert_eq!(s.get_unread_count(1), 0); s.increment_unread(1); s.increment_unread(1); assert_eq!(s.get_unread_count(1), 2); s.clear_channel_unread(1); assert_eq!(s.get_unread_count(1), 0); }
#[test] fn set_unread_counts() { let mut s = ChannelState::new(); s.set_channels(vec![ch(1,"a"), ch(2,"b")]); let mut c = HashMap::new(); c.insert(1,5); c.insert(2,3); s.set_unread_counts(c); assert_eq!(s.get_unread_count(1), 5); assert_eq!(s.get_unread_count(2), 3); }
#[test] fn get_messages_no_cache() { let s = ChannelState::new(); assert!(s.get_messages(99).is_none()); }
#[test] fn default_impl() { let s = ChannelState::default(); assert!(s.get_channels().is_empty()); }

// ── on_new_message ──

#[test]
fn on_new_message_updates_preview() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    let m = msg_with_sender(1, 1, "Hello world", 10, "alice");
    s.on_new_message(m);
    let preview = s.get_last_message(1).unwrap();
    assert_eq!(preview.sender_name, "alice");
    assert_eq!(preview.content_preview, "Hello world");
}

#[test]
fn on_new_message_increments_unread_for_other_user() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user_id(Some(10));
    // Message from user 20 (not me), not viewing the channel → Rust Core
    // auto-increments unread (business logic now lives here, not the handler).
    s.on_new_message(msg_with_sender(1, 1, "hi", 20, "bob"));
    assert_eq!(s.get_unread_count(1), 1);
}

#[test]
fn on_new_message_skips_unread_for_own_message() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user_id(Some(10));
    // Message from user 10 (me)
    s.on_new_message(msg_with_sender(1, 1, "my msg", 10, "me"));
    assert_eq!(s.get_unread_count(1), 0);
}

#[test]
fn on_new_message_skips_unread_for_current_channel() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user_id(Some(10));
    s.set_current_channel(Some(1)); // viewing channel 1
    // Message from another user in the channel I'm viewing
    s.on_new_message(msg_with_sender(1, 1, "hi", 20, "bob"));
    assert_eq!(s.get_unread_count(1), 0);
}

#[test]
fn on_new_message_increments_unread_for_other_channel() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen"), ch(2, "dev")]);
    s.set_current_user_id(Some(10));
    s.set_current_channel(Some(1)); // viewing channel 1
    // Message in channel 2 (not current, from another user) → auto-increment.
    s.on_new_message(msg_with_sender(1, 2, "hi", 20, "bob"));
    assert_eq!(s.get_unread_count(2), 1);
}

#[test]
fn on_new_message_no_unread_without_current_user_known() {
    // Defensive: if current_user is unset, every message counts as "other"
    // and increments unread — matches the pre-SSOT JS fallback (it read a
    // possibly-undefined user and still incremented when not viewing).
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.on_new_message(msg_with_sender(1, 1, "hi", 20, "bob"));
    assert_eq!(s.get_unread_count(1), 1);
}

#[test]
fn on_new_message_dedup_does_not_double_count_unread() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user_id(Some(10));
    s.on_new_message(msg_with_sender(1, 1, "hi", 20, "bob"));
    s.on_new_message(msg_with_sender(1, 1, "dup", 20, "bob")); // same id → dup
    assert_eq!(s.get_unread_count(1), 1, "dup must not re-increment");
}

#[test]
fn on_new_message_increments_mention_when_user_mentioned() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user_id(Some(10));
    let mut m = msg_with_sender(1, 1, "hey @me", 20, "bob");
    m.mentions_json = Some(r#"{"users":[10],"channel":false}"#.into());
    s.on_new_message(m);
    assert_eq!(s.get_unread_count(1), 1);
    assert_eq!(s.get_mention_count(1), 1);
}

#[test]
fn on_new_message_channel_mention_counts_as_mention() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user_id(Some(10));
    let mut m = msg_with_sender(1, 1, "@channel heads up", 20, "bob");
    m.mentions_json = Some(r#"{"channel":true}"#.into());
    s.on_new_message(m);
    assert_eq!(s.get_mention_count(1), 1);
}

#[test]
fn on_new_message_no_mention_when_other_user_mentioned() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user_id(Some(10));
    let mut m = msg_with_sender(1, 1, "hey @someone", 20, "bob");
    m.mentions_json = Some(r#"{"users":[99],"channel":false}"#.into());
    s.on_new_message(m);
    assert_eq!(s.get_unread_count(1), 1);
    assert_eq!(s.get_mention_count(1), 0);
}

#[test]
fn on_new_message_mention_skipped_when_viewing_channel() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user_id(Some(10));
    s.set_current_channel(Some(1)); // actively viewing
    let mut m = msg_with_sender(1, 1, "hey @me", 20, "bob");
    m.mentions_json = Some(r#"{"users":[10]}"#.into());
    s.on_new_message(m);
    assert_eq!(s.get_unread_count(1), 0);
    assert_eq!(s.get_mention_count(1), 0);
}

#[test]
fn on_new_message_deduplicates() {
    let mut s = ChannelState::new();
    assert!(s.on_new_message(msg(1, 1, "first")));
    assert!(!s.on_new_message(msg(1, 1, "dup")));
    assert_eq!(s.get_messages(1).unwrap().messages.len(), 1);
}

// ── Channel sorting ──

#[test]
fn sorted_by_last_message() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "old"), ch(2, "new"), ch(3, "newest")]);
    // Set last messages with different timestamps
    s.on_new_message(msg_with_time(1, 1, "a", "2026-01-01T00:00:00Z"));
    s.on_new_message(msg_with_time(2, 2, "b", "2026-01-01T00:00:02Z"));
    s.on_new_message(msg_with_time(3, 3, "c", "2026-01-01T00:00:01Z"));
    let ids = s.sorted_channel_ids(ChannelSortMode::LastMessage, false);
    assert_eq!(ids, vec![2, 3, 1]); // newest first
}

#[test]
fn sorted_excludes_archived() {
    let mut s = ChannelState::new();
    let mut archived = ch(2, "archived");
    archived.is_archived = true;
    s.set_channels(vec![ch(1, "active"), archived]);
    let ids = s.sorted_channel_ids(ChannelSortMode::Name, false);
    assert_eq!(ids, vec![1]);
    let ids_all = s.sorted_channel_ids(ChannelSortMode::Name, true);
    assert_eq!(ids_all.len(), 2);
}

#[test]
fn sorted_by_name() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "Zebra"), ch(2, "alpha"), ch(3, "Beta")]);
    let ids = s.sorted_channel_ids(ChannelSortMode::Name, true);
    assert_eq!(ids, vec![2, 3, 1]); // alpha, Beta, Zebra (case-insensitive)
}

#[test]
fn sorted_unread_first() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "no-unread"), ch(2, "has-unread"), ch(3, "also-unread")]);
    s.increment_unread(2);
    s.increment_unread(3);
    // Give channel 3 an older last message than channel 2
    s.on_new_message(msg_with_time(1, 3, "older", "2026-01-01T00:00:00Z"));
    s.on_new_message(msg_with_time(2, 2, "newer", "2026-01-01T00:00:01Z"));
    let ids = s.sorted_channel_ids(ChannelSortMode::UnreadFirst, true);
    // Unread channels first (2 before 3 because newer), then no-unread
    assert_eq!(ids[0], 2);
    assert_eq!(ids[1], 3);
    assert_eq!(ids[2], 1);
}

#[test]
fn sorted_channels_without_messages_come_last() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "no-msg"), ch(2, "has-msg")]);
    s.on_new_message(msg_with_time(1, 2, "hi", "2026-01-01T00:00:00Z"));
    let ids = s.sorted_channel_ids(ChannelSortMode::LastMessage, true);
    assert_eq!(ids, vec![2, 1]);
}

// ── Last message preview ──

#[test]
fn set_messages_updates_preview() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_messages(1, vec![
        msg_with_time(1, 1, "first", "2026-01-01T00:00:00Z"),
        msg_with_time(2, 1, "last", "2026-01-01T00:00:01Z"),
    ], false);
    let preview = s.get_last_message(1).unwrap();
    assert_eq!(preview.content_preview, "last");
}

#[test]
fn preview_truncates_long_content() {
    let long = "a".repeat(200);
    let m = msg_with_time(1, 1, &long, "2026-01-01T00:00:00Z");
    let preview = ChannelState::make_preview(&m);
    assert!(preview.content_preview.chars().count() <= 81); // 80 + '…'
    assert!(preview.content_preview.ends_with('…'));
}

#[test]
fn preview_code_message_type() {
    let mut m = msg(1, 1, "fn main() {}");
    m.message_type = Some("code".into());
    let preview = ChannelState::make_preview(&m);
    assert_eq!(preview.content_preview, "[Code]");
}

#[test]
fn preview_command_message_type() {
    let mut m = msg(1, 1, "/deploy prod");
    m.message_type = Some("command".into());
    let preview = ChannelState::make_preview(&m);
    assert_eq!(preview.content_preview, "[Command]");
}

#[test]
fn preview_sender_from_pod_info() {
    let mut m = msg(1, 1, "hi");
    m.sender_pod_info = Some(SenderPodInfo {
        pod_key: "pod-abc".into(),
        alias: Some("my-agent".into()),
        agent: Some(SenderAgentInfo { name: "claude".into(), ..Default::default() }),
    });
    let preview = ChannelState::make_preview(&m);
    assert_eq!(preview.sender_name, "my-agent");
}

#[test]
fn preview_sender_from_pod_agent_when_no_alias() {
    let mut m = msg(1, 1, "hi");
    m.sender_pod_info = Some(SenderPodInfo {
        pod_key: "pod-abc".into(),
        alias: None,
        agent: Some(SenderAgentInfo { name: "claude".into(), ..Default::default() }),
    });
    let preview = ChannelState::make_preview(&m);
    assert_eq!(preview.sender_name, "claude");
}

#[test]
fn preview_sender_from_pod_alias_fallback() {
    let mut m = msg(1, 1, "hi");
    m.sender_pod_info = Some(SenderPodInfo {
        pod_key: "pod-abc".into(),
        alias: Some("my-agent".into()),
        agent: None,
    });
    let preview = ChannelState::make_preview(&m);
    assert_eq!(preview.sender_name, "my-agent");
}

#[test]
fn preview_sender_from_pod_key_fallback() {
    let mut m = msg(1, 1, "hi");
    m.sender_pod_info = Some(SenderPodInfo {
        pod_key: "pod-abc".into(),
        alias: None,
        agent: None,
    });
    let preview = ChannelState::make_preview(&m);
    assert_eq!(preview.sender_name, "pod-abc");
}

// ── prepend_messages ──

#[test]
fn prepend_messages_merges_and_deduplicates() {
    let mut s = ChannelState::new();
    // Existing: messages 5, 6
    s.set_messages(1, vec![msg(5, 1, "e"), msg(6, 1, "f")], true);
    // Prepend: messages 3, 4, 5 (5 is duplicate)
    s.prepend_messages(1, vec![msg(3, 1, "c"), msg(4, 1, "d"), msg(5, 1, "dup")], false);
    let cache = s.get_messages(1).unwrap();
    assert_eq!(cache.messages.len(), 4); // 3, 4, 5, 6
    let ids: Vec<i64> = cache.messages.iter().map(|m| m.id).collect();
    assert_eq!(ids, vec![3, 4, 5, 6]);
    assert!(!cache.has_more);
}

#[test]
fn prepend_messages_maintains_ascending_order() {
    let mut s = ChannelState::new();
    s.set_messages(1, vec![msg(10, 1, "j")], true);
    s.prepend_messages(1, vec![msg(7, 1, "g"), msg(9, 1, "i"), msg(8, 1, "h")], true);
    let ids: Vec<i64> = s.get_messages(1).unwrap().messages.iter().map(|m| m.id).collect();
    assert_eq!(ids, vec![7, 8, 9, 10]);
}

#[test]
fn prepend_messages_to_empty_cache() {
    let mut s = ChannelState::new();
    s.prepend_messages(1, vec![msg(1, 1, "a"), msg(2, 1, "b")], true);
    let cache = s.get_messages(1).unwrap();
    assert_eq!(cache.messages.len(), 2);
    assert!(cache.has_more);
}

// ── Mention counts ──

#[test]
fn mention_count_tracking() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1,"a"), ch(2,"b")]);
    assert_eq!(s.get_mention_count(1), 0);
    s.increment_mention(1);
    s.increment_mention(1);
    s.increment_mention(2);
    assert_eq!(s.get_mention_count(1), 2);
    assert_eq!(s.get_mention_count(2), 1);
    s.clear_channel_mentions(1);
    assert_eq!(s.get_mention_count(1), 0);
    assert_eq!(s.get_mention_count(2), 1);
}

#[test]
fn total_unread_and_mention_counts() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "a"), ch(2, "b"), ch(3, "c")]);
    s.increment_unread(1);
    s.increment_unread(1);
    s.increment_unread(2);
    assert_eq!(s.total_unread_count(), 3);

    s.increment_mention(1);
    s.increment_mention(3);
    assert_eq!(s.total_mention_count(), 2);
}

#[test]
fn set_mention_counts() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "a"), ch(2, "b")]);
    let mut c = HashMap::new();
    c.insert(1, 3);
    c.insert(2, 7);
    s.set_mention_counts(c);
    assert_eq!(s.get_mention_count(1), 3);
    assert_eq!(s.get_mention_count(2), 7);
    assert_eq!(s.total_mention_count(), 10);
}

#[test]
fn total_counts_scoped_to_loaded_channels() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "in-org")]);
    s.increment_unread(1);
    s.increment_unread(2);
    s.increment_mention(1);
    s.increment_mention(2);
    // Channel 2 isn't in the loaded set — total skips it.
    assert_eq!(s.total_unread_count(), 1);
    assert_eq!(s.total_mention_count(), 1);
}

// ── Current user ──

#[test]
fn current_user_id() {
    let mut s = ChannelState::new();
    assert!(s.current_user_id().is_none());
    s.set_current_user_id(Some(42));
    assert_eq!(s.current_user_id(), Some(42));
    s.set_current_user_id(None);
    assert!(s.current_user_id().is_none());
}

// ── Single channel CRUD ──

fn user(id: i64, username: &str) -> User {
    User {
        id,
        email: format!("{username}@test.com"),
        username: username.into(),
        name: Some(username.into()),
        avatar_url: None,
        is_email_verified: None,
    }
}

#[test]
fn get_channel() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen"), ch(2, "dev")]);
    assert_eq!(s.get_channel(1).unwrap().name, "gen");
    assert_eq!(s.get_channel(2).unwrap().name, "dev");
    assert!(s.get_channel(99).is_none());
}

#[test]
fn add_channel_prepends() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.add_channel(ch(2, "dev"));
    assert_eq!(s.get_channels().len(), 2);
    assert_eq!(s.get_channels()[0].name, "dev"); // prepended
}

#[test]
fn add_channel_dedup() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.add_channel(ch(1, "dup"));
    assert_eq!(s.get_channels().len(), 1);
    assert_eq!(s.get_channels()[0].name, "gen"); // unchanged
}

#[test]
fn update_channel_in_place() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "old"), ch(2, "dev")]);
    let mut updated = ch(1, "new");
    updated.description = Some("updated desc".into());
    s.update_channel(1, updated);
    assert_eq!(s.get_channel(1).unwrap().name, "new");
    assert_eq!(s.get_channel(1).unwrap().description.as_deref(), Some("updated desc"));
    assert_eq!(s.get_channels().len(), 2); // no duplication
}

#[test]
fn update_channel_updates_current() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_channel(Some(1));
    s.update_channel(1, ch(1, "renamed"));
    assert_eq!(s.get_current_channel().unwrap().name, "renamed");
}

#[test]
fn update_channel_nonexistent_noop() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.update_channel(99, ch(99, "ghost"));
    assert_eq!(s.get_channels().len(), 1);
}

#[test]
fn remove_channel() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen"), ch(2, "dev")]);
    s.set_current_channel(Some(1));
    s.increment_unread(1);
    s.increment_mention(1);
    s.add_message(1, msg(100, 1, "hi"));
    s.remove_channel(1);
    assert_eq!(s.get_channels().len(), 1);
    assert!(s.get_current_channel().is_none()); // cleared
    assert_eq!(s.get_unread_count(1), 0);
    assert_eq!(s.get_mention_count(1), 0);
    assert!(s.get_messages(1).is_none());
}

// ── Channel filter ──

#[test]
fn filter_channels_by_name() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "general"), ch(2, "dev-ops"), ch(3, "design")]);
    let r = s.filter_channels("dev", true);
    assert_eq!(r.len(), 1);
    assert_eq!(r[0].name, "dev-ops");
}

#[test]
fn filter_channels_case_insensitive() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "General"), ch(2, "DevOps")]);
    let r = s.filter_channels("general", true);
    assert_eq!(r.len(), 1);
    assert_eq!(r[0].name, "General");
}

#[test]
fn filter_channels_by_description() {
    let mut s = ChannelState::new();
    let mut c = ch(1, "random");
    c.description = Some("Team discussions".into());
    s.set_channels(vec![c, ch(2, "dev")]);
    let r = s.filter_channels("discuss", true);
    assert_eq!(r.len(), 1);
    assert_eq!(r[0].name, "random");
}

#[test]
fn filter_channels_excludes_archived() {
    let mut s = ChannelState::new();
    let mut archived = ch(2, "old-dev");
    archived.is_archived = true;
    s.set_channels(vec![ch(1, "dev"), archived]);
    let r = s.filter_channels("dev", false);
    assert_eq!(r.len(), 1);
    assert_eq!(r[0].name, "dev");
    let r_all = s.filter_channels("dev", true);
    assert_eq!(r_all.len(), 2);
}

#[test]
fn filter_channels_empty_query_returns_all() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "a"), ch(2, "b")]);
    assert_eq!(s.filter_channels("", true).len(), 2);
}

// ── Atomic select_channel ──

#[test]
fn select_channel_sets_current_and_clears_counts() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen"), ch(2, "dev")]);
    s.increment_unread(1);
    s.increment_unread(1);
    s.increment_mention(1);
    let selected = s.select_channel(Some(1));
    assert_eq!(selected.unwrap().name, "gen");
    assert_eq!(s.get_unread_count(1), 0);
    assert_eq!(s.get_mention_count(1), 0);
}

#[test]
fn select_channel_none_clears_current() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.select_channel(Some(1));
    assert!(s.get_current_channel().is_some());
    s.select_channel(None);
    assert!(s.get_current_channel().is_none());
}

#[test]
fn select_channel_nonexistent() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    let selected = s.select_channel(Some(99));
    assert!(selected.is_none());
}

// ── Batch counts ──

#[test]
fn get_all_unread_counts() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1,"a"), ch(2,"b")]);
    s.increment_unread(1);
    s.increment_unread(1);
    s.increment_unread(2);
    let counts = s.get_all_unread_counts();
    assert_eq!(counts.get(&1).copied(), Some(2));
    assert_eq!(counts.get(&2).copied(), Some(1));
    assert_eq!(counts.len(), 2);
}

#[test]
fn get_all_mention_counts() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1,"a"), ch(3,"c")]);
    s.increment_mention(1);
    s.increment_mention(3);
    let counts = s.get_all_mention_counts();
    assert_eq!(counts.get(&1).copied(), Some(1));
    assert_eq!(counts.get(&3).copied(), Some(1));
}

// ── Current user + enrich_sender ──

#[test]
fn set_current_user() {
    let mut s = ChannelState::new();
    s.set_current_user(Some(user(10, "alice")));
    assert_eq!(s.current_user_id(), Some(10));
    assert_eq!(s.current_user().unwrap().username, "alice");
}

#[test]
fn enrich_sender_fills_missing_user() {
    let mut s = ChannelState::new();
    s.set_current_user(Some(user(10, "alice")));
    let mut m = msg(1, 1, "hi");
    m.sender_user_id = Some(10);
    assert!(m.sender_user.is_none());
    s.enrich_sender(&mut m);
    assert!(m.sender_user.is_some());
    assert_eq!(m.sender_user.as_ref().unwrap().username, "alice");
}

#[test]
fn enrich_sender_skips_if_already_present() {
    let mut s = ChannelState::new();
    s.set_current_user(Some(user(10, "alice")));
    let mut m = msg(1, 1, "hi");
    m.sender_user_id = Some(10);
    m.sender_user = Some(user(10, "original"));
    s.enrich_sender(&mut m);
    assert_eq!(m.sender_user.as_ref().unwrap().username, "original"); // unchanged
}

#[test]
fn enrich_sender_skips_different_user() {
    let mut s = ChannelState::new();
    s.set_current_user(Some(user(10, "alice")));
    let mut m = msg(1, 1, "hi");
    m.sender_user_id = Some(20); // different user
    s.enrich_sender(&mut m);
    assert!(m.sender_user.is_none()); // not enriched
}

#[test]
fn enrich_sender_skips_no_current_user() {
    let s = ChannelState::new();
    let mut m = msg(1, 1, "hi");
    m.sender_user_id = Some(10);
    s.enrich_sender(&mut m);
    assert!(m.sender_user.is_none());
}

#[test]
fn on_new_message_enriches_sender() {
    let mut s = ChannelState::new();
    s.set_channels(vec![ch(1, "gen")]);
    s.set_current_user(Some(user(10, "alice")));
    let mut m = msg(1, 1, "hi");
    m.sender_user_id = Some(10);
    m.created_at = Some("2026-01-01T00:00:00Z".into());
    s.on_new_message(m);
    let cached = &s.get_messages(1).unwrap().messages[0];
    assert_eq!(cached.sender_user.as_ref().unwrap().username, "alice");
    // Preview should also reflect enriched sender
    let preview = s.get_last_message(1).unwrap();
    assert_eq!(preview.sender_name, "alice");
}
