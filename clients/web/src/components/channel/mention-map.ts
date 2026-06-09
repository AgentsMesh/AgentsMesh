import type { Block, InlineElement, MentionRefInput, MessageContent } from "@/lib/viewModels/channelMessage";

export function extractMentionMap(content?: MessageContent): Record<string, MentionRefInput> {
  const out: Record<string, MentionRefInput> = {};
  if (!content?.blocks) return out;
  collect(content.blocks, out);
  return out;
}

function collect(blocks: Block[], out: Record<string, MentionRefInput>) {
  for (const block of blocks) {
    collectFromElements(block.elements, out);
    for (const row of block.rows ?? []) {
      for (const cell of row.cells ?? []) {
        collectFromElements(cell.elements, out);
      }
    }
    for (const item of block.items ?? []) {
      collect(item, out);
    }
    if (block.children?.length) {
      collect(block.children, out);
    }
  }
}

function collectFromElements(elements: InlineElement[] | undefined, out: Record<string, MentionRefInput>) {
  for (const el of elements ?? []) {
    if (el.type === "mention" && el.display && el.entity_key) {
      if (el.entity_type === "pod" || el.entity_type === "user") {
        out[el.display] = { entity_type: el.entity_type, entity_key: el.entity_key };
      }
    }
  }
}
