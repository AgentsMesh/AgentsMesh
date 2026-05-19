"use client";

import { useCreateBlockNote } from "@blocknote/react";
import { BlockNoteView } from "@blocknote/mantine";
import "@blocknote/mantine/style.css";
import { useEffect, useRef, useMemo, useCallback, useSyncExternalStore } from "react";
import { PartialBlock } from "@blocknote/core";
import { getFileService } from "@/lib/wasm-core";

async function uploadImageViaWasm(file: File): Promise<string> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  return getFileService().upload_file(bytes, file.name, file.type || "application/octet-stream");
}

interface BlockEditorProps {
  initialContent?: string;
  onChange?: (content: string) => void;
  editable?: boolean;
  placeholder?: string;
  className?: string;
}

function getThemeSnapshot(): "light" | "dark" {
  if (typeof document === "undefined") return "dark";
  return document.documentElement.classList.contains("dark") ? "dark" : "light";
}

function getServerThemeSnapshot(): "light" | "dark" {
  return "dark";
}

function subscribeToTheme(callback: () => void): () => void {
  if (typeof document === "undefined") return () => {};

  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      if (mutation.attributeName === "class") {
        callback();
        break;
      }
    }
  });

  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["class"],
  });

  return () => observer.disconnect();
}

function useThemeDetect(): "light" | "dark" {
  return useSyncExternalStore(
    subscribeToTheme,
    getThemeSnapshot,
    getServerThemeSnapshot
  );
}

async function uploadFile(file: File): Promise<string> {
  return uploadImageViaWasm(file);
}

function parseInitialContent(content?: string): PartialBlock[] | undefined {
  if (!content) return undefined;
  try {
    const parsed = JSON.parse(content);
    if (Array.isArray(parsed) && parsed.length > 0) {
      return parsed;
    }
    return undefined;
  } catch {
    return undefined;
  }
}

export function BlockEditor({
  initialContent,
  onChange,
  editable = true,
  placeholder: _placeholder,
  className,
}: BlockEditorProps) {
  void _placeholder;
  const theme = useThemeDetect();

  const parsedContent = useMemo(
    () => parseInitialContent(initialContent),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  );

  const editor = useCreateBlockNote({
    initialContent: parsedContent,
    uploadFile,
  });

  const lastContentRef = useRef<string | undefined>(initialContent);

  useEffect(() => {
    if (!initialContent || initialContent === lastContentRef.current) return;
    const newBlocks = parseInitialContent(initialContent);
    if (newBlocks) {
      editor.replaceBlocks(editor.document, newBlocks);
      lastContentRef.current = initialContent;
    }
  }, [initialContent, editor]);

  const handleChange = useCallback(() => {
    if (onChange) {
      const json = JSON.stringify(editor.document);
      lastContentRef.current = json;
      onChange(json);
    }
  }, [onChange, editor]);

  return (
    <div className={className}>
      <BlockNoteView
        editor={editor}
        editable={editable}
        theme={theme}
        onChange={handleChange}
      />
    </div>
  );
}

export function BlockViewer({
  content,
  className,
}: {
  content?: string;
  className?: string;
}) {
  const theme = useThemeDetect();
  const parsedContent = useMemo(() => parseInitialContent(content), [content]);

  const editor = useCreateBlockNote({
    initialContent: parsedContent,
  });

  useEffect(() => {
    if (content) {
      const newContent = parseInitialContent(content);
      if (newContent) {
        editor.replaceBlocks(editor.document, newContent);
      }
    }
  }, [content, editor]);

  return (
    <div className={className}>
      <BlockNoteView
        editor={editor}
        editable={false}
        theme={theme}
      />
    </div>
  );
}

export default BlockEditor;
