"use client";

import React from "react";
import type { Block, BlockRef } from "@/lib/api/blockstoreTypes";
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
  BLOCK_TYPE_LINK_TO_PAGE,
  BLOCK_TYPE_LIST,
  BLOCK_TYPE_MENTION,
  BLOCK_TYPE_NUMBERED_LIST_ITEM,
  BLOCK_TYPE_PAGE,
  BLOCK_TYPE_PARAGRAPH,
  BLOCK_TYPE_QUOTE,
  BLOCK_TYPE_SYNCED_BLOCK,
  BLOCK_TYPE_TABLE,
  BLOCK_TYPE_TASK,
  BLOCK_TYPE_TOGGLE,
  BLOCK_TYPE_VIDEO,
  BLOCK_TYPE_VIEW,
} from "@/lib/api/blockstoreTypes";
import { useBlock, useNestChildren, useRefs } from "@/stores/blockstore";

import { PageRenderer } from "./renderers/PageRenderer";
import { ParagraphRenderer } from "./renderers/ParagraphRenderer";
import { TaskRenderer } from "./renderers/TaskRenderer";
import { ListRenderer } from "./renderers/ListRenderer";
import { HeadingRenderer } from "./renderers/HeadingRenderer";
import { DividerRenderer } from "./renderers/DividerRenderer";
import { CodeRenderer } from "./renderers/CodeRenderer";
import { QuoteRenderer } from "./renderers/QuoteRenderer";
import { CalloutRenderer } from "./renderers/CalloutRenderer";
import { ListItemRenderer } from "./renderers/ListItemRenderer";
import { ToggleRenderer } from "./renderers/ToggleRenderer";
import { LinkToPageRenderer } from "./renderers/LinkToPageRenderer";
import { ImageRenderer } from "./renderers/ImageRenderer";
import { FileRenderer } from "./renderers/FileRenderer";
import { VideoRenderer } from "./renderers/VideoRenderer";
import { EmbedRenderer } from "./renderers/EmbedRenderer";
import { BookmarkRenderer } from "./renderers/BookmarkRenderer";
import { AudioRenderer } from "./renderers/AudioRenderer";
import { DocumentRenderer } from "./renderers/DocumentRenderer";
import { EquationRenderer } from "./renderers/EquationRenderer";
import { ChartRenderer } from "./renderers/chart/ChartRenderer";
import { SyncedBlockRenderer } from "./renderers/SyncedBlockRenderer";
import { TableBlockRenderer } from "./renderers/TableBlockRenderer";
import { MentionRenderer } from "./renderers/MentionRenderer";
import { ColumnListRenderer, ColumnRenderer } from "./renderers/ColumnListRenderer";
import { SortableNest } from "./editor/SortableNest";
import { RecordEditor } from "./editor/RecordEditor";
import { ViewRenderer } from "./views/ViewRenderer";
import { useBlockTypeSpec } from "@/lib/blockstore/useBlockTypeSpec";

export interface BlockRendererProps {
  blockID: string;
  depth?: number;
}

export function BlockRenderer({ blockID, depth = 0 }: BlockRendererProps) {
  const block = useBlock(blockID);
  if (!block) return null;
  switch (block.type) {
    case BLOCK_TYPE_PAGE:
      return <PageRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_PARAGRAPH:
      return <ParagraphRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_TASK:
      return <TaskRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_LIST:
      return <ListRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_VIEW:
      return <ViewRenderer block={block} />;
    case BLOCK_TYPE_HEADING:
      return <HeadingRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_DIVIDER:
      return <DividerRenderer block={block} />;
    case BLOCK_TYPE_CODE:
      return <CodeRenderer block={block} />;
    case BLOCK_TYPE_QUOTE:
      return <QuoteRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_CALLOUT:
      return <CalloutRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_BULLETED_LIST_ITEM:
    case BLOCK_TYPE_NUMBERED_LIST_ITEM:
      return <ListItemRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_TOGGLE:
      return <ToggleRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_LINK_TO_PAGE:
      return <LinkToPageRenderer block={block} />;
    case BLOCK_TYPE_IMAGE:
      return <ImageRenderer block={block} />;
    case BLOCK_TYPE_FILE:
      return <FileRenderer block={block} />;
    case BLOCK_TYPE_VIDEO:
      return <VideoRenderer block={block} />;
    case BLOCK_TYPE_EMBED:
      return <EmbedRenderer block={block} />;
    case BLOCK_TYPE_BOOKMARK:
      return <BookmarkRenderer block={block} />;
    case BLOCK_TYPE_AUDIO:
      return <AudioRenderer block={block} />;
    case BLOCK_TYPE_DOCUMENT:
      return <DocumentRenderer block={block} />;
    case BLOCK_TYPE_EQUATION:
      return <EquationRenderer block={block} />;
    case BLOCK_TYPE_CHART:
      return <ChartRenderer block={block} />;
    case BLOCK_TYPE_SYNCED_BLOCK:
      return <SyncedBlockRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_TABLE:
      return <TableBlockRenderer block={block} />;
    case BLOCK_TYPE_MENTION:
      return <MentionRenderer block={block} />;
    case BLOCK_TYPE_COLUMN_LIST:
      return <ColumnListRenderer block={block} depth={depth} />;
    case BLOCK_TYPE_COLUMN:
      return <ColumnRenderer block={block} depth={depth} />;
    default:
      return <DynamicRecord block={block} depth={depth} />;
  }
}

function DynamicRecord({ block, depth }: { block: Block; depth: number }) {
  const spec = useBlockTypeSpec(block.workspace_id, block.type);
  if (spec && spec.columns && spec.columns.length > 0) {
    return <RecordEditor block={block} spec={spec} depth={depth} />;
  }
  return <UnknownBlock block={block} />;
}

function UnknownBlock({ block }: { block: Block }) {
  return (
    <div className="rounded border border-dashed border-muted-foreground/40 bg-muted/30 p-2 text-xs text-muted-foreground">
      unknown block type <code className="font-mono">{block.type}</code>
    </div>
  );
}

export function NestChildren({ parentID, depth }: { parentID: string; depth: number }) {
  const refIDs = useNestChildren(parentID);
  const refs = useRefs();
  const parent = useBlock(parentID);
  if (!refIDs || refIDs.length === 0 || !parent) return null;
  return (
    <div className="flex flex-col gap-1">
      <SortableNest
        workspaceID={parent.workspace_id}
        parentID={parentID}
        childRefIDs={refIDs}
        renderItem={(refID) => {
          const ref = refs[refID] as BlockRef | undefined;
          if (!ref) return null;
          return <BlockRenderer blockID={ref.to_id} depth={depth + 1} />;
        }}
      />
    </div>
  );
}
