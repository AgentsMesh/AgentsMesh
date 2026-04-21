import { beforeEach, describe, expect, it, vi } from "vitest";

// F5 guard: handleBlockstoreEvent must drop ops whose workspace_id was not
// seeded into Rust state's last_op_ids via loadSubtree. The org-wide WS broadcast
// delivers ops from every workspace the user's org hosts; filtering here is the
// line that keeps an unrelated workspace's data out of this client's store.

interface Bucket {
  blocks: Map<string, Record<string, unknown>>;
  lastOpIds: Map<string, number>;
}

const bucket: Bucket = { blocks: new Map(), lastOpIds: new Map() };

vi.mock("@/lib/wasm-core", async () => {
  const actual = await vi.importActual<typeof import("@/lib/wasm-core")>("@/lib/wasm-core");
  return {
    ...actual,
    getBlockstoreService: () => ({
      last_op_ids_json: () => {
        const obj: Record<string, number> = {};
        bucket.lastOpIds.forEach((v, k) => (obj[k] = v));
        return JSON.stringify(obj);
      },
      set_last_op_id: (wsID: string, id: number) => {
        bucket.lastOpIds.set(wsID, id);
      },
      apply_remote_op: (opJson: string) => {
        const op = JSON.parse(opJson);
        if (op.op === "createBlock") {
          bucket.blocks.set(op.forward.id, op.forward);
        }
      },
      get_block_json: (id: string) => {
        const b = bucket.blocks.get(id);
        return b ? JSON.stringify(b) : null;
      },
    }),
  };
});

import { useBlockstoreStore, readBlock } from "./blockstore";
import { handleBlockstoreEvent } from "./blockstoreSubscribe";

describe("handleBlockstoreEvent workspace-scope filter (F5)", () => {
  beforeEach(() => {
    bucket.blocks.clear();
    bucket.lastOpIds.clear();
    useBlockstoreStore.getState().actions.reset();
  });

  it("applies an op for a subscribed workspace", () => {
    const wsID = "ws-subscribed";
    bucket.lastOpIds.set(wsID, 0);

    handleBlockstoreEvent({
      type: "blockstore:op",
      data: {
        id: 1,
        workspace_id: wsID,
        op: "createBlock",
        forward: { id: "block-1", type: "paragraph", data: { text: "hello" } },
        applied_at: "2026-04-18T00:00:00Z",
      },
    } as never);

    expect(readBlock("block-1")).toBeTruthy();
  });

  it("drops an op for an unsubscribed workspace", () => {
    bucket.lastOpIds.set("ws-subscribed", 0);

    handleBlockstoreEvent({
      type: "blockstore:op",
      data: {
        id: 2,
        workspace_id: "ws-other",
        op: "createBlock",
        forward: { id: "leak-block", type: "paragraph", data: {} },
        applied_at: "2026-04-18T00:00:00Z",
      },
    } as never);

    expect(readBlock("leak-block")).toBeNull();
  });

  it("ignores non-blockstore event types", () => {
    bucket.lastOpIds.set("ws-subscribed", 0);
    handleBlockstoreEvent({ type: "pod:created", data: {} } as never);
    expect(bucket.blocks.size).toBe(0);
  });
});
