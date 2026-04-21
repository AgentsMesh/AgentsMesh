import { BLOCK_TYPE_PAGE, type Block, type BlockRef } from "@/lib/api/blockstoreTypes";

export interface PageNode {
  id: string;
  title: string;
  icon?: string;
  children: PageNode[];
}

/**
 * Walk the nest tree rooted at `rootId`, keeping only blocks of type "page".
 * `nestChildren` stores nest-relation **ref** ids (not block ids); each ref
 * maps to a child block via `refs[refId].to_id`.
 */
export function buildPageTree(
  blocks: Record<string, Block>,
  refs: Record<number, BlockRef>,
  nestChildren: Record<string, number[]>,
  rootId: string,
): PageNode[] {
  const root = blocks[rootId];
  if (!root) return [];
  if (root.type !== BLOCK_TYPE_PAGE) {
    return childPages(blocks, refs, nestChildren, rootId);
  }
  const node = buildOne(blocks, refs, nestChildren, rootId);
  return node ? [node] : [];
}

function childPages(
  blocks: Record<string, Block>,
  refs: Record<number, BlockRef>,
  nestChildren: Record<string, number[]>,
  parentId: string,
): PageNode[] {
  const refIds = nestChildren[parentId] ?? [];
  const out: PageNode[] = [];
  // Defensive dedupe: if multiple nest refs point at the same to_id (e.g.
  // a ghost from a skipped-but-stale local apply, or a server resend), keep
  // only the first. The Rust SSOT side owns the canonical relation.
  const seen = new Set<string>();
  for (const rid of refIds) {
    const ref = refs[rid];
    if (!ref || seen.has(ref.to_id)) continue;
    const node = buildOne(blocks, refs, nestChildren, ref.to_id);
    if (node) {
      out.push(node);
      seen.add(ref.to_id);
    }
  }
  return out;
}

function buildOne(
  blocks: Record<string, Block>,
  refs: Record<number, BlockRef>,
  nestChildren: Record<string, number[]>,
  id: string,
): PageNode | null {
  const block = blocks[id];
  if (!block || block.type !== BLOCK_TYPE_PAGE) return null;
  return {
    id,
    title: titleOf(block),
    icon: iconOf(block),
    children: childPages(blocks, refs, nestChildren, id),
  };
}

function titleOf(block: Block): string {
  const t = block.data?.title;
  if (typeof t === "string" && t.trim()) return t;
  if (block.text && block.text.trim()) return block.text;
  return "Untitled";
}

function iconOf(block: Block): string | undefined {
  const icon = block.data?.icon;
  return typeof icon === "string" ? icon : undefined;
}

export function countByType(
  blocks: Record<string, Block>,
  workspaceId: string,
): Record<string, number> {
  const out: Record<string, number> = {};
  for (const b of Object.values(blocks)) {
    if (b.workspace_id !== workspaceId) continue;
    out[b.type] = (out[b.type] ?? 0) + 1;
  }
  return out;
}

const TYPE_COLORS = ["#8250DF", "#1A7F37", "#D29922", "#0969DA", "#CF222E", "#7C3AED"];
export function colorForType(slug: string): string {
  let h = 0;
  for (let i = 0; i < slug.length; i++) h = (h * 31 + slug.charCodeAt(i)) >>> 0;
  return TYPE_COLORS[h % TYPE_COLORS.length];
}
