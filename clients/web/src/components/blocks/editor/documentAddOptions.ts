"use client";

import {
  BLOCK_TYPE_AUDIO,
  BLOCK_TYPE_BOOKMARK,
  BLOCK_TYPE_BULLETED_LIST_ITEM,
  BLOCK_TYPE_CALLOUT,
  BLOCK_TYPE_CHART,
  BLOCK_TYPE_CODE,
  BLOCK_TYPE_COLUMN,
  BLOCK_TYPE_COLUMN_LIST,
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
  BLOCK_TYPE_VIEW,
  type BlockTypeSpec,
  type ViewLayout,
} from "@/lib/api/blockstoreTypes";

import {
  CHART_SUB_TYPES,
  chartHint,
  chartInitialData,
  chartLabel,
} from "../renderers/chart/chartDefaults";
import type { SlashOption } from "./SlashMenu";

interface DispatchAPI {
  insertChild: (
    parentID: string,
    type: string,
    data?: Record<string, unknown>,
    opts?: { text?: string | null },
  ) => Promise<string | undefined>;
}

interface BuildOptionsArgs {
  dispatch: DispatchAPI;
  rootBlockID: string;
  dynamicSpecs: Record<string, BlockTypeSpec>;
}

// buildAddOptions composes the full slash menu for the document's "+ Add
// block" button. Split out of DocumentView so the component itself stays
// focused on lifecycle + layout; this file owns the option catalog.
export function buildAddOptions({ dispatch, rootBlockID, dynamicSpecs }: BuildOptionsArgs): SlashOption[] {
  const addChild =
    (type: string, data: Record<string, unknown> = {}, text: string | null = "") =>
    async () => {
      await dispatch.insertChild(rootBlockID, type, data, { text });
    };

  const makeViewOption = (layout: ViewLayout, label: string, hint: string): SlashOption => ({
    id: `view-${layout}`,
    label,
    hint,
    onSelect: async () => {
      await dispatch.insertChild(rootBlockID, BLOCK_TYPE_VIEW, {
        source_type: BLOCK_TYPE_TASK,
        layout,
        group_by: layout === "kanban" ? "status" : undefined,
        title: `${label} of tasks`,
      });
    },
  });

  return [
    { id: "paragraph", label: "Paragraph", hint: "Plain text", onSelect: addChild(BLOCK_TYPE_PARAGRAPH, { text: "" }) },
    { id: "heading-1", label: "Heading 1", hint: "Large section title", onSelect: addChild(BLOCK_TYPE_HEADING, { level: 1, text: "" }) },
    { id: "heading-2", label: "Heading 2", hint: "Medium section title", onSelect: addChild(BLOCK_TYPE_HEADING, { level: 2, text: "" }) },
    { id: "heading-3", label: "Heading 3", hint: "Small section title", onSelect: addChild(BLOCK_TYPE_HEADING, { level: 3, text: "" }) },
    { id: "bulleted", label: "Bulleted list", hint: "Unordered list item", onSelect: addChild(BLOCK_TYPE_BULLETED_LIST_ITEM, { text: "" }) },
    { id: "numbered", label: "Numbered list", hint: "Ordered list item", onSelect: addChild(BLOCK_TYPE_NUMBERED_LIST_ITEM, { text: "" }) },
    { id: "task", label: "Task", hint: "To-do item", onSelect: addChild(BLOCK_TYPE_TASK, { title: "", status: "todo" }) },
    { id: "toggle", label: "Toggle", hint: "Collapsible group", onSelect: addChild(BLOCK_TYPE_TOGGLE, { summary: "", collapsed: false }) },
    { id: "quote", label: "Quote", hint: "Citation or block quote", onSelect: addChild(BLOCK_TYPE_QUOTE, { text: "" }) },
    { id: "callout", label: "Callout", hint: "Note / warning / tip", onSelect: addChild(BLOCK_TYPE_CALLOUT, { text: "", emoji: "💡" }) },
    { id: "code", label: "Code", hint: "Fenced code block", onSelect: addChild(BLOCK_TYPE_CODE, { code: "", language: "plain" }, null) },
    { id: "document", label: "Document", hint: "Rich-text document (BlockNote)", onSelect: addChild(BLOCK_TYPE_DOCUMENT, { blocknote_ast: [] }, "") },
    { id: "divider", label: "Divider", hint: "Horizontal rule", onSelect: addChild(BLOCK_TYPE_DIVIDER, {}, null) },
    {
      id: "columns-2",
      label: "2 columns",
      hint: "Side-by-side layout",
      onSelect: async () => {
        const listID = await dispatch.insertChild(rootBlockID, BLOCK_TYPE_COLUMN_LIST, {}, { text: null });
        if (listID) {
          await dispatch.insertChild(listID, BLOCK_TYPE_COLUMN, {}, { text: null });
          await dispatch.insertChild(listID, BLOCK_TYPE_COLUMN, {}, { text: null });
        }
      },
    },
    { id: "image", label: "Image", hint: "Upload an image", onSelect: addChild(BLOCK_TYPE_IMAGE, {}, null) },
    { id: "file", label: "File", hint: "Attach a file", onSelect: addChild(BLOCK_TYPE_FILE, {}, null) },
    { id: "video", label: "Video", hint: "Upload or embed", onSelect: addChild(BLOCK_TYPE_VIDEO, {}, null) },
    { id: "audio", label: "Audio", hint: "Upload an audio clip", onSelect: addChild(BLOCK_TYPE_AUDIO, {}, null) },
    { id: "embed", label: "Embed", hint: "YouTube / Figma / Loom", onSelect: addChild(BLOCK_TYPE_EMBED, {}, null) },
    { id: "bookmark", label: "Bookmark", hint: "Web link preview", onSelect: addChild(BLOCK_TYPE_BOOKMARK, {}, null) },
    { id: "equation", label: "Equation", hint: "LaTeX expression", onSelect: addChild(BLOCK_TYPE_EQUATION, { latex: "", display: "block" }, null) },
    { id: "mention", label: "Mention", hint: "@-reference a user", onSelect: addChild(BLOCK_TYPE_MENTION, { user_id: 0, display: "" }, null) },
    { id: "synced", label: "Synced block", hint: "Mirror another block", onSelect: addChild(BLOCK_TYPE_SYNCED_BLOCK, { source_id: "" }, null) },
    { id: "table-block", label: "Table (static)", hint: "Grid of cells", onSelect: addChild(BLOCK_TYPE_TABLE, { rows: [["", ""], ["", ""]], header_row: true }, null) },
    { id: "list", label: "List container", hint: "Group of items", onSelect: addChild(BLOCK_TYPE_LIST, { name: "" }, null) },
    makeViewOption("kanban", "Kanban view", "Group tasks by status"),
    makeViewOption("table", "Table view", "Spreadsheet over tasks"),
    makeViewOption("timeline", "Timeline view", "Gantt by start/end dates"),
    makeViewOption("tree", "Tree view", "Nested outline"),
    makeViewOption("gallery", "Gallery view", "Card grid"),
    ...CHART_SUB_TYPES.map((sub) => ({
      id: `chart-${sub}`,
      label: chartLabel(sub),
      hint: chartHint(sub),
      onSelect: addChild(BLOCK_TYPE_CHART, chartInitialData(sub), null),
    })),
    ...Object.values(dynamicSpecs).map((spec) => ({
      id: `indicator-${spec.type}`,
      label: spec.label ?? spec.type,
      hint: spec.description ?? `New ${spec.type}`,
      onSelect: async () => {
        await dispatch.insertChild(rootBlockID, spec.type, seedIndicatorData(spec), { text: "" });
      },
    })),
  ];
}

// seedIndicatorData generates placeholder values for every required column on
// an indicator type_def so the server's presence check passes. The user then
// fills in real values via RecordEditor.
function seedIndicatorData(spec: BlockTypeSpec): Record<string, unknown> {
  const initial: Record<string, unknown> = {};
  for (const col of spec.columns ?? []) {
    if (col.deprecated || col.computed) continue;
    if (col.default !== undefined) {
      initial[col.key] = col.default;
      continue;
    }
    if (!col.required) continue;
    switch (col.type) {
      case "text":
      case "url":
        initial[col.key] = "";
        break;
      case "number":
        initial[col.key] = 0;
        break;
      case "boolean":
        initial[col.key] = false;
        break;
      case "select":
        initial[col.key] = col.options?.[0]?.value ?? "";
        break;
      case "multi_select":
        initial[col.key] = [];
        break;
      case "date":
        initial[col.key] = new Date().toISOString().slice(0, 10);
        break;
      case "user":
        initial[col.key] = 0;
        break;
      case "block_ref":
        initial[col.key] = "";
        break;
    }
  }
  return initial;
}
