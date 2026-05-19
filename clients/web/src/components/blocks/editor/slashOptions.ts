"use client";

import {
  BLOCK_TYPE_AUDIO,
  BLOCK_TYPE_BOOKMARK,
  BLOCK_TYPE_BULLETED_LIST_ITEM,
  BLOCK_TYPE_CALLOUT,
  BLOCK_TYPE_CODE,
  BLOCK_TYPE_DIVIDER,
  BLOCK_TYPE_DOCUMENT,
  BLOCK_TYPE_EMBED,
  BLOCK_TYPE_EQUATION,
  BLOCK_TYPE_FILE,
  BLOCK_TYPE_HEADING,
  BLOCK_TYPE_IMAGE,
  BLOCK_TYPE_LIST,
  BLOCK_TYPE_MENTION,
  BLOCK_TYPE_NUMBERED_LIST_ITEM,
  BLOCK_TYPE_PARAGRAPH,
  BLOCK_TYPE_QUOTE,
  BLOCK_TYPE_SYNCED_BLOCK,
  BLOCK_TYPE_TABLE,
  BLOCK_TYPE_TASK,
  BLOCK_TYPE_TOGGLE,
  BLOCK_TYPE_VIDEO,
} from "@/lib/api/blockstoreTypes";

import type { SlashOption } from "./SlashMenu";

export interface SlashDispatcher {
  insertSiblingAfter(
    siblingID: string,
    type: string,
    initialData?: Record<string, unknown>,
    opts?: { text?: string | null },
  ): Promise<string | null>;
}

export function standardSlashOptions(
  dispatch: SlashDispatcher,
  blockID: string,
): SlashOption[] {
  const after = (type: string, data: Record<string, unknown> = {}, text: string | null = "") =>
    async () => {
      await dispatch.insertSiblingAfter(blockID, type, data, { text });
    };
  return [
    { id: "paragraph", label: "Paragraph", hint: "Plain text", onSelect: after(BLOCK_TYPE_PARAGRAPH, { text: "" }) },
    { id: "heading-1", label: "Heading 1", hint: "Large section title", onSelect: after(BLOCK_TYPE_HEADING, { level: 1, text: "" }) },
    { id: "heading-2", label: "Heading 2", hint: "Medium section title", onSelect: after(BLOCK_TYPE_HEADING, { level: 2, text: "" }) },
    { id: "heading-3", label: "Heading 3", hint: "Small section title", onSelect: after(BLOCK_TYPE_HEADING, { level: 3, text: "" }) },
    { id: "bulleted", label: "Bulleted list", hint: "Unordered list item", onSelect: after(BLOCK_TYPE_BULLETED_LIST_ITEM, { text: "" }) },
    { id: "numbered", label: "Numbered list", hint: "Ordered list item", onSelect: after(BLOCK_TYPE_NUMBERED_LIST_ITEM, { text: "" }) },
    { id: "task", label: "Task", hint: "To-do item", onSelect: after(BLOCK_TYPE_TASK, { title: "", status: "todo" }) },
    { id: "toggle", label: "Toggle", hint: "Collapsible group", onSelect: after(BLOCK_TYPE_TOGGLE, { summary: "", collapsed: false }) },
    { id: "quote", label: "Quote", hint: "Citation or block quote", onSelect: after(BLOCK_TYPE_QUOTE, { text: "" }) },
    { id: "callout", label: "Callout", hint: "Note / warning / tip", onSelect: after(BLOCK_TYPE_CALLOUT, { text: "", emoji: "💡" }) },
    { id: "code", label: "Code block", hint: "Fenced code with language", onSelect: after(BLOCK_TYPE_CODE, { code: "", language: "plain" }, null) },
    { id: "document", label: "Document", hint: "Rich-text document (BlockNote)", onSelect: after(BLOCK_TYPE_DOCUMENT, { blocknote_ast: [] }, "") },
    { id: "divider", label: "Divider", hint: "Horizontal rule", onSelect: after(BLOCK_TYPE_DIVIDER, {}, null) },
    { id: "image", label: "Image", hint: "Upload or link", onSelect: after(BLOCK_TYPE_IMAGE, {}, null) },
    { id: "file", label: "File", hint: "Attach a file", onSelect: after(BLOCK_TYPE_FILE, {}, null) },
    { id: "video", label: "Video", hint: "Upload or embed", onSelect: after(BLOCK_TYPE_VIDEO, {}, null) },
    { id: "audio", label: "Audio", hint: "Upload an audio clip", onSelect: after(BLOCK_TYPE_AUDIO, {}, null) },
    { id: "embed", label: "Embed", hint: "YouTube / Figma / Loom", onSelect: after(BLOCK_TYPE_EMBED, {}, null) },
    { id: "bookmark", label: "Bookmark", hint: "Web link preview", onSelect: after(BLOCK_TYPE_BOOKMARK, {}, null) },
    { id: "equation", label: "Equation", hint: "LaTeX expression", onSelect: after(BLOCK_TYPE_EQUATION, { latex: "", display: "block" }, null) },
    { id: "mention", label: "Mention", hint: "@-reference a user", onSelect: after(BLOCK_TYPE_MENTION, { user_id: 0, display: "" }, null) },
    { id: "synced", label: "Synced block", hint: "Mirror another block", onSelect: after(BLOCK_TYPE_SYNCED_BLOCK, { source_id: "" }, null) },
    { id: "table-block", label: "Table (static)", hint: "Grid of cells", onSelect: after(BLOCK_TYPE_TABLE, { rows: [["", ""], ["", ""]], header_row: true }, null) },
    { id: "list", label: "List container", hint: "Group of items", onSelect: after(BLOCK_TYPE_LIST, { name: "" }, null) },
  ];
}
