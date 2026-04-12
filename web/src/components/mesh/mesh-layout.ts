import type { Node, Edge } from "@xyflow/react";
import type { MeshNode, MeshEdge } from "@/stores/mesh";
import type { RunnerInfoData } from "@/lib/api";

const POD_WIDTH = 200;
const POD_HEIGHT = 160;
const POD_GAP_X = 20;
const POD_GAP_Y = 20;
const PODS_PER_ROW = 2;
const GROUP_PADDING_X = 20;
const GROUP_PADDING_TOP = 50;
const GROUP_PADDING_BOTTOM = 20;
const GROUP_GAP = 40;
const CANVAS_WIDTH = 1200;

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
  const cursor = computeGridStart(podsByRunner, savedPositions);

  for (const [runnerId, runnerPods] of podsByRunner) {
    const { groupNode, podNodes, nextCursor } = layoutRunnerGroup(
      runnerId, runnerPods, runnerInfoMap, savedPositions, cursor
    );
    nodes.push(groupNode, ...podNodes);
    cursor.x = nextCursor.x;
    cursor.y = nextCursor.y;
    cursor.rowMaxH = nextCursor.rowMaxH;
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

interface GridCursor { x: number; y: number; rowMaxH: number }

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

function computeGridStart(
  podsByRunner: Map<number, MeshNode[]>,
  savedPositions?: Record<string, { x: number; y: number }>
): GridCursor {
  if (!savedPositions) return { x: 0, y: 0, rowMaxH: 0 };
  let maxBottom = 0;
  for (const [runnerId, runnerPods] of podsByRunner) {
    const saved = savedPositions[`runner-group-${runnerId}`];
    if (saved) {
      const rows = Math.ceil(runnerPods.length / PODS_PER_ROW);
      const gh = GROUP_PADDING_TOP + GROUP_PADDING_BOTTOM + rows * POD_HEIGHT + (rows - 1) * POD_GAP_Y;
      maxBottom = Math.max(maxBottom, saved.y + gh + GROUP_GAP);
    }
  }
  return { x: 0, y: maxBottom, rowMaxH: 0 };
}

function calcGroupSize(podCount: number) {
  const rows = Math.ceil(podCount / PODS_PER_ROW);
  const cols = Math.min(podCount, PODS_PER_ROW);
  const width = GROUP_PADDING_X * 2 + cols * POD_WIDTH + (cols - 1) * POD_GAP_X;
  const height = GROUP_PADDING_TOP + GROUP_PADDING_BOTTOM + rows * POD_HEIGHT + (rows - 1) * POD_GAP_Y;
  return { width, height };
}

function layoutRunnerGroup(
  runnerId: number,
  runnerPods: MeshNode[],
  runnerInfoMap: Map<number, RunnerInfoData>,
  savedPositions: Record<string, { x: number; y: number }> | undefined,
  cursor: GridCursor,
) {
  const runnerInfo = runnerInfoMap.get(runnerId);
  const runnerNodeId = runnerPods[0]?.runner_node_id || `runner-${runnerId}`;
  const runnerStatus = runnerPods[0]?.runner_status || runnerInfo?.status || "offline";
  const groupId = `runner-group-${runnerId}`;
  const { width: groupWidth, height: groupHeight } = calcGroupSize(runnerPods.length);

  const savedPos = savedPositions?.[groupId];
  let nextCursor = { ...cursor };

  let position: { x: number; y: number };
  if (savedPos) {
    position = savedPos;
  } else {
    if (cursor.x + groupWidth > CANVAS_WIDTH && cursor.x > 0) {
      nextCursor = { x: 0, y: cursor.y + cursor.rowMaxH + GROUP_GAP, rowMaxH: 0 };
    }
    position = { x: nextCursor.x, y: nextCursor.y };
    nextCursor.x += groupWidth + GROUP_GAP;
    nextCursor.rowMaxH = Math.max(nextCursor.rowMaxH, groupHeight);
  }

  const groupNode: Node = {
    id: groupId,
    type: "runnerGroup",
    position,
    data: { runnerNodeId, runnerStatus, podCount: runnerPods.length },
    style: { width: groupWidth, height: groupHeight },
  };

  const podNodes: Node[] = runnerPods.map((pod, index) => {
    const podSaved = savedPositions?.[pod.pod_key];
    const col = index % PODS_PER_ROW;
    const row = Math.floor(index / PODS_PER_ROW);
    const defaultPos = {
      x: GROUP_PADDING_X + col * (POD_WIDTH + POD_GAP_X),
      y: GROUP_PADDING_TOP + row * (POD_HEIGHT + POD_GAP_Y),
    };
    return {
      id: pod.pod_key,
      type: "pod",
      position: podSaved ?? defaultPos,
      parentId: groupId,
      extent: "parent" as const,
      draggable: true,
      data: { node: pod },
    };
  });

  return { groupNode, podNodes, nextCursor };
}
