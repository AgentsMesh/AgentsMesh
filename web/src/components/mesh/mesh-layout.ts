import type { Node, Edge } from "@xyflow/react";
import type { MeshNode, MeshEdge } from "@/stores/mesh";
import type { RunnerInfoData } from "@/lib/api";

// Layout constants
const POD_WIDTH = 200;
const POD_HEIGHT = 140;
const POD_GAP_X = 20;
const POD_GAP_Y = 20;
const PODS_PER_ROW = 2;
const GROUP_PADDING_X = 20;
const GROUP_PADDING_TOP = 50;
const GROUP_PADDING_BOTTOM = 20;
const GROUP_GAP = 40;

/**
 * Grouped layout algorithm: arranges pods into Runner "workstation" groups.
 */
export function calculateGroupedLayout(
  pods: MeshNode[],
  edges: MeshEdge[],
  runners?: RunnerInfoData[],
  savedPositions?: Record<string, { x: number; y: number }>
): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = [];
  const flowEdges: Edge[] = [];

  const podsByRunner = groupPodsByRunner(pods);
  const runnerInfoMap = buildRunnerInfoMap(runners);

  let autoGroupX = computeAutoGroupX(podsByRunner, savedPositions);

  for (const [runnerId, runnerPods] of podsByRunner) {
    const { groupNode, podNodes, nextAutoX } = layoutRunnerGroup(
      runnerId, runnerPods, runnerInfoMap, savedPositions, autoGroupX
    );
    nodes.push(groupNode, ...podNodes);
    autoGroupX = nextAutoX;
  }

  for (const edge of edges) {
    flowEdges.push({
      id: `binding-${edge.id}-${edge.source}-${edge.target}`,
      source: edge.source,
      target: edge.target,
      type: "binding",
      data: { status: edge.status, grantedScopes: edge.granted_scopes, pendingScopes: edge.pending_scopes },
    });
  }

  return { nodes, edges: flowEdges };
}

function groupPodsByRunner(pods: MeshNode[]): Map<number, MeshNode[]> {
  const map = new Map<number, MeshNode[]>();
  for (const pod of pods) {
    if (!map.has(pod.runner_id)) map.set(pod.runner_id, []);
    map.get(pod.runner_id)!.push(pod);
  }
  return map;
}

function buildRunnerInfoMap(runners?: RunnerInfoData[]): Map<number, RunnerInfoData> {
  const map = new Map<number, RunnerInfoData>();
  if (runners) for (const r of runners) map.set(r.id, r);
  return map;
}

function computeAutoGroupX(
  podsByRunner: Map<number, MeshNode[]>,
  savedPositions?: Record<string, { x: number; y: number }>
): number {
  let autoGroupX = 0;
  if (!savedPositions) return autoGroupX;
  for (const [runnerId, runnerPods] of podsByRunner) {
    const saved = savedPositions[`runner-group-${runnerId}`];
    if (saved) {
      const cols = Math.min(runnerPods.length, PODS_PER_ROW);
      const gw = GROUP_PADDING_X * 2 + cols * POD_WIDTH + (cols - 1) * POD_GAP_X;
      autoGroupX = Math.max(autoGroupX, saved.x + gw + GROUP_GAP);
    }
  }
  return autoGroupX;
}

function layoutRunnerGroup(
  runnerId: number,
  runnerPods: MeshNode[],
  runnerInfoMap: Map<number, RunnerInfoData>,
  savedPositions: Record<string, { x: number; y: number }> | undefined,
  autoGroupX: number,
) {
  const runnerInfo = runnerInfoMap.get(runnerId);
  const runnerNodeId = runnerPods[0]?.runner_node_id || `runner-${runnerId}`;
  const runnerStatus = runnerPods[0]?.runner_status || runnerInfo?.status || "offline";
  const groupId = `runner-group-${runnerId}`;

  const rows = Math.ceil(runnerPods.length / PODS_PER_ROW);
  const cols = Math.min(runnerPods.length, PODS_PER_ROW);
  const groupWidth = GROUP_PADDING_X * 2 + cols * POD_WIDTH + (cols - 1) * POD_GAP_X;
  const groupHeight = GROUP_PADDING_TOP + GROUP_PADDING_BOTTOM + rows * POD_HEIGHT + (rows - 1) * POD_GAP_Y;

  const savedPos = savedPositions?.[groupId];
  const position = savedPos ?? { x: autoGroupX, y: 0 };

  const groupNode: Node = {
    id: groupId,
    type: "runnerGroup",
    position,
    data: { runnerNodeId, runnerStatus, podCount: runnerPods.length },
    style: { width: groupWidth, height: groupHeight },
  };

  const podNodes: Node[] = runnerPods.map((pod, index) => {
    const col = index % PODS_PER_ROW;
    const row = Math.floor(index / PODS_PER_ROW);
    return {
      id: pod.pod_key,
      type: "pod",
      position: { x: GROUP_PADDING_X + col * (POD_WIDTH + POD_GAP_X), y: GROUP_PADDING_TOP + row * (POD_HEIGHT + POD_GAP_Y) },
      parentId: groupId,
      extent: "parent" as const,
      draggable: false,
      data: { node: pod },
    };
  });

  return { groupNode, podNodes, nextAutoX: savedPos ? autoGroupX : autoGroupX + groupWidth + GROUP_GAP };
}
