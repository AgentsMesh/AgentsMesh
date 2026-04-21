import { getBlockstoreService, parseWasmAny } from "@/lib/wasm-core";
import type {
  ApplyOpsRequest,
  ApplyOpsResult,
  Block,
  BlockOp,
  BlockRef,
  ChildrenResult,
  SearchHit,
  Workspace,
} from "@/lib/api/blockstoreTypes";

const svc = () => getBlockstoreService();

export const blockstoreApi = {
  async applyOps(req: ApplyOpsRequest): Promise<ApplyOpsResult> {
    const json = await svc().apply_ops(JSON.stringify(req));
    return JSON.parse(json) as ApplyOpsResult;
  },

  async listWorkspaces(): Promise<{ workspaces: Workspace[] }> {
    const json = await svc().list_workspaces();
    return JSON.parse(json) as { workspaces: Workspace[] };
  },

  async ensureDefaultWorkspace(): Promise<Workspace> {
    const json = await svc().ensure_default_workspace();
    return JSON.parse(json) as Workspace;
  },

  async getBlock(id: string): Promise<Block> {
    // WASM SSOT: load_subtree populates the cache. For standalone reads we
    // fall back to a fresh fetch via the client — here we just read from
    // the cache (callers typically load a subtree first).
    const raw = parseWasmAny<Block>(svc().get_block_json(id));
    if (!raw) throw new Error(`block not found: ${id}`);
    return raw;
  },

  async listChildren(id: string, _rel = "nest"): Promise<ChildrenResult> {
    // Read from WASM cache after a subtree load; equivalent shape.
    const raw = svc().list_children_json(id);
    return (JSON.parse(raw) as ChildrenResult) ?? { blocks: [], refs: [] };
  },

  async listBacklinks(id: string): Promise<{ refs: BlockRef[] }> {
    const raw = svc().list_backlinks_json(id);
    return (JSON.parse(raw) as { refs: BlockRef[] }) ?? { refs: [] };
  },

  async getSubtree(wsID: string, rootID: string, _maxDepth = 64): Promise<ChildrenResult> {
    await svc().load_subtree(wsID, rootID);
    const raw = svc().list_children_json(rootID);
    const result = (JSON.parse(raw) as ChildrenResult) ?? { blocks: [], refs: [] };
    // list_children_json returns the root's children — the root itself is
    // loaded into WASM state but never in this payload. Pull it explicitly so
    // callers (DocumentView) can render the root node.
    try {
      const rootJson = svc().get_block_json(rootID);
      const root = parseWasmAny<Block>(rootJson);
      if (root && !result.blocks.some((b) => b.id === root.id)) {
        result.blocks = [root, ...result.blocks];
      }
    } catch {
      // Root not present yet (fresh workspace before first op) — tolerate.
    }
    return result;
  },

  async catchupOps(wsID: string, _after = 0, _limit = 200): Promise<{ ops: BlockOp[] }> {
    await svc().catchup(wsID);
    // The server-authoritative ops have been applied to WASM state. Callers
    // of catchupOps historically iterated ops to feed applyRemoteOp; with the
    // WASM path the state is already converged. Return an empty list so legacy
    // callers don't double-apply.
    return { ops: [] };
  },

  async listTypeDefs(wsID: string): Promise<{ blocks: Block[] }> {
    await svc().load_type_defs(wsID);
    const raw = svc().type_defs_json(wsID);
    return (JSON.parse(raw) as { blocks: Block[] }) ?? { blocks: [] };
  },

  async semanticSearch(
    wsID: string,
    query: string,
    opts: { topK?: number; minScore?: number; type?: string } = {},
  ): Promise<{ hits: SearchHit[] }> {
    const req = {
      query,
      ...(opts.topK !== undefined ? { top_k: opts.topK } : {}),
      ...(opts.minScore !== undefined ? { min_score: opts.minScore } : {}),
      ...(opts.type !== undefined ? { type: opts.type } : {}),
    };
    const json = await svc().semantic_search(wsID, JSON.stringify(req));
    return JSON.parse(json) as { hits: SearchHit[] };
  },
};
