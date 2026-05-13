// Hand-maintained `prost::Message` mirrors of
// `proto/mesh/v1/mesh.proto`. Tag numbers match the .proto byte-for-byte;
// `tools/validate_prost_tags` runs at build time to catch drift
// (watch list §8). NO `Serialize`/`Deserialize` derives on these
// structs — binary wire only (conventions §2.5, §3).

#[derive(Clone, PartialEq, prost::Message)]
pub struct MeshNode {
    #[prost(string, tag = "1")]
    pub pod_key: String,
    #[prost(string, tag = "2")]
    pub status: String,
    #[prost(string, tag = "3")]
    pub agent_status: String,
    #[prost(string, optional, tag = "4")]
    pub model: Option<String>,
    #[prost(string, optional, tag = "5")]
    pub title: Option<String>,
    #[prost(string, optional, tag = "6")]
    pub alias: Option<String>,
    #[prost(string, tag = "7")]
    pub agent_slug: String,
    #[prost(int64, optional, tag = "8")]
    pub ticket_id: Option<i64>,
    #[prost(string, optional, tag = "9")]
    pub ticket_slug: Option<String>,
    #[prost(string, optional, tag = "10")]
    pub ticket_title: Option<String>,
    #[prost(int64, optional, tag = "11")]
    pub repository_id: Option<i64>,
    #[prost(int64, tag = "12")]
    pub created_by_id: i64,
    #[prost(int64, tag = "13")]
    pub runner_id: i64,
    #[prost(string, tag = "14")]
    pub runner_node_id: String,
    #[prost(string, tag = "15")]
    pub runner_status: String,
    #[prost(string, optional, tag = "16")]
    pub started_at: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MeshEdge {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub source: String,
    #[prost(string, tag = "3")]
    pub target: String,
    #[prost(string, repeated, tag = "4")]
    pub granted_scopes: Vec<String>,
    #[prost(string, repeated, tag = "5")]
    pub pending_scopes: Vec<String>,
    #[prost(string, tag = "6")]
    pub status: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ChannelInfo {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub name: String,
    #[prost(string, optional, tag = "3")]
    pub description: Option<String>,
    #[prost(string, repeated, tag = "4")]
    pub pod_keys: Vec<String>,
    #[prost(int32, tag = "5")]
    pub message_count: i32,
    #[prost(bool, tag = "6")]
    pub is_archived: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RunnerInfo {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub node_id: String,
    #[prost(string, tag = "3")]
    pub status: String,
    #[prost(int32, tag = "4")]
    pub max_concurrent_pods: i32,
    #[prost(int32, tag = "5")]
    pub current_pods: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MeshTopology {
    #[prost(message, repeated, tag = "1")]
    pub nodes: Vec<MeshNode>,
    #[prost(message, repeated, tag = "2")]
    pub edges: Vec<MeshEdge>,
    #[prost(message, repeated, tag = "3")]
    pub channels: Vec<ChannelInfo>,
    #[prost(message, repeated, tag = "4")]
    pub runners: Vec<RunnerInfo>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetMeshTopologyRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetTicketPodsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(bool, optional, tag = "3")]
    pub active_only: Option<bool>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetTicketPodsResponse {
    #[prost(message, repeated, tag = "1")]
    pub pods: Vec<MeshNode>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct BatchGetTicketPodsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, repeated, tag = "2")]
    pub ticket_ids: Vec<i64>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct TicketPodList {
    #[prost(message, repeated, tag = "1")]
    pub pods: Vec<MeshNode>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct BatchGetTicketPodsResponse {
    #[prost(map = "int64, message", tag = "1")]
    pub ticket_pods: ::std::collections::HashMap<i64, TicketPodList>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreatePodForTicketRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub ticket_slug: String,
    #[prost(int64, tag = "3")]
    pub runner_id: i64,
    #[prost(string, optional, tag = "4")]
    pub prompt: Option<String>,
    #[prost(string, optional, tag = "5")]
    pub model: Option<String>,
    #[prost(string, optional, tag = "6")]
    pub permission_mode: Option<String>,
}

// MeshMessageService — pod-to-pod direct messaging. Mirrors
// service MeshMessageService in proto/mesh/v1/mesh.proto.

#[derive(Clone, PartialEq, prost::Message)]
pub struct MeshMessage {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, optional, tag = "2")]
    pub sender_pod: Option<String>,
    #[prost(string, optional, tag = "3")]
    pub receiver_pod: Option<String>,
    #[prost(string, optional, tag = "4")]
    pub message_type: Option<String>,
    #[prost(string, optional, tag = "5")]
    pub content: Option<String>,
    #[prost(string, optional, tag = "6")]
    pub correlation_id: Option<String>,
    #[prost(int64, optional, tag = "7")]
    pub reply_to_id: Option<i64>,
    #[prost(bool, optional, tag = "8")]
    pub is_read: Option<bool>,
    #[prost(string, optional, tag = "9")]
    pub created_at: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MeshDeadLetterEntry {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(message, optional, tag = "2")]
    pub message: Option<MeshMessage>,
    #[prost(string, optional, tag = "3")]
    pub error: Option<String>,
    #[prost(string, optional, tag = "4")]
    pub created_at: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListMeshMessagesRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub pod_key: String,
    #[prost(bool, optional, tag = "3")]
    pub unread_only: Option<bool>,
    #[prost(int32, optional, tag = "4")]
    pub offset: Option<i32>,
    #[prost(int32, optional, tag = "5")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListMeshMessagesResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<MeshMessage>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
    #[prost(int64, tag = "5")]
    pub unread_count: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetMeshUnreadCountRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub pod_key: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetMeshUnreadCountResponse {
    #[prost(int64, tag = "1")]
    pub count: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetMeshMessageRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MarkAllReadRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub pod_key: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct MarkAllReadResponse {
    #[prost(int64, tag = "1")]
    pub marked_count: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetConversationRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub correlation_id: String,
    #[prost(int32, optional, tag = "3")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetConversationResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<MeshMessage>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetSentMessagesRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub pod_key: String,
    #[prost(int32, optional, tag = "3")]
    pub offset: Option<i32>,
    #[prost(int32, optional, tag = "4")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetSentMessagesResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<MeshMessage>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetDeadLettersRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int32, optional, tag = "2")]
    pub offset: Option<i32>,
    #[prost(int32, optional, tag = "3")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetDeadLettersResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<MeshDeadLetterEntry>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ReplayDeadLetterRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub entry_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ReplayDeadLetterResponse {
    #[prost(string, optional, tag = "1")]
    pub message: Option<String>,
    #[prost(message, optional, tag = "2")]
    pub replayed_message: Option<MeshMessage>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn mesh_node_round_trip_preserves_every_field() {
        let node = MeshNode {
            pod_key: "pod-001".into(),
            status: "running".into(),
            agent_status: "active".into(),
            model: Some("claude-3-7-sonnet".into()),
            title: Some("Build a chatbot".into()),
            alias: Some("worker".into()),
            agent_slug: "claude-code".into(),
            ticket_id: Some(42),
            ticket_slug: Some("AGM-123".into()),
            ticket_title: Some("Fix the bug".into()),
            repository_id: Some(7),
            created_by_id: 99,
            runner_id: 11,
            runner_node_id: "node-abc".into(),
            runner_status: "online".into(),
            started_at: Some("2026-05-10T00:00:00Z".into()),
        };
        let bytes = node.encode_to_vec();
        let decoded = MeshNode::decode(&bytes[..]).unwrap();
        assert_eq!(decoded, node);
    }

    #[test]
    fn mesh_node_optional_fields_absent_distinguishable() {
        let node = MeshNode {
            pod_key: "p".into(),
            status: "running".into(),
            agent_status: "idle".into(),
            agent_slug: "codex".into(),
            created_by_id: 1,
            runner_id: 0,
            runner_node_id: "".into(),
            runner_status: "".into(),
            ..Default::default()
        };
        let bytes = node.encode_to_vec();
        let decoded = MeshNode::decode(&bytes[..]).unwrap();
        assert!(decoded.model.is_none());
        assert!(decoded.title.is_none());
        assert!(decoded.alias.is_none());
        assert!(decoded.ticket_id.is_none());
        assert!(decoded.started_at.is_none());
    }

    #[test]
    fn mesh_edge_round_trip_preserves_scopes() {
        let edge = MeshEdge {
            id: 17,
            source: "pod-a".into(),
            target: "pod-b".into(),
            granted_scopes: vec!["pod:read".into(), "pod:write".into()],
            pending_scopes: vec!["pod:admin".into()],
            status: "active".into(),
        };
        let bytes = edge.encode_to_vec();
        let decoded = MeshEdge::decode(&bytes[..]).unwrap();
        assert_eq!(decoded, edge);
    }

    #[test]
    fn channel_info_round_trip_with_pod_keys() {
        let ch = ChannelInfo {
            id: 5,
            name: "general".into(),
            description: Some("All hands".into()),
            pod_keys: vec!["p1".into(), "p2".into()],
            message_count: 42,
            is_archived: false,
        };
        let bytes = ch.encode_to_vec();
        let decoded = ChannelInfo::decode(&bytes[..]).unwrap();
        assert_eq!(decoded, ch);
    }

    #[test]
    fn runner_info_round_trip() {
        let r = RunnerInfo {
            id: 3,
            node_id: "runner-host-01".into(),
            status: "online".into(),
            max_concurrent_pods: 10,
            current_pods: 4,
        };
        let bytes = r.encode_to_vec();
        let decoded = RunnerInfo::decode(&bytes[..]).unwrap();
        assert_eq!(decoded, r);
    }

    #[test]
    fn mesh_topology_round_trip_preserves_collections() {
        let topo = MeshTopology {
            nodes: vec![MeshNode {
                pod_key: "p1".into(),
                status: "running".into(),
                agent_status: "active".into(),
                agent_slug: "claude".into(),
                created_by_id: 1,
                runner_id: 1,
                runner_node_id: "n".into(),
                runner_status: "online".into(),
                ..Default::default()
            }],
            edges: vec![MeshEdge {
                id: 1,
                source: "p1".into(),
                target: "p2".into(),
                granted_scopes: vec!["pod:read".into()],
                pending_scopes: vec![],
                status: "active".into(),
            }],
            channels: vec![ChannelInfo {
                id: 1,
                name: "ops".into(),
                pod_keys: vec!["p1".into()],
                message_count: 0,
                is_archived: false,
                ..Default::default()
            }],
            runners: vec![RunnerInfo {
                id: 1,
                node_id: "n".into(),
                status: "online".into(),
                max_concurrent_pods: 5,
                current_pods: 1,
            }],
        };
        let bytes = topo.encode_to_vec();
        let decoded = MeshTopology::decode(&bytes[..]).unwrap();
        assert_eq!(decoded.nodes.len(), 1);
        assert_eq!(decoded.edges.len(), 1);
        assert_eq!(decoded.channels.len(), 1);
        assert_eq!(decoded.runners.len(), 1);
        assert_eq!(decoded.edges[0].granted_scopes, vec!["pod:read"]);
    }

    #[test]
    fn get_ticket_pods_request_optional_active_only_distinguishable() {
        // active_only = None → field absent on wire (proto3 optional trap, watch list §3)
        let req_absent = GetTicketPodsRequest {
            org_slug: "acme".into(),
            ticket_slug: "T-1".into(),
            active_only: None,
        };
        let req_explicit_false = GetTicketPodsRequest {
            org_slug: "acme".into(),
            ticket_slug: "T-1".into(),
            active_only: Some(false),
        };
        let bytes_absent = req_absent.encode_to_vec();
        let bytes_false = req_explicit_false.encode_to_vec();
        // Optional wraps the wire — Some(false) emits the tag, None elides.
        assert_ne!(bytes_absent.len(), bytes_false.len());
        let decoded_absent = GetTicketPodsRequest::decode(&bytes_absent[..]).unwrap();
        let decoded_false = GetTicketPodsRequest::decode(&bytes_false[..]).unwrap();
        assert!(decoded_absent.active_only.is_none());
        assert_eq!(decoded_false.active_only, Some(false));
    }

    #[test]
    fn batch_get_ticket_pods_response_round_trip_with_map() {
        use std::collections::HashMap;
        let mut map = HashMap::new();
        map.insert(100, TicketPodList {
            pods: vec![MeshNode {
                pod_key: "p-100".into(),
                status: "running".into(),
                agent_status: "active".into(),
                agent_slug: "claude".into(),
                created_by_id: 1,
                runner_id: 1,
                runner_node_id: "n".into(),
                runner_status: "online".into(),
                ..Default::default()
            }],
        });
        map.insert(200, TicketPodList { pods: vec![] });
        let resp = BatchGetTicketPodsResponse { ticket_pods: map };
        let bytes = resp.encode_to_vec();
        let decoded = BatchGetTicketPodsResponse::decode(&bytes[..]).unwrap();
        assert_eq!(decoded.ticket_pods.len(), 2);
        assert_eq!(decoded.ticket_pods.get(&100).unwrap().pods.len(), 1);
        assert!(decoded.ticket_pods.get(&200).unwrap().pods.is_empty());
    }

    #[test]
    fn create_pod_for_ticket_request_all_fields() {
        let req = CreatePodForTicketRequest {
            org_slug: "acme".into(),
            ticket_slug: "T-1".into(),
            runner_id: 7,
            prompt: Some("hello".into()),
            model: Some("opus".into()),
            permission_mode: Some("auto".into()),
        };
        let bytes = req.encode_to_vec();
        let decoded = CreatePodForTicketRequest::decode(&bytes[..]).unwrap();
        assert_eq!(decoded, req);
    }

    // --- MeshMessageService round-trips ---

    #[test]
    fn mesh_message_round_trip_all_fields() {
        let m = MeshMessage {
            id: 42,
            sender_pod: Some("pod-a".into()),
            receiver_pod: Some("pod-b".into()),
            message_type: Some("requirement".into()),
            content: Some("{\"text\":\"hello\"}".into()),
            correlation_id: Some("corr-1".into()),
            reply_to_id: Some(7),
            is_read: Some(false),
            created_at: Some("2026-05-10T00:00:00Z".into()),
        };
        let bytes = m.encode_to_vec();
        let decoded = MeshMessage::decode(&bytes[..]).unwrap();
        assert_eq!(decoded, m);
    }

    #[test]
    fn mesh_message_optionals_absent() {
        let m = MeshMessage { id: 1, ..Default::default() };
        let bytes = m.encode_to_vec();
        let decoded = MeshMessage::decode(&bytes[..]).unwrap();
        assert!(decoded.sender_pod.is_none());
        assert!(decoded.content.is_none());
        assert!(decoded.is_read.is_none());
    }

    #[test]
    fn mesh_dead_letter_entry_round_trip() {
        let e = MeshDeadLetterEntry {
            id: 11,
            message: Some(MeshMessage { id: 100, ..Default::default() }),
            error: Some("timeout".into()),
            created_at: Some("2026-05-10T00:00:00Z".into()),
        };
        let bytes = e.encode_to_vec();
        let decoded = MeshDeadLetterEntry::decode(&bytes[..]).unwrap();
        assert_eq!(decoded, e);
    }

    #[test]
    fn list_mesh_messages_request_round_trip() {
        let req = ListMeshMessagesRequest {
            org_slug: "acme".into(),
            pod_key: "pod-1".into(),
            unread_only: Some(true),
            offset: Some(0),
            limit: Some(20),
        };
        let bytes = req.encode_to_vec();
        let decoded = ListMeshMessagesRequest::decode(&bytes[..]).unwrap();
        assert_eq!(decoded, req);
    }

    #[test]
    fn list_mesh_messages_response_preserves_unread_count() {
        let resp = ListMeshMessagesResponse {
            items: vec![MeshMessage { id: 1, ..Default::default() }],
            total: 1,
            limit: 20,
            offset: 0,
            unread_count: 7,
        };
        let bytes = resp.encode_to_vec();
        let decoded = ListMeshMessagesResponse::decode(&bytes[..]).unwrap();
        assert_eq!(decoded.unread_count, 7);
        assert_eq!(decoded.items.len(), 1);
    }

    #[test]
    fn get_mesh_unread_count_response_round_trip() {
        let resp = GetMeshUnreadCountResponse { count: 5 };
        let bytes = resp.encode_to_vec();
        let decoded = GetMeshUnreadCountResponse::decode(&bytes[..]).unwrap();
        assert_eq!(decoded.count, 5);
    }

    #[test]
    fn mark_all_read_response_round_trip() {
        let resp = MarkAllReadResponse { marked_count: 12 };
        let bytes = resp.encode_to_vec();
        let decoded = MarkAllReadResponse::decode(&bytes[..]).unwrap();
        assert_eq!(decoded.marked_count, 12);
    }

    #[test]
    fn get_conversation_round_trip() {
        let req = GetConversationRequest {
            org_slug: "acme".into(),
            correlation_id: "c1".into(),
            limit: Some(50),
        };
        let resp = GetConversationResponse {
            items: vec![MeshMessage { id: 1, ..Default::default() }],
            total: 1,
            limit: 50,
        };
        let rb = req.encode_to_vec();
        let _ = GetConversationRequest::decode(&rb[..]).unwrap();
        let respb = resp.encode_to_vec();
        let decoded = GetConversationResponse::decode(&respb[..]).unwrap();
        assert_eq!(decoded.total, 1);
    }

    #[test]
    fn get_dead_letters_response_round_trip() {
        let resp = GetDeadLettersResponse {
            items: vec![MeshDeadLetterEntry {
                id: 1,
                message: Some(MeshMessage { id: 99, ..Default::default() }),
                error: Some("retry exhausted".into()),
                ..Default::default()
            }],
            total: 1,
            limit: 50,
            offset: 0,
        };
        let bytes = resp.encode_to_vec();
        let decoded = GetDeadLettersResponse::decode(&bytes[..]).unwrap();
        assert_eq!(decoded.items.len(), 1);
        assert_eq!(decoded.items[0].error, Some("retry exhausted".into()));
    }

    #[test]
    fn replay_dead_letter_response_round_trip() {
        let resp = ReplayDeadLetterResponse {
            message: Some("Replayed successfully".into()),
            replayed_message: Some(MeshMessage { id: 7, ..Default::default() }),
        };
        let bytes = resp.encode_to_vec();
        let decoded = ReplayDeadLetterResponse::decode(&bytes[..]).unwrap();
        assert_eq!(decoded.message.as_deref(), Some("Replayed successfully"));
        assert_eq!(decoded.replayed_message.unwrap().id, 7);
    }
}
