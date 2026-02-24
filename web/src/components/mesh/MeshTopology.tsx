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
  type OnNodeDrag,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { CenteredSpinner } from "@/components/ui/spinner";
import PodNode from "./PodNode";
import BindingEdge from "./BindingEdge";
import RunnerGroupNode from "./RunnerGroupNode";
import { useMeshStore, type MeshNode, type MeshEdge } from "@/stores/mesh";
import type { RunnerInfoData } from "@/lib/api";

// Custom node types
const nodeTypes: NodeTypes = {
  pod: PodNode,
  runnerGroup: RunnerGroupNode,
};

// Custom edge types
const edgeTypes: EdgeTypes = {
  binding: BindingEdge,
};

// Layout constants
const POD_WIDTH = 200;
const POD_HEIGHT = 140;
const POD_GAP_X = 20;
const POD_GAP_Y = 20;
const PODS_PER_ROW = 2;
const GROUP_PADDING_X = 20;
const GROUP_PADDING_TOP = 50; // space for header
const GROUP_PADDING_BOTTOM = 20;
const GROUP_GAP = 40;

// Grouped layout algorithm - arranges pods into Runner "workstation" groups
function calculateGroupedLayout(
  pods: MeshNode[],
  edges: MeshEdge[],
  runners?: RunnerInfoData[],
  savedPositions?: Record<string, { x: number; y: number }>
): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = [];
  const flowEdges: Edge[] = [];

  // Group pods by runner_id
  const podsByRunner = new Map<number, MeshNode[]>();
  for (const pod of pods) {
    const runnerId = pod.runner_id;
    if (!podsByRunner.has(runnerId)) {
      podsByRunner.set(runnerId, []);
    }
    podsByRunner.get(runnerId)!.push(pod);
  }

  // Build runner info lookup
  const runnerInfoMap = new Map<number, RunnerInfoData>();
  if (runners) {
    for (const r of runners) {
      runnerInfoMap.set(r.id, r);
    }
  }

  // Calculate auto-layout start X: place new groups after saved ones to avoid overlap
  let autoGroupX = 0;
  if (savedPositions) {
    for (const [runnerId, runnerPods] of podsByRunner) {
      const gid = `runner-group-${runnerId}`;
      const saved = savedPositions[gid];
      if (saved) {
        const cols = Math.min(runnerPods.length, PODS_PER_ROW);
        const gw = GROUP_PADDING_X * 2 + cols * POD_WIDTH + (cols - 1) * POD_GAP_X;
        autoGroupX = Math.max(autoGroupX, saved.x + gw + GROUP_GAP);
      }
    }
  }

  // Create group nodes and place pods inside
  for (const [runnerId, runnerPods] of podsByRunner) {
    const runnerInfo = runnerInfoMap.get(runnerId);
    const runnerNodeId = runnerPods[0]?.runner_node_id || `runner-${runnerId}`;
    const runnerStatus = runnerPods[0]?.runner_status || runnerInfo?.status || "offline";
    const groupId = `runner-group-${runnerId}`;

    // Calculate group dimensions based on pod count
    const rows = Math.ceil(runnerPods.length / PODS_PER_ROW);
    const cols = Math.min(runnerPods.length, PODS_PER_ROW);
    const groupWidth = GROUP_PADDING_X * 2 + cols * POD_WIDTH + (cols - 1) * POD_GAP_X;
    const groupHeight = GROUP_PADDING_TOP + GROUP_PADDING_BOTTOM + rows * POD_HEIGHT + (rows - 1) * POD_GAP_Y;

    // Use saved position if available, otherwise auto-layout
    const savedPos = savedPositions?.[groupId];
    const position = savedPos ?? { x: autoGroupX, y: 0 };

    // Create the runner group node
    nodes.push({
      id: groupId,
      type: "runnerGroup",
      position,
      data: {
        runnerNodeId,
        runnerStatus,
        podCount: runnerPods.length,
      },
      style: { width: groupWidth, height: groupHeight },
    });

    // Place pods inside the group using grid layout (positions relative to parent)
    runnerPods.forEach((pod, index) => {
      const col = index % PODS_PER_ROW;
      const row = Math.floor(index / PODS_PER_ROW);
      const x = GROUP_PADDING_X + col * (POD_WIDTH + POD_GAP_X);
      const y = GROUP_PADDING_TOP + row * (POD_HEIGHT + POD_GAP_Y);

      nodes.push({
        id: pod.pod_key,
        type: "pod",
        position: { x, y },
        parentId: groupId,
        extent: "parent" as const,
        draggable: false, // Pods use grid layout within their Runner Group
        data: { node: pod },
      });
    });

    // Advance auto-layout X only for groups without saved positions
    if (!savedPos) {
      autoGroupX += groupWidth + GROUP_GAP;
    }
  }

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
  const { topology, selectedNode, selectNode, fetchTopology, updateNodePosition } =
    useMeshStore();

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // Fetch topology on mount - realtime events handle subsequent updates
  useEffect(() => {
    fetchTopology();
  }, [fetchTopology]);

  // Update nodes and edges when topology changes
  // Uses saved positions from store to preserve drag state across topology refreshes
  useEffect(() => {
    if (topology) {
      const layout = calculateGroupedLayout(
        topology.nodes,
        topology.edges,
        topology.runners,
        useMeshStore.getState().nodePositions
      );
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

  // Save Runner Group position after drag
  const onNodeDragStop: OnNodeDrag = useCallback(
    (_event, node) => {
      if (node.type === "runnerGroup") {
        updateNodePosition(node.id, node.position);
      }
    },
    [updateNodePosition]
  );

  // Handle pane click (deselect)
  const onPaneClick = useCallback(() => {
    selectNode(null);
  }, [selectNode]);

  // Node color for minimap
  const nodeColor = useCallback((node: Node) => {
    if (node.type === "runnerGroup") {
      return "#e5e7eb"; // gray for group containers
    }
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
    return <CenteredSpinner />;
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
        onNodeDragStop={onNodeDragStop}
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
