"use client";

import { useEffect } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Controls,
  Background,
  MiniMap,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useLoopalSession } from "@/stores/loopalConsole";
import LoopalAgentNode from "./LoopalAgentNode";
import { loopalTopologyLayout } from "./loopalTopologyLayout";

const nodeTypes: NodeTypes = { loopalAgent: LoopalAgentNode };

function Flow({ podKey, minimap }: { podKey: string; minimap?: boolean }) {
  const { topology } = useLoopalSession(podKey);
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // Re-layout only when the topology *structure* changes. `topology` is a fresh
  // array identity on every loopal event (refreshCache rebuilds the cache), so
  // keying the effect on the array would reset node positions — discarding the
  // user's pan/drag — on every unrelated bg-output delta.
  // Sort by agent_id for a stable structural signature: a reordered-but-unchanged
  // topology (snapshot rebuild order vs live-append order) must not re-fire the
  // layout effect and reset node positions, discarding the user's pan/drag.
  const sorted = [...topology].sort((a, b) =>
    a.agent_id < b.agent_id ? -1 : a.agent_id > b.agent_id ? 1 : 0,
  );
  const sig = JSON.stringify(sorted.map((a) => [a.agent_id, a.parent, a.name, a.model]));
  useEffect(() => {
    const l = loopalTopologyLayout(sorted);
    setNodes(l.nodes);
    setEdges(l.edges);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sig, setNodes, setEdges]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      nodeTypes={nodeTypes}
      fitView
      minZoom={0.2}
      maxZoom={1.5}
      nodesConnectable={false}
    >
      <Controls showInteractive={false} />
      {minimap && <MiniMap />}
      <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
    </ReactFlow>
  );
}

// Mini (in-dock) and full (sheet) renders share this; ReactFlowProvider is
// required because the console mounts it as an independent surface.
export function LoopalTopologyFlow({ podKey, minimap }: { podKey: string; minimap?: boolean }) {
  return (
    <ReactFlowProvider>
      <Flow podKey={podKey} minimap={minimap} />
    </ReactFlowProvider>
  );
}
