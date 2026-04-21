import { invoke } from "./invoke";

/**
 * Desktop proxy for the Rust Blockstore service. The service lives in the
 * main process; renderer needs async IPC for writes AND a local cache so
 * that the zustand store's sync readers (`list_children_json`, etc.) can
 * return data without a round-trip.
 *
 * After each async populator (load_subtree / load_type_defs / ensure_default),
 * we eagerly hydrate the cache by pulling the relevant JSON from the main
 * process. Sync getters then read from cache.
 */
export class ElectronBlockstoreService {
  private blockCache = new Map<string, string>();
  private childrenCache = new Map<string, string>();
  private typeDefsCache = new Map<string, string>();
  private backlinksCache = new Map<string, string>();

  // Flat-map caches for new SSOT API (blocks_json / refs_json / ...).
  // Main process owns SSOT; renderer mirrors it on every populator call
  // so sync zustand selectors can read without round-trip.
  private blocksJson = "{}";
  private refsJson = "{}";
  private nestChildrenJson = "{}";
  private backlinksJsonFlat = "{}";
  private lastOpIdsJson = "{}";
  private workspacesJsonCache = "{}";

  private async refreshFlatCaches(): Promise<void> {
    const [blocks, refs, nestChildren, backlinks, lastOpIds, workspaces] = await Promise.all([
      invoke<string>("blockstoreBlocksJson"),
      invoke<string>("blockstoreRefsJson"),
      invoke<string>("blockstoreNestChildrenJson"),
      invoke<string>("blockstoreBacklinksJson"),
      invoke<string>("blockstoreLastOpIdsJson"),
      invoke<string>("blockstoreWorkspacesJson"),
    ]);
    this.blocksJson = blocks || "{}";
    this.refsJson = refs || "{}";
    this.nestChildrenJson = nestChildren || "{}";
    this.backlinksJsonFlat = backlinks || "{}";
    this.lastOpIdsJson = lastOpIds || "{}";
    this.workspacesJsonCache = workspaces || "{}";
  }

  async apply_ops(reqJson: string): Promise<string> {
    const result = await invoke<string>("blockstoreApplyOps", reqJson);
    await this.refreshFlatCaches();
    return result;
  }

  async list_workspaces(): Promise<string> {
    const result = await invoke<string>("blockstoreListWorkspaces");
    await this.refreshFlatCaches();
    return result;
  }

  async ensure_default_workspace(): Promise<string> {
    const result = await invoke<string>("blockstoreEnsureDefaultWorkspace");
    await this.refreshFlatCaches();
    return result;
  }

  async load_subtree(workspaceId: string, rootId: string): Promise<void> {
    await invoke("blockstoreLoadSubtree", workspaceId, rootId);
    await this.hydrateSubtree(rootId);
    await this.refreshFlatCaches();
  }

  async load_type_defs(workspaceId: string): Promise<void> {
    await invoke("blockstoreLoadTypeDefs", workspaceId);
    const json = await invoke<string>("blockstoreTypeDefsJson", workspaceId);
    this.typeDefsCache.set(workspaceId, json);
  }

  async catchup(workspaceId: string): Promise<void> {
    await invoke("blockstoreCatchup", workspaceId);
  }

  async semantic_search(workspaceId: string, reqJson: string): Promise<string> {
    return invoke<string>("blockstoreSemanticSearch", workspaceId, reqJson);
  }

  apply_remote_op(opJson: string): void {
    // Rust BlockOp.applied_at is typed as String, but backend WS pushes
    // ints (Unix ms). Coerce before IPC to avoid serde deserialize failure.
    let normalized = opJson;
    try {
      const op = JSON.parse(opJson) as Record<string, unknown>;
      if (typeof op.applied_at === "number") {
        op.applied_at = new Date(op.applied_at).toISOString();
        normalized = JSON.stringify(op);
      }
    } catch {
      // fall through with original payload
    }
    void invoke("blockstoreApplyRemoteOp", normalized).then(() => this.refreshFlatCaches());
  }

  workspaces_json(): string {
    return this.workspacesJsonCache;
  }

  // New flat-map SSOT getters — mirror main-process state via async hydrate.
  blocks_json(): string { return this.blocksJson; }
  refs_json(): string { return this.refsJson; }
  nest_children_json(): string { return this.nestChildrenJson; }
  backlinks_json(): string { return this.backlinksJsonFlat; }
  last_op_ids_json(): string { return this.lastOpIdsJson; }

  get_block_json(id: string): string | null {
    return this.blockCache.get(id) ?? null;
  }

  list_children_json(parentId: string): string {
    return this.childrenCache.get(parentId) ?? '{"blocks":[],"refs":[]}';
  }

  list_backlinks_json(targetId: string): string {
    return this.backlinksCache.get(targetId) ?? '{"refs":[]}';
  }

  type_defs_json(workspaceId: string): string {
    return this.typeDefsCache.get(workspaceId) ?? '{"blocks":[]}';
  }

  async last_op_id(workspaceId: string): Promise<bigint> {
    return invoke<bigint>("blockstoreLastOpId", workspaceId);
  }

  async set_last_op_id(workspaceId: string, id: bigint): Promise<void> {
    await invoke("blockstoreSetLastOpId", workspaceId, id);
  }

  private async hydrateSubtree(rootId: string): Promise<void> {
    const seen = new Set<string>();
    await this.hydrateNode(rootId, seen);
  }

  private async hydrateNode(id: string, seen: Set<string>): Promise<void> {
    if (seen.has(id)) return;
    seen.add(id);

    const [blockJson, childrenJson] = await Promise.all([
      invoke<string | null>("blockstoreGetBlockJson", id),
      invoke<string>("blockstoreListChildrenJson", id),
    ]);
    if (blockJson) this.blockCache.set(id, blockJson);
    this.childrenCache.set(id, childrenJson);

    try {
      const { blocks } = JSON.parse(childrenJson) as { blocks?: Array<{ id: string }> };
      if (!blocks?.length) return;
      // Cache child block JSON, then recurse to populate their children.
      for (const b of blocks) {
        this.blockCache.set(b.id, JSON.stringify(b));
      }
      await Promise.all(blocks.map((b) => this.hydrateNode(b.id, seen)));
    } catch {
      // Malformed payload — leave node cached with what we already have.
    }
  }
}
