// wire(proto.channel.v1) → state(proto.channel_state.v1) projection for the
// fetch→state path. Fetch responses carry wire types; this folds them into the
// state schema so set_channels/set_messages (and their business logic:
// unread/preview/persist) own the result — UI never sees wire, and the TS
// fromProtoX + xToProto round-trip is eliminated.

use agentsmesh_types::proto_channel_v1::{
    Channel as WireChannel, ChannelMember as WireMember, ChannelMessage as WireMessage,
    ChannelMessageSenderPod as WireSenderPod, ChannelMessageSenderUser as WireSenderUser,
    ChannelPod as WirePod,
};
use agentsmesh_types::proto_pod_v1::Pod;

use super::ChannelState;
use crate::channel_types::{
    Channel as StateChannel, ChannelMember as StateMember, ChannelMessage as StateMessage,
    SenderAgentInfo, SenderPodInfo, SenderUser,
};

// wire ≈ state (same field names); only 6 wire non-optional fields need Some()
// for the state schema. Client-derived fields default — set_channels restores
// prior unread/mention/last_message/last_activity.
pub fn wire_channel_to_state(w: WireChannel) -> StateChannel {
    StateChannel {
        id: w.id,
        name: w.name,
        is_archived: w.is_archived,
        is_member: w.is_member,
        description: w.description,
        document: w.document,
        repository_id: w.repository_id,
        ticket_id: w.ticket_id,
        ticket_slug: w.ticket_slug,
        created_by_pod: w.created_by_pod,
        created_by_user_id: w.created_by_user_id,
        organization_id: Some(w.organization_id),
        visibility: Some(w.visibility),
        member_count: Some(w.member_count),
        created_at: Some(w.created_at),
        updated_at: Some(w.updated_at),
        agent_count: Some(w.agent_count),
        ..Default::default()
    }
}

fn wire_sender_user_to_state(w: WireSenderUser) -> SenderUser {
    SenderUser {
        id: w.id,
        username: w.username,
        name: w.name,
        avatar_url: w.avatar_url,
        ..Default::default() // email / is_email_verified not carried on the wire
    }
}

fn wire_sender_pod_to_state(w: WireSenderPod) -> SenderPodInfo {
    SenderPodInfo {
        pod_key: w.pod_key,
        alias: w.alias,
        agent: w.agent.map(|a| SenderAgentInfo { name: a.name, ..Default::default() }),
    }
}

pub fn wire_message_to_state(w: WireMessage) -> StateMessage {
    StateMessage {
        id: w.id,
        channel_id: w.channel_id,
        sender_pod: w.sender_pod,
        sender_user_id: w.sender_user_id,
        body: w.body,
        content_json: w.content_json,
        mentions_json: w.mentions_json,
        reply_to: w.reply_to,
        edited_at: w.edited_at,
        message_type: Some(w.message_type),
        created_at: Some(w.created_at),
        is_deleted: Some(w.is_deleted),
        sender_user: w.sender_user.map(wire_sender_user_to_state),
        sender_pod_info: w.sender_pod_info.map(wire_sender_pod_to_state),
        ..Default::default() // sender_agent_info / updated_at / legacy / client-derived
    }
}

fn wire_member_to_state(w: WireMember) -> StateMember {
    StateMember {
        channel_id: w.channel_id,
        user_id: w.user_id,
        role: w.role,
        is_muted: w.is_muted,
        joined_at: w.joined_at,
        ..Default::default() // user (SenderUser) not carried on the members wire
    }
}

fn wire_pod_to_state(w: WirePod) -> Pod {
    Pod {
        id: w.id,
        pod_key: w.pod_key,
        alias: w.alias,
        status: w.status,
        agent_status: w.agent_status,
        ..Default::default() // channel cache reads only the 5 summary fields
    }
}

impl ChannelState {
    // Fetch→state for the channel list. Replaces TS channelFromProto +
    // channelToProtoChannel + replace_cached_channels.
    pub fn apply_fetched_channels(&mut self, wire: Vec<WireChannel>) {
        let channels = wire.into_iter().map(wire_channel_to_state).collect();
        self.set_channels(channels);
    }

    // Single-object fetch (B): convert the wire GetChannel response + upsert,
    // mirroring the store's getChannel→insert dispatch (replaces TS
    // channelFromProto + channelToProtoChannel).
    pub fn apply_fetched_channel(&mut self, wire: WireChannel) {
        let mut channel = wire_channel_to_state(wire);
        let id = channel.id;
        // Preserve the client-derived fields the single-channel wire fetch
        // doesn't carry (mirror set_channels), so a GetChannel refresh can't
        // blank the unread badge / preview / sort key on an existing channel.
        let prev = self.get_channel(id).map(|p| {
            (p.unread_count, p.mention_count, p.last_message.clone(), p.last_activity_at.clone())
        });
        if let Some((u, m, lm, ts)) = prev {
            if channel.unread_count == 0 { channel.unread_count = u; }
            if channel.mention_count == 0 { channel.mention_count = m; }
            if channel.last_message.is_none() { channel.last_message = lm; }
            if channel.last_activity_at.is_none() { channel.last_activity_at = ts; }
            self.update_channel(id, channel);
        } else {
            self.add_channel(channel);
        }
    }

    // Fetch→state for a channel's messages. Replaces TS messageFromProto +
    // channelMessageToProto + replace_cached_channel_messages. set_messages
    // owns dedup/preview/persist + LRU eviction.
    pub fn apply_fetched_messages(&mut self, channel_id: i64, wire: Vec<WireMessage>, has_more: bool) {
        let messages = wire.into_iter().map(wire_message_to_state).collect();
        self.set_messages(channel_id, messages, has_more);
    }

    // Fetch→state for older messages (pagination load-more). prepend_messages
    // keeps chronological order + dedup. Replaces prepend_cached_channel_messages.
    pub fn apply_fetched_messages_prepend(&mut self, channel_id: i64, wire: Vec<WireMessage>, has_more: bool) {
        let messages = wire.into_iter().map(wire_message_to_state).collect();
        self.prepend_messages(channel_id, messages, has_more);
    }

    // Fetch→state for a channel's members. Replaces TS memberFromProto +
    // channelMemberDataToProto + replace_channel_members.
    pub fn apply_fetched_members(&mut self, channel_id: i64, wire: Vec<WireMember>) {
        let members = wire.into_iter().map(wire_member_to_state).collect();
        self.set_channel_members(channel_id, members);
    }

    // Fetch→state for a channel's pods. Replaces TS podFromProto +
    // channelPodSummaryToProtoPod + replace_channel_pods.
    pub fn apply_fetched_pods(&mut self, channel_id: i64, wire: Vec<WirePod>) {
        let pods = wire.into_iter().map(wire_pod_to_state).collect();
        self.set_channel_pods(channel_id, pods);
    }
}

#[cfg(test)]
mod tests {
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
}
