"use client";

import { useCallback, useEffect } from "react";
import {
  ReactFlow, Controls, Background, MiniMap,
  useNodesState, useEdgesState,
  type Node, type Edge, type NodeTypes, type EdgeTypes, type OnNodeDrag,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { CenteredSpinner } from "@/components/ui/spinner";
import PodNode from "./PodNode";
import BindingEdge from "./BindingEdge";
import RunnerGroupNode from "./RunnerGroupNode";
import { useMeshStore, type MeshNode } from "@/stores/mesh";
import { calculateGroupedLayout } from "./mesh-layout";

const nodeTypes: NodeTypes = { pod: PodNode, runnerGroup: RunnerGroupNode };
const edgeTypes: EdgeTypes = { binding: BindingEdge };

export default function MeshTopology() {
  const { topology, selectedNode, selectNode, fetchTopology, updateNodePosition } = useMeshStore();
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  useEffect(() => { fetchTopology(); }, [fetchTopology]);

  useEffect(() => {
    if (topology) {
      const layout = calculateGroupedLayout(topology.nodes, topology.edges, topology.runners, useMeshStore.getState().nodePositions);
      setNodes(layout.nodes);
      setEdges(layout.edges);
    }
  }, [topology, setNodes, setEdges]);

  useEffect(() => {
    setNodes((nds) => nds.map((node) =>
      node.type === "pod" ? { ...node, data: { ...node.data, isSelected: node.id === selectedNode } } : node
    ));
  }, [selectedNode, setNodes]);

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    if (node.type === "pod") selectNode(node.id);
  }, [selectNode]);

  const onNodeDragStop: OnNodeDrag = useCallback((_event, node) => {
    if (node.type === "runnerGroup" || node.type === "pod") updateNodePosition(node.id, node.position);
  }, [updateNodePosition]);

  const onPaneClick = useCallback(() => { selectNode(null); }, [selectNode]);

  const nodeColor = useCallback((node: Node) => {
    if (node.type === "runnerGroup") return "#e5e7eb";
    const data = node.data as { node: MeshNode };
    switch (data.node?.status) {
      case "running": return "#22c55e";
      case "initializing": return "#eab308";
      case "failed": return "#ef4444";
      default: return "#6b7280";
    }
  }, []);

  if (!topology) return <CenteredSpinner />;

  if (topology.nodes.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <svg className="w-16 h-16 mx-auto text-muted-foreground mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
          </svg>
          <h3 className="text-lg font-medium text-foreground mb-2">No Active Pods</h3>
          <p className="text-muted-foreground">Start an AgentPod to see it in the mesh</p>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full h-full">
      <ReactFlow nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
        onNodeClick={onNodeClick} onNodeDragStop={onNodeDragStop} onPaneClick={onPaneClick}
        nodeTypes={nodeTypes} edgeTypes={edgeTypes} fitView minZoom={0.1} maxZoom={2}
        defaultViewport={{ x: 0, y: 0, zoom: 1 }}>
        <Controls />
        <MiniMap nodeColor={nodeColor} zoomable pannable />
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
      </ReactFlow>
    </div>
  );
}
