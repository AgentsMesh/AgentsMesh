"use client";

import React, { useRef, useState } from "react";
import { Film, Upload } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { uploadImage } from "@/lib/api/file";
import { cn, getErrorMessage } from "@/lib/utils";

import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function VideoRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const url = (block.data?.url as string | undefined) ?? "";
  const provider = detectProvider(url);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const onPick = async (file: File) => {
    setError(null);
    setUploading(true);
    try {
      const uploaded = await uploadImage(file);
      await dispatch.updateBlockData(
        block.id,
        { url: uploaded, provider: "native" },
        { text: file.name },
      );
    } catch (e) {
      setError(getErrorMessage(e, "Upload failed"));
    } finally {
      setUploading(false);
    }
  };

  const onPasteUrl = () => {
    const next = window.prompt("Video URL (YouTube / Vimeo / direct link):", url);
    if (next) dispatch.updateBlockData(block.id, { url: next, provider: detectProvider(next) });
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
        <div className="overflow-hidden rounded-md border border-border bg-black">
          {provider === "youtube" || provider === "vimeo" ? (
            <iframe
              src={embedURL(url, provider)}
              className="aspect-video w-full"
              allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture"
              sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox allow-presentation"
              referrerPolicy="no-referrer-when-downgrade"
              allowFullScreen
            />
          ) : (
            <video controls src={url} className="w-full" />
          )}
        </div>
      ) : (
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => inputRef.current?.click()}
            disabled={uploading}
            className={cn(
              "flex flex-1 items-center justify-center gap-2 rounded-md border border-dashed border-border bg-muted/30 p-3 text-sm text-muted-foreground hover:bg-muted/50",
              uploading && "opacity-50",
            )}
          >
            {uploading ? <Upload className="h-4 w-4 animate-pulse" /> : <Film className="h-4 w-4" />}
            <span>{uploading ? "Uploading…" : "Upload video"}</span>
          </button>
          <button
            type="button"
            onClick={onPasteUrl}
            className="rounded-md border border-border px-3 text-sm text-muted-foreground hover:bg-muted/50"
          >
            Paste URL
          </button>
        </div>
      )}
      <input
        ref={inputRef}
        type="file"
        accept="video/*"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) void onPick(file);
          e.target.value = "";
        }}
      />
      {error && <p className="mt-1 text-xs text-destructive">{error}</p>}
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}

function detectProvider(url: string): "native" | "youtube" | "vimeo" {
  if (/youtu\.be|youtube\.com/.test(url)) return "youtube";
  if (/vimeo\.com/.test(url)) return "vimeo";
  return "native";
}

function embedURL(url: string, provider: "youtube" | "vimeo"): string {
  if (provider === "youtube") {
    const m = url.match(/(?:v=|youtu\.be\/)([\w-]{6,})/);
    return m ? `https://www.youtube.com/embed/${m[1]}` : url;
  }
  if (provider === "vimeo") {
    const m = url.match(/vimeo\.com\/(\d+)/);
    return m ? `https://player.vimeo.com/video/${m[1]}` : url;
  }
  return url;
}
