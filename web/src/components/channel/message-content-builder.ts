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
  // Split by mention first; Markdown inline is then tokenised per plain chunk.
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
      elements.push({ type: "text", text: part });
      continue;
    }
    elements.push(...tokeniseMarkdown(part));
  }

  return elements;
}

/**
 * Minimal inline-Markdown tokenizer. Supports: **bold**, *italic*, ~~strike~~,
 * `code`, [text](url). Unmatched fragments fall through as plain text so the
 * Toolbar's wrap buttons always produce something renderable even on partial
 * input.
 */
const MD_INLINE = /\*\*([^*]+?)\*\*|\*([^*]+?)\*|~~([^~]+?)~~|`([^`]+?)`|\[([^\]]+?)\]\((\S+?)\)/g;

export function tokeniseMarkdown(text: string): InlineElement[] {
  const out: InlineElement[] = [];
  let lastIdx = 0;
  MD_INLINE.lastIndex = 0;
  let match: RegExpExecArray | null;
  while ((match = MD_INLINE.exec(text)) !== null) {
    if (match.index > lastIdx) {
      out.push({ type: "text", text: text.slice(lastIdx, match.index) });
    }
    if (match[1] !== undefined) {
      out.push({ type: "text", text: match[1], style: { bold: true } });
    } else if (match[2] !== undefined) {
      out.push({ type: "text", text: match[2], style: { italic: true } });
    } else if (match[3] !== undefined) {
      out.push({ type: "text", text: match[3], style: { strike: true } });
    } else if (match[4] !== undefined) {
      out.push({ type: "text", text: match[4], style: { code: true } });
    } else if (match[5] !== undefined && match[6] !== undefined) {
      out.push({ type: "link", text: match[5], url: match[6] });
    }
    lastIdx = MD_INLINE.lastIndex;
  }
  if (lastIdx < text.length) {
    out.push({ type: "text", text: text.slice(lastIdx) });
  }
  return out.length > 0 ? out : [{ type: "text", text }];
}
