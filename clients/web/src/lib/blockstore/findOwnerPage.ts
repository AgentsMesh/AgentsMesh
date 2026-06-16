import { BLOCK_TYPE_PAGE, type Block, type BlockRef } from "@/lib/viewModels/blockstore";

// Walk the nest tree upward from blockID to the nearest page ancestor — the page
// that actually renders blockID in its DOM (sub-pages render as links, so a block
// only lives in the DOM of its closest page ancestor). A page target owns itself.
export function findOwnerPage(
  blockID: string,
  blocks: Record<string, Block>,
  refs: Record<number, BlockRef>,
  nestChildren: Record<string, number[]>,
): string | null {
  const parentOf = buildParentIndex(refs, nestChildren);
  const seen = new Set<string>();
  let current: string | undefined = blockID;
  while (current && !seen.has(current)) {
    seen.add(current);
    if (blocks[current]?.type === BLOCK_TYPE_PAGE) return current;
    current = parentOf[current];
  }
  return null;
}

function buildParentIndex(
  refs: Record<number, BlockRef>,
  nestChildren: Record<string, number[]>,
): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [parentID, refIDs] of Object.entries(nestChildren)) {
    for (const rid of refIDs) {
      const childID = refs[rid]?.to_id;
      if (childID && !(childID in out)) out[childID] = parentID;
    }
  }
  return out;
}
