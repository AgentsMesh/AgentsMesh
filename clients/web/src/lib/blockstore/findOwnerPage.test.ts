import { describe, expect, it } from "vitest";

import { BLOCK_TYPE_PAGE, type Block, type BlockRef } from "@/lib/viewModels/blockstore";

import { findOwnerPage } from "./findOwnerPage";

function blk(id: string, type: string): Block {
  return { id, type } as unknown as Block;
}
function ref(id: number, from: string, to: string): BlockRef {
  return { id, from_id: from, to_id: to, rel: "nest" } as unknown as BlockRef;
}

const blocks = {
  root: blk("root", BLOCK_TYPE_PAGE),
  sub: blk("sub", BLOCK_TYPE_PAGE),
  para: blk("para", "paragraph"),
  toggle: blk("toggle", "toggle"),
  deep: blk("deep", "paragraph"),
};
const refs = {
  1: ref(1, "root", "sub"),
  2: ref(2, "root", "toggle"),
  3: ref(3, "toggle", "para"),
  4: ref(4, "sub", "deep"),
};
const nest = { root: [1, 2], toggle: [3], sub: [4] };

describe("findOwnerPage", () => {
  it("returns the page itself when the target is a page", () => {
    expect(findOwnerPage("sub", blocks, refs, nest)).toBe("sub");
    expect(findOwnerPage("root", blocks, refs, nest)).toBe("root");
  });

  it("returns the nearest page ancestor for a nested non-page block", () => {
    expect(findOwnerPage("para", blocks, refs, nest)).toBe("root");
  });

  it("stops at the nearest sub-page, not the root", () => {
    expect(findOwnerPage("deep", blocks, refs, nest)).toBe("sub");
  });

  it("returns null for an orphan non-page block", () => {
    expect(findOwnerPage("ghost", blocks, refs, nest)).toBeNull();
  });
});
