"use client";

import { useCreateBlockNote } from "@blocknote/react";
import { BlockNoteView } from "@blocknote/mantine";
import "@blocknote/mantine/style.css";
import { useEffect, useMemo, useCallback, useSyncExternalStore } from "react";
import { PartialBlock } from "@blocknote/core";
import { useAuthStore } from "@/stores/auth";
import { getApiBaseUrl } from "@/lib/env";

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
  const { token, currentOrg } = useAuthStore.getState();

  if (!currentOrg) {
    throw new Error("No organization selected");
  }

  const formData = new FormData();
  formData.append("file", file);

  const API_BASE_URL = getApiBaseUrl();
  const res = await fetch(`${API_BASE_URL}/api/v1/orgs/${currentOrg.slug}/files/upload`, {
    method: "POST",
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      "X-Organization-Slug": currentOrg.slug,
    },
    body: formData,
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: "Upload failed" }));
    throw new Error(errorData.error || "Upload failed");
  }

  const data = await res.json();
  return data.url;
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

  // Handle onChange with debounce to avoid excessive updates
  const handleChange = useCallback(() => {
    if (onChange) {
      onChange(JSON.stringify(editor.document));
    }
  }, [onChange, editor]);

  return (
    <div className={className}>
      {/* @blocknote/react@0.47 and @blocknote/mantine@0.46 have mismatched core types */}
      <BlockNoteView
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        editor={editor as any}
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
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        editor={editor as any}
        editable={false}
        theme={theme}
      />
    </div>
  );
}

export default BlockEditor;
