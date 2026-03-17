"use client";

import { useCreateBlockNote } from "@blocknote/react";
import { BlockNoteView } from "@blocknote/mantine";
import "@blocknote/mantine/style.css";
import { useEffect, useRef, useMemo, useCallback, useSyncExternalStore } from "react";
import { PartialBlock } from "@blocknote/core";
import { uploadImage } from "@/lib/api/file";

interface BlockEditorProps {
  initialContent?: string; // JSON string
  onChange?: (content: string) => void;
  editable?: boolean;
  placeholder?: string;
  className?: string;
}

// Get current theme from document
function getThemeSnapshot(): "light" | "dark" {
  if (typeof document === "undefined") return "dark";
  return document.documentElement.classList.contains("dark") ? "dark" : "light";
}

// Server snapshot (used during SSR)
function getServerThemeSnapshot(): "light" | "dark" {
  return "dark";
}

// Subscribe to theme changes via MutationObserver
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

// Hook to detect current theme from document using useSyncExternalStore
function useThemeDetect(): "light" | "dark" {
  return useSyncExternalStore(
    subscribeToTheme,
    getThemeSnapshot,
    getServerThemeSnapshot
  );
}

// Upload file to backend using organization-scoped API
async function uploadFile(file: File): Promise<string> {
  return uploadImage(file);
}

// Parse initial content safely
function parseInitialContent(content?: string): PartialBlock[] | undefined {
  if (!content) return undefined;
  try {
    const parsed = JSON.parse(content);
    // Ensure it's an array
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
  void _placeholder; // Reserved for future use
  const theme = useThemeDetect();

  // Parse content once on mount
  const parsedContent = useMemo(
    () => parseInitialContent(initialContent),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [] // Only parse on mount
  );

  const editor = useCreateBlockNote({
    initialContent: parsedContent,
    uploadFile,
  });

  // Track the last content string we emitted (onChange) or received (external sync).
  // Used to distinguish our own saves from external updates (e.g., WebSocket).
  const lastContentRef = useRef<string | undefined>(initialContent);

  // Sync external content changes (e.g., from WebSocket updates) into the editor.
  // When initialContent changes to something we didn't emit, replace the editor blocks.
  // The subsequent onChange from replaceBlocks is harmless — the debounced save will
  // write back the same content, and the next useEffect comparison will match.
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

// Read-only viewer for displaying content
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

  // Update content when it changes
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
