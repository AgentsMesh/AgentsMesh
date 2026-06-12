import { describe, it, expect } from "vitest";
import { loopalTopologyLayout } from "../topology/loopalTopologyLayout";
import type { LoopalAgentNode } from "@/stores/loopalConsole";

function agent(
  name: string,
  parent: string | null,
  agent_id = name,
  model: string | null = null,
): LoopalAgentNode {
  return { name, parent, agent_id, model };
}

describe("loopalTopologyLayout", () => {
  it("returns empty for no agents", () => {
    expect(loopalTopologyLayout([])).toEqual({ nodes: [], edges: [] });
  });

  it("places a root above its child and links them", () => {
    const { nodes, edges } = loopalTopologyLayout([agent("main", null), agent("worker", "main")]);
    const main = nodes.find((n) => n.id === "main")!;
    const worker = nodes.find((n) => n.id === "worker")!;
    expect(main.position.y).toBeLessThan(worker.position.y);
    expect(edges).toHaveLength(1);
    expect(edges[0]).toMatchObject({ source: "main", target: "worker" });
  });

  it("spreads sibling leaves and centers the parent over them", () => {
    const { nodes } = loopalTopologyLayout([
      agent("main", null),
      agent("a", "main"),
      agent("b", "main"),
    ]);
    const main = nodes.find((n) => n.id === "main")!;
    const a = nodes.find((n) => n.id === "a")!;
    const b = nodes.find((n) => n.id === "b")!;
    expect(a.position.x).not.toEqual(b.position.x);
    expect(main.position.x).toBeCloseTo((a.position.x + b.position.x) / 2);
  });

  it("flags root vs child nodes", () => {
    const { nodes } = loopalTopologyLayout([agent("main", null), agent("worker", "main")]);
    expect((nodes.find((n) => n.id === "main")!.data as { isRoot: boolean }).isRoot).toBe(true);
    expect((nodes.find((n) => n.id === "worker")!.data as { isRoot: boolean }).isRoot).toBe(false);
  });

  it("terminates on a cycle instead of looping forever", () => {
    const { nodes } = loopalTopologyLayout([agent("a", "b"), agent("b", "a")]);
    expect(nodes).toHaveLength(2);
  });

  it("places every node in a rootless cycle (no origin collapse)", () => {
    const { nodes } = loopalTopologyLayout([agent("a", "b", "ida"), agent("b", "a", "idb")]);
    const a = nodes.find((n) => n.id === "ida")!;
    const b = nodes.find((n) => n.id === "idb")!;
    expect(a.position).not.toEqual(b.position);
  });

  it("keeps same-named sibling agents as distinct nodes (dedup by agent_id)", () => {
    const { nodes, edges } = loopalTopologyLayout([
      agent("main", null, "id-main"),
      agent("worker", "main", "w1"),
      agent("worker", "main", "w2"),
    ]);
    expect(nodes.map((n) => n.id).sort()).toEqual(["id-main", "w1", "w2"]);
    expect(edges.map((e) => `${e.source}->${e.target}`).sort()).toEqual([
      "id-main->w1",
      "id-main->w2",
    ]);
  });

  it("resolves a same-named parent deterministically (first spawn wins)", () => {
    const { nodes, edges } = loopalTopologyLayout([
      agent("lead", null, "lead1"),
      agent("lead", null, "lead2"),
      agent("child", "lead", "c1"),
    ]);
    expect(nodes).toHaveLength(3);
    expect(edges).toHaveLength(1);
    expect(edges[0]).toMatchObject({ source: "lead1", target: "c1" });
  });

  it("uses agent_id as node id when present", () => {
    const { nodes, edges } = loopalTopologyLayout([
      agent("main", null, "id-main"),
      agent("worker", "main", "id-worker"),
    ]);
    expect(nodes.map((n) => n.id).sort()).toEqual(["id-main", "id-worker"]);
    expect(edges[0]).toMatchObject({ source: "id-main", target: "id-worker" });
  });
});
