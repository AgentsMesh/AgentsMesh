import type { MessageContent, InlineElement, Block } from "@/lib/api/channel-message-types";

interface MentionRef {
  entityType: string;
  entityKey: string;
}

export function buildMessageContent(
  text: string,
  mentionByText: Map<string, MentionRef>
): MessageContent {
  const lines = text.split(/\n/);
  return {
    schema_version: 1,
    kind: "text",
    blocks: lines.map((line) => ({
      type: "paragraph" as const,
      elements: parseInlineElements(line, mentionByText),
    })),
  };
}

export function extractMentionMap(content?: MessageContent): Map<string, MentionRef> {
  const mentions = new Map<string, MentionRef>();
  if (!content?.blocks) return mentions;
  function processBlocks(blocks: Block[]) {
    for (const block of blocks) {
      collectMentions(block.elements, mentions);
      for (const item of block.items ?? []) {
        collectMentions(item, mentions);
      }
      if (block.children?.length) {
        processBlocks(block.children);
      }
    }
  }
  processBlocks(content.blocks);
  return mentions;
}

function collectMentions(elements: InlineElement[] | undefined, mentions: Map<string, MentionRef>) {
  for (const el of elements ?? []) {
    if (el.type === "mention" && el.display && el.entity_key) {
      mentions.set(el.display, {
        entityType: el.entity_type ?? "pod",
        entityKey: el.entity_key,
      });
    }
  }
}

function parseInlineElements(
  line: string,
  mentionByText: Map<string, MentionRef>
): InlineElement[] {
  const mentionRegex = /(@[\w.\-]+)/g;
  const parts = line.split(mentionRegex);
  const elements: InlineElement[] = [];

  for (const part of parts) {
    if (!part) continue;
    if (part.startsWith("@")) {
      const token = part.slice(1);
      const ref = mentionByText.get(token);
      if (ref) {
        elements.push({
          type: "mention",
          entity_type: ref.entityType as "pod" | "user",
          entity_key: ref.entityKey,
          display: token,
        });
        continue;
      }
    }
    elements.push({ type: "text", text: part });
  }

  return elements;
}
