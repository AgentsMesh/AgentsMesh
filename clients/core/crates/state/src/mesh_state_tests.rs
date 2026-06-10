use crate::mesh_state::MeshState;
use agentsmesh_types::proto_mesh_v1::{ChannelInfo, MeshEdge, MeshNode, MeshTopology, RunnerInfo};

fn sample_topology() -> MeshTopology {
    MeshTopology {
        nodes: vec![
            MeshNode { pod_key: "p1".into(), alias: Some("worker".into()), status: "running".into(), agent_status: "idle".into(), agent_slug: "claude".into(), runner_id: 1, ..Default::default() },
            MeshNode { pod_key: "p2".into(), status: "creating".into(), agent_slug: "aider".into(), runner_id: 1, ..Default::default() },
            MeshNode { pod_key: "p3".into(), status: "terminated".into(), agent_slug: "claude".into(), runner_id: 2, ..Default::default() },
        ],
        edges: vec![MeshEdge { source: "p1".into(), target: "p2".into(), status: "bound".into(), ..Default::default() }],
        channels: vec![ChannelInfo { id: 1, name: "general".into(), pod_keys: vec!["p1".into(), "p2".into()], ..Default::default() }],
        runners: vec![
            RunnerInfo { id: 1, node_id: "r1".into(), status: "online".into(), ..Default::default() },
            RunnerInfo { id: 2, node_id: "r2".into(), status: "offline".into(), ..Default::default() },
        ],
    }
}

#[test] fn new_is_empty() { let s = MeshState::new(); assert!(s.topology().is_none()); assert!(s.selected_node().is_none()); }
#[test] fn set_topology() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert_eq!(s.topology().unwrap().nodes.len(), 3); }
#[test] fn clear_topology() { let mut s = MeshState::new(); s.set_topology(sample_topology()); s.clear_topology(); assert!(s.topology().is_none()); }
#[test] fn select_node() { let mut s = MeshState::new(); s.select_node(Some("p1".into())); assert_eq!(s.selected_node(), Some("p1")); s.select_node(None); assert!(s.selected_node().is_none()); }
#[test] fn get_node_by_key() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert_eq!(s.get_node_by_key("p1").unwrap().agent_slug, "claude"); assert!(s.get_node_by_key("x").is_none()); }
#[test] fn get_node_no_topo() { let s = MeshState::new(); assert!(s.get_node_by_key("p1").is_none()); }
#[test] fn get_edges() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert_eq!(s.get_edges_for_node("p1").len(), 1); assert!(s.get_edges_for_node("p3").is_empty()); }
#[test] fn get_edges_no_topo() { let s = MeshState::new(); assert!(s.get_edges_for_node("p1").is_empty()); }
#[test] fn get_channels() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert_eq!(s.get_channels_for_node("p1").len(), 1); assert!(s.get_channels_for_node("p3").is_empty()); }
#[test] fn get_channels_no_topo() { let s = MeshState::new(); assert!(s.get_channels_for_node("p1").is_empty()); }
#[test] fn get_nodes_by_runner() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert_eq!(s.get_nodes_by_runner(1).len(), 2); assert!(s.get_nodes_by_runner(99).is_empty()); }
#[test] fn get_active_nodes() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert_eq!(s.get_active_nodes().len(), 2); }
#[test] fn get_runner_info() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert_eq!(s.get_runner_info(1).unwrap().node_id, "r1"); assert!(s.get_runner_info(99).is_none()); }
#[test] fn update_node_status_patches_existing() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert!(s.update_node_status("p2", "running", Some("executing"))); let n = s.get_node_by_key("p2").unwrap(); assert_eq!(n.status, "running"); assert_eq!(n.agent_status, "executing"); }
#[test] fn update_node_status_missing_node() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert!(!s.update_node_status("nope", "running", None)); }
#[test] fn update_node_status_no_topo() { let mut s = MeshState::new(); assert!(!s.update_node_status("p1", "running", None)); }
#[test] fn update_node_status_empty_status_keeps_old() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert!(s.update_node_status("p1", "", Some("executing"))); let n = s.get_node_by_key("p1").unwrap(); assert_eq!(n.status, "running"); assert_eq!(n.agent_status, "executing"); }
#[test] fn update_node_status_agent_none_keeps_old() { let mut s = MeshState::new(); s.set_topology(sample_topology()); assert!(s.update_node_status("p1", "paused", None)); let n = s.get_node_by_key("p1").unwrap(); assert_eq!(n.status, "paused"); assert_eq!(n.agent_status, "idle"); }
