// Unit tests for wire_convert.rs (split out to keep the source file under
// the 200-line limit; `#[path]`-included so it stays a same-crate unit test
// with private access via `use super::*`).

use super::*;
use agentsmesh_types::proto_channel_v1::ChannelMessageSenderAgent;

#[test]
fn channel_wire_to_state_maps_fields_and_wraps_options() {
    let w = WireChannel {
        id: 7,
        organization_id: 3,
        name: "gen".into(),
        visibility: "public".into(),
        member_count: 5,
        agent_count: 2,
        created_at: "t1".into(),
        updated_at: "t2".into(),
        is_archived: true,
        is_member: true,
        description: Some("d".into()),
        ticket_slug: Some("tk".into()),
        ..Default::default()
    };
    let s = wire_channel_to_state(w);
    assert_eq!(s.id, 7);
    assert_eq!(s.name, "gen");
    assert_eq!(s.organization_id, Some(3)); // wire i64 → state Option
    assert_eq!(s.visibility, Some("public".to_string()));
    assert_eq!(s.member_count, Some(5));
    assert_eq!(s.agent_count, Some(2));
    assert_eq!(s.created_at, Some("t1".to_string()));
    assert_eq!(s.description, Some("d".to_string())); // Option → Option direct
    assert!(s.is_archived);
    assert_eq!(s.unread_count, 0); // client-derived default
}

#[test]
fn apply_fetched_channels_preserves_client_derived_state() {
    let mut st = ChannelState::new();
    st.set_channels(vec![StateChannel { id: 1, name: "a".into(), ..Default::default() }]);
    st.increment_unread(1);
    st.increment_unread(1);
    // fetch returns the channel with no unread — set_channels must preserve it
    st.apply_fetched_channels(vec![WireChannel { id: 1, name: "renamed".into(), ..Default::default() }]);
    assert_eq!(st.get_channel(1).unwrap().name, "renamed"); // wire field applied
    assert_eq!(st.get_unread_count(1), 2); // client-derived preserved
}

#[test]
fn message_wire_to_state_maps_nested_sender() {
    let w = WireMessage {
        id: 10,
        channel_id: 1,
        body: "hi".into(),
        message_type: "text".into(),
        created_at: "t".into(),
        is_deleted: false,
        sender_user: Some(WireSenderUser {
            id: 5,
            username: "alice".into(),
            name: Some("Alice".into()),
            avatar_url: None,
        }),
        sender_pod_info: Some(WireSenderPod {
            pod_key: "pod-1".into(),
            alias: Some("bot".into()),
            agent: Some(ChannelMessageSenderAgent { name: "claude".into() }),
        }),
        ..Default::default()
    };
    let s = wire_message_to_state(w);
    assert_eq!(s.message_type, Some("text".to_string())); // wire String → state Option
    assert_eq!(s.is_deleted, Some(false));
    let su = s.sender_user.expect("sender_user");
    assert_eq!(su.username, "alice");
    assert_eq!(su.name, Some("Alice".to_string()));
    let spi = s.sender_pod_info.expect("sender_pod_info");
    assert_eq!(spi.pod_key, "pod-1");
    assert_eq!(spi.agent.expect("agent").name, "claude");
}

#[test]
fn apply_fetched_members_and_pods() {
    let mut st = ChannelState::new();
    st.apply_fetched_members(1, vec![WireMember {
        channel_id: 1,
        user_id: 2,
        role: "member".into(),
        is_muted: false,
        joined_at: "t".into(),
    }]);
    let members = st.get_channel_members(1);
    assert_eq!(members.len(), 1);
    assert_eq!(members[0].user_id, 2);

    st.apply_fetched_pods(1, vec![WirePod {
        id: 9,
        pod_key: "p".into(),
        alias: None,
        status: "running".into(),
        agent_status: "idle".into(),
    }]);
    let pods = st.get_channel_pods(1);
    assert_eq!(pods.len(), 1);
    assert_eq!(pods[0].pod_key, "p");
}
