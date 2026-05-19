"use client";

import React, { useRef, useState } from "react";
import { ImageIcon, Upload } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { uploadImage } from "@/lib/api/file";
import { cn, getErrorMessage } from "@/lib/utils";

import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function ImageRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const url = (block.data?.url as string | undefined) ?? "";
  const alt = (block.data?.alt as string | undefined) ?? "";
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
      await dispatch.updateBlockData(block.id, { url: uploaded, alt: file.name }, { text: file.name });
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
        // eslint-disable-next-line @next/next/no-img-element
        <img src={url} alt={alt} className="max-h-[480px] max-w-full rounded border border-border" />
      ) : (
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={uploading}
          className={cn(
            "flex w-full items-center justify-center gap-2 rounded-md border border-dashed border-border bg-muted/30 p-6 text-sm text-muted-foreground hover:bg-muted/50",
            uploading && "opacity-50",
          )}
        >
          {uploading ? <Upload className="h-4 w-4 animate-pulse" /> : <ImageIcon className="h-4 w-4" />}
          <span>{uploading ? "Uploading…" : "Click to upload image"}</span>
        </button>
      )}
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) void onPick(file);
          e.target.value = ""; // allow re-selecting the same file
        }}
      />
      {error && <p className="mt-1 text-xs text-destructive">{error}</p>}
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}
