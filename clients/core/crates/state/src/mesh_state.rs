use agentsmesh_types::proto_mesh_v1::{ChannelInfo, MeshEdge, MeshNode, MeshTopology, RunnerInfo};

#[derive(Debug, Default)]
pub struct MeshState {
    topology: Option<MeshTopology>,
    selected_node: Option<String>,
}

impl MeshState {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn topology(&self) -> Option<&MeshTopology> {
        self.topology.as_ref()
    }

    pub fn selected_node(&self) -> Option<&str> {
        self.selected_node.as_deref()
    }

    pub fn set_topology(&mut self, topology: MeshTopology) {
        tracing::debug!(target: "mesh", nodes = topology.nodes.len(), edges = topology.edges.len(), "set topology");
        self.topology = Some(topology);
    }

    pub fn clear_topology(&mut self) {
        tracing::debug!(target: "mesh", "clear topology");
        self.topology = None;
    }

    pub fn select_node(&mut self, pod_key: Option<String>) {
        self.selected_node = pod_key;
    }

    /// Patch an existing topology node's status/agent_status in place (no-op if
    /// the pod_key isn't a node). Empty `status` is ignored so an agent-only
    /// event can't blank the node status.
    pub fn update_node_status(&mut self, pod_key: &str, status: &str, agent_status: Option<&str>) -> bool {
        let Some(topology) = self.topology.as_mut() else {
            return false;
        };
        let Some(node) = topology.nodes.iter_mut().find(|n| n.pod_key == pod_key) else {
            return false;
        };
        if !status.is_empty() {
            node.status = status.to_string();
        }
        if let Some(agent) = agent_status {
            node.agent_status = agent.to_string();
        }
        true
    }

    pub fn get_node_by_key(&self, pod_key: &str) -> Option<&MeshNode> {
        self.topology
            .as_ref()
            .and_then(|t| t.nodes.iter().find(|n| n.pod_key == pod_key))
    }

    pub fn get_edges_for_node(&self, pod_key: &str) -> Vec<&MeshEdge> {
        match &self.topology {
            Some(t) => t
                .edges
                .iter()
                .filter(|e| e.source == pod_key || e.target == pod_key)
                .collect(),
            None => Vec::new(),
        }
    }

    pub fn get_channels_for_node(&self, pod_key: &str) -> Vec<&ChannelInfo> {
        match &self.topology {
            Some(t) => t
                .channels
                .iter()
                .filter(|c| c.pod_keys.iter().any(|k| k == pod_key))
                .collect(),
            None => Vec::new(),
        }
    }

    pub fn get_nodes_by_runner(&self, runner_id: i64) -> Vec<&MeshNode> {
        match &self.topology {
            Some(t) => t
                .nodes
                .iter()
                .filter(|n| n.runner_id == runner_id)
                .collect(),
            None => Vec::new(),
        }
    }

    pub fn get_active_nodes(&self) -> Vec<&MeshNode> {
        match &self.topology {
            Some(t) => t
                .nodes
                .iter()
                .filter(|n| n.status == "running" || n.status == "creating")
                .collect(),
            None => Vec::new(),
        }
    }

    pub fn get_runner_info(&self, runner_id: i64) -> Option<&RunnerInfo> {
        self.topology
            .as_ref()
            .and_then(|t| t.runners.iter().find(|r| r.id == runner_id))
    }
}
