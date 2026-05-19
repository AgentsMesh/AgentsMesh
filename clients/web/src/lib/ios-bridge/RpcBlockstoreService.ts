// Mirrors packages/electron-adapter/src/blockstore.ts ElectronBlockstoreService:
// async writes via bridge to native Rust SSOT, sync flat-map getters via mirror.

let nextId = 1;
const pending = new Map<number, { resolve: (v: unknown) => void; reject: (e: Error) => void }>();

function ensureBridgeWired() {
  const w = window as unknown as {
    __amResolve?: (id: number, payload: { result?: unknown; error?: string }) => void;
  };
  if (w.__amResolve) return;
  w.__amResolve = (id, payload) => {
    const p = pending.get(id);
    pending.delete(id);
    if (!p) return;
    if (payload.error) p.reject(new Error(payload.error));
    else p.resolve(payload.result);
  };
}

interface IosBridge {
  postMessage: (msg: unknown) => void;
}
function bridge(): IosBridge {
  const w = window as unknown as { webkit?: { messageHandlers?: { amBridge?: IosBridge } } };
  const b = w.webkit?.messageHandlers?.amBridge;
  if (!b) throw new Error("iOS amBridge not available — running outside embed mode?");
  return b;
}

export function rpc<T>(method: string, args: Record<string, unknown> = {}): Promise<T> {
  ensureBridgeWired();
  const id = nextId++;
  return new Promise<T>((resolve, reject) => {
    pending.set(id, { resolve: resolve as (v: unknown) => void, reject });
    bridge().postMessage({ id, method, args });
  });
}

export class RpcBlockstoreService {
  private _workspacesJson = "{}";
  private _blocksJson = "{}";
  private _refsJson = "{}";
  private _nestChildrenJson = "{}";
  private _backlinksJson = "{}";
  private _lastOpIdsJson = "{}";

  private async refreshFlatCaches(): Promise<void> {
    const [w, b, r, nc, bl, lo] = await Promise.all([
      rpc<string>("workspaces_json"),
      rpc<string>("blocks_json"),
      rpc<string>("refs_json"),
      rpc<string>("nest_children_json"),
      rpc<string>("backlinks_json"),
      rpc<string>("last_op_ids_json"),
    ]);
    this._workspacesJson = w || "{}";
    this._blocksJson = b || "{}";
    this._refsJson = r || "{}";
    this._nestChildrenJson = nc || "{}";
    this._backlinksJson = bl || "{}";
    this._lastOpIdsJson = lo || "{}";
  }

  async apply_ops(reqJson: string): Promise<string> {
    const r = await rpc<unknown>("apply_ops", { req: JSON.parse(reqJson) });
    await this.refreshFlatCaches();
    return JSON.stringify(r);
  }

  async list_workspaces(): Promise<string> {
    const r = await rpc<unknown>("list_workspaces");
    await this.refreshFlatCaches();
    return JSON.stringify(r);
  }

  async ensure_default_workspace(): Promise<string> {
    const r = await rpc<unknown>("ensure_default_workspace");
    await this.refreshFlatCaches();
    return JSON.stringify(r);
  }

  async load_subtree(wsId: string, rootId: string): Promise<void> {
    await rpc("load_subtree", { wsId, rootId });
    await this.refreshFlatCaches();
  }

  async load_type_defs(wsId: string): Promise<void> {
    await rpc("load_type_defs", { wsId });
  }

  async catchup(wsId: string): Promise<void> {
    await rpc("catchup", { wsId });
    await this.refreshFlatCaches();
  }

  async semantic_search(wsId: string, queryJson: string): Promise<string> {
    const r = await rpc<unknown>("semantic_search", { wsId, query: JSON.parse(queryJson) });
    return JSON.stringify(r);
  }

  apply_remote_op(opJson: string): void {
    void rpc("apply_remote_op", { op: JSON.parse(opJson) }).then(() => this.refreshFlatCaches());
  }

  set_last_op_id(wsId: string, id: number): void {
    void rpc("set_last_op_id", { wsId, id });
  }

  workspaces_json(): string { return this._workspacesJson; }
  blocks_json(): string { return this._blocksJson; }
  refs_json(): string { return this._refsJson; }
  nest_children_json(): string { return this._nestChildrenJson; }
  backlinks_json(): string { return this._backlinksJson; }
  last_op_ids_json(): string { return this._lastOpIdsJson; }

  get_block_json(id: string): string | null {
    const map = safeParse<Record<string, unknown>>(this._blocksJson);
    const block = map?.[id];
    return block ? JSON.stringify(block) : null;
  }

  list_children_json(id: string): string {
    const idx = safeParse<Record<string, unknown[]>>(this._nestChildrenJson) ?? {};
    const blocks = safeParse<Record<string, unknown>>(this._blocksJson) ?? {};
    const refs = safeParse<Record<string, unknown>>(this._refsJson) ?? {};
    const childRefIds = (idx[id] as unknown[] | undefined) ?? [];
    const childBlocks: unknown[] = [];
    const childRefs: unknown[] = [];
    for (const rid of childRefIds as Array<string | number>) {
      const ref = refs[String(rid)] as { to_id?: string } | undefined;
      if (!ref) continue;
      childRefs.push(ref);
      const child = ref.to_id ? blocks[ref.to_id] : undefined;
      if (child) childBlocks.push(child);
    }
    return JSON.stringify({ blocks: childBlocks, refs: childRefs });
  }

  list_backlinks_json(id: string): string {
    const idx = safeParse<Record<string, unknown[]>>(this._backlinksJson) ?? {};
    const refs = safeParse<Record<string, unknown>>(this._refsJson) ?? {};
    const refIds = (idx[id] as unknown[] | undefined) ?? [];
    const out: unknown[] = [];
    for (const rid of refIds as Array<string | number>) {
      const ref = refs[String(rid)];
      if (ref) out.push(ref);
    }
    return JSON.stringify({ refs: out });
  }

  type_defs_json(_wsId: string): string {
    const blocks = safeParse<Record<string, { workspace_id?: string; type?: string }>>(this._blocksJson) ?? {};
    const out: unknown[] = [];
    for (const b of Object.values(blocks)) {
      if (b?.workspace_id === _wsId && b?.type === "type-def") out.push(b);
    }
    return JSON.stringify({ blocks: out });
  }
}

function safeParse<T>(s: string): T | undefined {
  try { return JSON.parse(s) as T; } catch { return undefined; }
}

export async function primeSubtreeCache(wsId: string, rootId: string): Promise<void> {
  await rpc("load_subtree", { wsId, rootId });
}
