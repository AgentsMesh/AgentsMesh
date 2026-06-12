import type { Node, Edge } from "@xyflow/react";
import type { LoopalAgentNode } from "@/stores/loopalConsole";

const NODE_W = 180;
const NODE_H = 64;
const GAP_X = 24;
const GAP_Y = 72;

// Tidy hierarchical layout for the agent topology tree. Loopal emits `parent`
// as a node *name* (parent.agent), so parent resolution is name-based and
// first-wins when names repeat; every other index (pos/visited/children/edges)
// is keyed by the unique agent_id so same-named agents stay distinct nodes and
// edges. A visited set guards cycles, and a final pass places any node
// unreachable from a root (orphan/cycle) so nothing collapses onto (0,0).
export function loopalTopologyLayout(agents: LoopalAgentNode[]): { nodes: Node[]; edges: Edge[] } {
  if (agents.length === 0) return { nodes: [], edges: [] };

  const idOf = (a: LoopalAgentNode) => a.agent_id || a.name;
  const byName = new Map<string, LoopalAgentNode>();
  for (const a of agents) if (!byName.has(a.name)) byName.set(a.name, a);
  const parentOf = (a: LoopalAgentNode) => (a.parent ? byName.get(a.parent) : undefined);
  const isRoot = (a: LoopalAgentNode) => !parentOf(a);

  const children = new Map<string, LoopalAgentNode[]>();
  for (const a of agents) {
    const p = parentOf(a);
    if (p) {
      const list = children.get(idOf(p)) ?? [];
      list.push(a);
      children.set(idOf(p), list);
    }
  }

  const pos = new Map<string, { x: number; y: number }>();
  const visited = new Set<string>();
  let leafX = 0;

  function place(a: LoopalAgentNode, depth: number): number {
    const id = idOf(a);
    if (visited.has(id)) return pos.get(id)?.x ?? 0;
    visited.add(id);
    const kids = children.get(id) ?? [];
    let x: number;
    if (kids.length === 0) {
      x = leafX * (NODE_W + GAP_X);
      leafX += 1;
    } else {
      const xs = kids.map((k) => place(k, depth + 1));
      x = (Math.min(...xs) + Math.max(...xs)) / 2;
    }
    pos.set(id, { x, y: depth * (NODE_H + GAP_Y) });
    return x;
  }
  for (const r of agents.filter(isRoot)) place(r, 0);
  for (const a of agents) if (!visited.has(idOf(a))) place(a, 0);

  const nodes: Node[] = agents.map((a) => ({
    id: idOf(a),
    type: "loopalAgent",
    position: pos.get(idOf(a)) ?? { x: 0, y: 0 },
    data: { name: a.name, model: a.model, isRoot: isRoot(a) },
  }));

  const edges: Edge[] = agents
    .map((a): Edge | null => {
      const p = parentOf(a);
      return p ? { id: `${idOf(p)}->${idOf(a)}`, source: idOf(p), target: idOf(a) } : null;
    })
    .filter((e): e is Edge => e !== null);

  return { nodes, edges };
}
