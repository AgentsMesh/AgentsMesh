import type { Block, MentionRefInput, MessageContent } from "@/lib/api/channel-message-types";

export function extractMentionMap(content?: MessageContent): Record<string, MentionRefInput> {
  const out: Record<string, MentionRefInput> = {};
  if (!content?.blocks) return out;
  collect(content.blocks, out);
  return out;
}

function collect(blocks: Block[], out: Record<string, MentionRefInput>) {
  for (const block of blocks) {
    for (const el of block.elements ?? []) {
      if (el.type === "mention" && el.display && el.entity_key) {
        if (el.entity_type === "pod" || el.entity_type === "user") {
          out[el.display] = { entity_type: el.entity_type, entity_key: el.entity_key };
        }
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
