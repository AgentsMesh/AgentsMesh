"use client";

import { Suspense, lazy } from "react";
import { Button } from "@/components/ui/button";

// Lazy load BlockEditor to avoid SSR issues
const BlockEditor = lazy(() => import("@/components/ui/block-editor"));

interface TicketEditFormProps {
  title: string;
  content: string;
  onTitleChange: (value: string) => void;
  onContentChange: (value: string) => void;
  onSave: () => void;
  onCancel: () => void;
  t: (key: string) => string;
}

/**
 * Inline edit form for ticket title and content
 */
export function TicketEditForm({
  title,
  content,
  onTitleChange,
  onContentChange,
  onSave,
  onCancel,
  t,
}: TicketEditFormProps) {
  return (
    <div className="space-y-4">
      {/* Title Input */}
      <input
        type="text"
        className="w-full text-2xl font-semibold px-3 py-2 border border-border rounded-md"
        value={title}
        onChange={(e) => onTitleChange(e.target.value)}
      />

      {/* Rich Content Editor */}
      <div>
        <label className="text-sm font-medium text-muted-foreground mb-1 block">
          {t("tickets.detail.content")}
        </label>
        <div className="border border-border rounded-md overflow-hidden min-h-[200px] bg-card">
          <Suspense fallback={<div className="h-[200px] animate-pulse bg-muted" />}>
            <BlockEditor
              initialContent={content}
              onChange={onContentChange}
              editable={true}
            />
          </Suspense>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex gap-2">
        <Button size="sm" onClick={onSave}>
          {t("common.save")}
        </Button>
        <Button size="sm" variant="outline" onClick={onCancel}>
          {t("common.cancel")}
        </Button>
      </div>
    </div>
  );
}

export default TicketEditForm;
