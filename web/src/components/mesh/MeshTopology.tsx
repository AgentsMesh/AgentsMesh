"use client";

import { useCallback, useEffect } from "react";
import {
  ReactFlow,
  Controls,
  Background,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type NodeTypes,
  type EdgeTypes,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import PodNode from "./PodNode";
import BindingEdge from "./BindingEdge";
import { useMeshStore, type MeshNode, type MeshEdge } from "@/stores/mesh";

// Custom node types - using proper types for ReactFlow
const nodeTypes: NodeTypes = {
  pod: PodNode,
};

// Custom edge types - using proper types for ReactFlow
const edgeTypes: EdgeTypes = {
  binding: BindingEdge,
};

// Layout algorithm - simple force-directed-like placement
function calculateLayout(
  pods: MeshNode[],
  edges: MeshEdge[]
): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = [];
  const flowEdges: Edge[] = [];

  // Create pod nodes
  const podCount = pods.length;
  const radius = Math.max(200, podCount * 40);

  pods.forEach((pod, index) => {
    // Circular layout for pods
    const angle = (2 * Math.PI * index) / podCount;
    const x = pod.position?.x ?? 400 + radius * Math.cos(angle);
    const y = pod.position?.y ?? 300 + radius * Math.sin(angle);

    nodes.push({
      id: pod.pod_key,
      type: "pod",
      position: { x, y },
      data: { node: pod },
    });
  });

  // Create binding edges between pods
  edges.forEach((edge) => {
    flowEdges.push({
      id: `binding-${edge.id}-${edge.source}-${edge.target}`,
      source: edge.source,
      target: edge.target,
      type: "binding",
      data: {
        status: edge.status,
        grantedScopes: edge.granted_scopes,
        pendingScopes: edge.pending_scopes,
      },
    });
  });

  return { nodes, edges: flowEdges };
}

export default function MeshTopology() {
  const { topology, selectedNode, selectNode, fetchTopology } =
    useMeshStore();

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // Fetch topology on mount - realtime events handle subsequent updates
  useEffect(() => {
    fetchTopology();
  }, [fetchTopology]);

  // Update nodes and edges when topology changes
  useEffect(() => {
    if (topology) {
      const layout = calculateLayout(topology.nodes, topology.edges);
      setNodes(layout.nodes);
      setEdges(layout.edges);
    }
  }, [topology, setNodes, setEdges]);

  // Update selection state
  useEffect(() => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.type === "pod") {
          return {
            ...node,
            data: {
              ...node.data,
              isSelected: node.id === selectedNode,
            },
          };
        }
        return node;
      })
    );
  }, [selectedNode, setNodes]);

  // Handle node click
  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      if (node.type === "pod") {
        selectNode(node.id);
      }
    },
    [selectNode]
  );

  // Handle pane click (deselect)
  const onPaneClick = useCallback(() => {
    selectNode(null);
  }, [selectNode]);

  // Node color for minimap
  const nodeColor = useCallback((node: Node) => {
    const data = node.data as { node: MeshNode };
    switch (data.node?.status) {
      case "running":
        return "#22c55e";
      case "initializing":
        return "#eab308";
      case "failed":
        return "#ef4444";
      default:
        return "#6b7280";
    }
  }, []);

  if (!topology) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (topology.nodes.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <svg
            className="w-16 h-16 mx-auto text-muted-foreground mb-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
            />
          </svg>
          <h3 className="text-lg font-medium text-foreground mb-2">No Active Pods</h3>
          <p className="text-muted-foreground">
            Start an AgentPod to see it in the mesh
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full h-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        fitView
        minZoom={0.1}
        maxZoom={2}
        defaultViewport={{ x: 0, y: 0, zoom: 1 }}
      >
        <Controls />
        <MiniMap nodeColor={nodeColor} zoomable pannable />
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
      </ReactFlow>
    </div>
  );
}
