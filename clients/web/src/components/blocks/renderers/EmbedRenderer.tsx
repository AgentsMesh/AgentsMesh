"use client";

import React from "react";
import { ExternalLink } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { isSafeURL, sanitizeURL } from "@/lib/blockstore/urlGuard";

import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

// EmbedRenderer wraps third-party iframes (YouTube / Figma / Loom /
// CodeSandbox / generic). The provider is detected from the URL; whitelisted
// providers get a rich iframe, others fall back to an ExternalLink.
export function EmbedRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const url = (block.data?.url as string | undefined) ?? "";
  const provider = detectEmbedProvider(url);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const onPasteUrl = () => {
    const next = window.prompt("Embed URL (YouTube / Figma / Loom / …):", url);
    if (!next) return;
    if (!isSafeURL(next)) {
      window.alert("Only http(s) URLs are allowed.");
      return;
    }
    dispatch.updateBlockData(
      block.id,
      { url: next, provider: detectEmbedProvider(next) },
      { text: next },
    );
  };

  return (
    <BlockChrome
      className=""
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      {url ? (
        provider.iframe && isSafeURL(provider.iframe) ? (
          <iframe
            src={provider.iframe}
            className="aspect-video w-full rounded-md border border-border"
            allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture; fullscreen"
            sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox allow-presentation"
            referrerPolicy="no-referrer-when-downgrade"
            allowFullScreen
          />
        ) : (
          <a
            href={sanitizeURL(url) || "#"}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-sm hover:bg-muted/50"
          >
            <ExternalLink className="h-4 w-4" />
            <span className="truncate">{url}</span>
          </a>
        )
      ) : (
        <button
          type="button"
          onClick={onPasteUrl}
          className="w-full rounded-md border border-dashed border-border bg-muted/30 p-3 text-sm text-muted-foreground hover:bg-muted/50"
        >
          Paste embed URL (YouTube / Figma / Loom / …)
        </button>
      )}
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}

// detectEmbedProvider maps known hosts to an iframe src. Unknown URLs return
// iframe=null so the renderer falls back to a clickable link card.
function detectEmbedProvider(url: string): { name: string; iframe: string | null } {
  if (!url) return { name: "none", iframe: null };
  if (/youtu\.be|youtube\.com/.test(url)) {
    const m = url.match(/(?:v=|youtu\.be\/)([\w-]{6,})/);
    return { name: "youtube", iframe: m ? `https://www.youtube.com/embed/${m[1]}` : null };
  }
  if (/figma\.com/.test(url)) {
    return {
      name: "figma",
      iframe: `https://www.figma.com/embed?embed_host=agentsmesh&url=${encodeURIComponent(url)}`,
    };
  }
  if (/loom\.com/.test(url)) {
    const m = url.match(/loom\.com\/share\/([\w-]+)/);
    return { name: "loom", iframe: m ? `https://www.loom.com/embed/${m[1]}` : null };
  }
  if (/codesandbox\.io/.test(url)) {
    return { name: "codesandbox", iframe: url.replace("/s/", "/embed/") };
  }
  return { name: "generic", iframe: null };
}
