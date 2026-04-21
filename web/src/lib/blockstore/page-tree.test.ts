import { describe, it, expect } from "vitest";
import { buildPageTree } from "./page-tree";
import { BLOCK_TYPE_PAGE, type Block, type BlockRef } from "@/lib/api/blockstoreTypes";

function page(id: string, title: string): Block {
  return {
    id,
    workspace_id: "ws",
    type: BLOCK_TYPE_PAGE,
    data: { title },
    text: null,
    meta: {},
    created_by: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    deleted_at: null,
  } as unknown as Block;
}

function nestRef(id: number, from: string, to: string): BlockRef {
  return {
    id,
    workspace_id: "ws",
    from_id: from,
    to_id: to,
    rel: "nest",
    order_key: null,
    anchor: null,
    meta: {},
    created_by: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  } as unknown as BlockRef;
}

describe("buildPageTree · nest children dedupe", () => {
  // Regression: create-page previously produced a ghost nest ref (op_id used
  // as ref_id locally) plus a real nest ref (server-assigned) once the WS
  // echo arrived. The sidebar then rendered two PageNodes with the same id,
  // triggering "Encountered two children with the same key" React warnings.
  it("collapses duplicate nest refs pointing at the same to_id", () => {
    const blocks = {
      root: page("root", "Workspace"),
      childA: page("childA", "Child A"),
    };
    const refs = {
      1: nestRef(1, "root", "childA"), // ghost (synthesized locally)
      2: nestRef(2, "root", "childA"), // authoritative (from WS)
    };
    const nestChildren = { root: [1, 2] };

    const tree = buildPageTree(blocks, refs, nestChildren, "root");
    expect(tree).toHaveLength(1);
    expect(tree[0].children).toHaveLength(1);
    expect(tree[0].children[0].id).toBe("childA");
  });

  it("keeps distinct children intact", () => {
    const blocks = {
      root: page("root", "Workspace"),
      a: page("a", "A"),
      b: page("b", "B"),
    };
    const refs = {
      1: nestRef(1, "root", "a"),
      2: nestRef(2, "root", "b"),
    };
    const nestChildren = { root: [1, 2] };

    const tree = buildPageTree(blocks, refs, nestChildren, "root");
    expect(tree[0].children.map((n) => n.id)).toEqual(["a", "b"]);
  });
});
