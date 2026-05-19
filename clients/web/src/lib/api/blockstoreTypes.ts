// Block Store Phase 1 — shared types.
// Mirrors the Go domain structs in backend/internal/domain/blockstore.

export type JSONMap = Record<string, unknown>;

export type ActorType = "user" | "agent" | "system";

export type OpKind =
  | "createBlock"
  | "updateBlock"
  | "deleteBlock"
  | "addRef"
  | "removeRef"
  | "updateRef";

export const REL_NEST = "nest";
export const REL_MENTION = "mention";
export const REL_EMBED = "embed";
export const REL_DEPENDS_ON = "depends_on";
export const REL_DERIVED_FROM = "derived_from";
export const REL_TAG = "tag";
export const REL_COMMENTS_ON = "comments_on";

export const BLOCK_TYPE_PAGE = "page";
export const BLOCK_TYPE_PARAGRAPH = "paragraph";
export const BLOCK_TYPE_TASK = "task";
export const BLOCK_TYPE_LIST = "list";
export const BLOCK_TYPE_VIEW = "view";
export const BLOCK_TYPE_COMMENT = "comment";
export const BLOCK_TYPE_TYPEDEF = "block_type_def";
export const BLOCK_TYPE_HEADING = "heading";
export const BLOCK_TYPE_DIVIDER = "divider";
export const BLOCK_TYPE_CODE = "code";
export const BLOCK_TYPE_QUOTE = "quote";
export const BLOCK_TYPE_CALLOUT = "callout";
export const BLOCK_TYPE_BULLETED_LIST_ITEM = "bulleted_list_item";
export const BLOCK_TYPE_NUMBERED_LIST_ITEM = "numbered_list_item";
export const BLOCK_TYPE_TOGGLE = "toggle";
export const BLOCK_TYPE_LINK_TO_PAGE = "link_to_page";
export const BLOCK_TYPE_IMAGE = "image";
export const BLOCK_TYPE_FILE = "file";
export const BLOCK_TYPE_VIDEO = "video";
export const BLOCK_TYPE_EMBED = "embed";
export const BLOCK_TYPE_BOOKMARK = "bookmark";
export const BLOCK_TYPE_AUDIO = "audio";
export const BLOCK_TYPE_EQUATION = "equation";
export const BLOCK_TYPE_CHART = "chart";
export const BLOCK_TYPE_SYNCED_BLOCK = "synced_block";
export const BLOCK_TYPE_TABLE = "table";
export const BLOCK_TYPE_MENTION = "mention";
export const BLOCK_TYPE_DOCUMENT = "document";
export const BLOCK_TYPE_COLUMN_LIST = "column_list";
export const BLOCK_TYPE_COLUMN = "column";

export type ViewLayout = "list" | "kanban" | "table" | "timeline" | "tree" | "gallery";

export type ChartSubType = "bar" | "line" | "pie" | "area" | "scatter" | "radar";

export interface ViewFilter {
  key: string;
  op: "eq" | "ne" | "contains";
  value: unknown;
}

export interface ViewSort {
  key: string;
  direction: "asc" | "desc";
}

export interface ViewColumn {
  key: string;
  label?: string;
  width?: number;
}

export type AggregateOp = "count" | "count_distinct" | "sum" | "avg" | "min" | "max";

export interface SummaryColumn {
  key: string;           // source data column (ignored for "count")
  aggregate: AggregateOp;
  label?: string;
  format?: "int" | "percent" | "date" | "number";
}

export interface ViewSpec {
  source_type: string;
  source_types?: string[];
  layout: ViewLayout;
  filters?: ViewFilter[];
  sort?: ViewSort[];
  group_by?: string;
  columns?: ViewColumn[];
  summary_columns?: SummaryColumn[];
  title?: string;
}

export interface Block {
  id: string;
  workspace_id: string;
  type: string;
  data: JSONMap;
  text?: string | null;
  meta: JSONMap;
  created_by: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

export interface BlockRef {
  id: number;
  workspace_id: string;
  from_id: string;
  to_id: string;
  rel: string;
  order_key?: string | null;
  anchor?: string | null;
  meta: JSONMap;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface BlockOp {
  id: number;
  workspace_id: string;
  idempotency_key?: string | null;
  actor_type: ActorType;
  actor_id: number;
  op: OpKind;
  target_block?: string | null;
  target_ref?: number | null;
  payload: JSONMap;
  forward: JSONMap;
  inverse: JSONMap;
  parent_op_id?: number | null;
  applied_at: string;
}

export interface Workspace {
  id: string;
  organization_id: number;
  slug: string;
  name: string;
  root_block_id?: string | null;
  created_at: string;
}

export interface OpEnvelope {
  op: OpKind;
  payload: JSONMap;
}

export interface ApplyOpsRequest {
  workspace_id: string;
  ops: OpEnvelope[];
  idempotency_key?: string;
  parent_op_id?: number | null;
}

export interface ApplyOpsResult {
  op_ids: number[];
  was_replay: boolean;
  parent_op_id?: number | null;
}

export interface ChildrenResult {
  blocks: Block[];
  refs: BlockRef[];
}

export interface SearchHit {
  block_id: string;
  type: string;
  snippet: string;
  score: number;
}

// Tier 1 — schema-driven indicator types. Mirrors the server's ColumnSpec /
// BlockTypeSpec. When a block_type_def is written with `columns`, the
// generic RecordEditor renders instances of that type automatically.

export type ColumnType =
  | "text"
  | "number"
  | "boolean"
  | "select"
  | "multi_select"
  | "date"
  | "url"
  | "user"
  | "block_ref";

export interface SelectOption {
  value: string;
  label?: string;
  color?: string;
}

export interface ColumnSpec {
  key: string;
  label?: string;
  type: ColumnType;
  required?: boolean;
  default?: unknown;
  options?: SelectOption[];
  description?: string;
  /** When non-empty, this column is read-only and its value is derived at
   *  render time from the record's other fields via a safe arithmetic
   *  expression (see lib/blockstore/computeColumn). */
  computed?: string;
  deprecated?: boolean;
}

export interface BlockTypeSpec {
  type: string;
  revision?: number;
  label?: string;
  description?: string;
  default_view?: string;
  supported_views?: string[];
  allowed_children?: string[];
  columns?: ColumnSpec[];
}
