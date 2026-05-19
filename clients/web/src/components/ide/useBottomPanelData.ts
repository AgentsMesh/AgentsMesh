"use client";

import { useMemo, useCallback } from "react";
import { useWorkspaceStore } from "@/stores/workspace";
import { useMeshStore, useTopology, type MeshEdge, type ChannelInfo, type MeshNode } from "@/stores/mesh";
import { usePod } from "@/stores/pod";
import { useAutopilotStore, useAutopilotControllers, type AutopilotController } from "@/stores/autopilot";

export function useBottomPanelData() {
  const panes = useWorkspaceStore((s) => s.panes);
  const activePane = useWorkspaceStore((s) => s.activePane);
  const topology = useTopology();
  const fetchTopology = useMeshStore((s) => s.fetchTopology);

  const selectedPodKey = useMemo(() => {
    if (!activePane) return null;
    const pane = panes.find((p) => p.id === activePane);
    return pane?.podKey ?? null;
  }, [activePane, panes]);

  const currentPod = usePod(selectedPodKey ?? undefined) ?? null;

  const activePhases = ["initializing", "running", "paused", "user_takeover", "waiting_approval"];
  const allControllers = useAutopilotControllers();
  const activeAutopilot = selectedPodKey
    ? allControllers.find((c: AutopilotController) => c.pod_key === selectedPodKey && activePhases.includes(c.phase))
    : undefined;

  const podChannels = useMemo(() => {
    if (!selectedPodKey || !topology) return [];
    return topology.channels.filter((c: ChannelInfo) => c.pod_keys.includes(selectedPodKey));
  }, [selectedPodKey, topology]);

  const podEdges = useMemo(() => {
    if (!selectedPodKey || !topology) return [];
    return topology.edges.filter((e: MeshEdge) => e.source === selectedPodKey || e.target === selectedPodKey);
  }, [selectedPodKey, topology]);

  const { incomingBindings, outgoingBindings } = useMemo(() => {
    const incoming: MeshEdge[] = [];
    const outgoing: MeshEdge[] = [];
    podEdges.forEach((edge: MeshEdge) => {
      if (edge.target === selectedPodKey) incoming.push(edge);
      else if (edge.source === selectedPodKey) outgoing.push(edge);
    });
    return { incomingBindings: incoming, outgoingBindings: outgoing };
  }, [podEdges, selectedPodKey]);

  const getPodInfo = useCallback(
    (podKey: string) => topology?.nodes.find((n: MeshNode) => n.pod_key === podKey),
    [topology]
  );

  return {
    selectedPodKey, currentPod, activeAutopilot,
    topology, fetchTopology,
    podChannels, incomingBindings, outgoingBindings, getPodInfo,
  };
}
