"use client";

import React, { useRef, useState } from "react";
import { Music, Upload } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { uploadImage } from "@/lib/api/file";
import { cn, getErrorMessage } from "@/lib/utils";

import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function AudioRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const url = (block.data?.url as string | undefined) ?? "";
  const title = (block.data?.title as string | undefined) ?? "";
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
        { url: uploaded, title: file.name },
        { text: file.name },
      );
    } catch (e) {
      setError(getErrorMessage(e, "Upload failed"));
    } finally {
      setUploading(false);
    }
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
        <div className="flex flex-col gap-1 rounded-md border border-border bg-muted/20 p-2">
          {title && <span className="text-xs text-muted-foreground">{title}</span>}
          <audio controls src={url} className="w-full" />
        </div>
      ) : (
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={uploading}
          className={cn(
            "flex w-full items-center gap-2 rounded-md border border-dashed border-border bg-muted/30 p-3 text-sm text-muted-foreground hover:bg-muted/50",
            uploading && "opacity-50",
          )}
        >
          {uploading ? <Upload className="h-4 w-4 animate-pulse" /> : <Music className="h-4 w-4" />}
          <span>{uploading ? "Uploading…" : "Upload audio"}</span>
        </button>
      )}
      <input
        ref={inputRef}
        type="file"
        accept="audio/*"
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
