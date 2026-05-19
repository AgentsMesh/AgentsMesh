"use client";

import { useState, useRef, useEffect, useCallback, KeyboardEvent } from "react";
import { cn } from "@/lib/utils";
import { Check, X, Loader2 } from "lucide-react";

export interface InlineEditableTextProps {
  value: string;
  onSave: (value: string) => Promise<void>;
  placeholder?: string;
  className?: string;
  inputClassName?: string;
  multiline?: boolean;
  disabled?: boolean;
  debounceMs?: number;
  autoSave?: boolean;
}

export function InlineEditableText({
  value,
  onSave,
  placeholder = "Click to edit...",
  className,
  inputClassName,
  multiline = false,
  disabled = false,
  debounceMs = 500,
  autoSave = false,
}: InlineEditableTextProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(value);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null);
  const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (!isEditing) {
      setEditValue(value);
    }
  }, [value, isEditing]);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
    };
  }, []);

  const handleSave = useCallback(async () => {
    if (editValue === value) {
      setIsEditing(false);
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      await onSave(editValue);
      setIsEditing(false);
    } catch (err) {
      console.error("Failed to save:", err);
      setError(err instanceof Error ? err.message : "Failed to save");
    } finally {
      setIsSaving(false);
    }
  }, [editValue, value, onSave]);

  const handleCancel = useCallback(() => {
    setEditValue(value);
    setIsEditing(false);
    setError(null);
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current);
    }
  }, [value]);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      handleCancel();
    } else if (e.key === "Enter" && !multiline) {
      e.preventDefault();
      handleSave();
    } else if (e.key === "Enter" && multiline && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSave();
    }
  }, [handleCancel, handleSave, multiline]);

  const handleChange = useCallback((newValue: string) => {
    setEditValue(newValue);
    setError(null);

    if (autoSave) {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
      saveTimeoutRef.current = setTimeout(() => {
        if (newValue !== value) {
          handleSave();
        }
      }, debounceMs);
    }
  }, [autoSave, debounceMs, value, handleSave]);

  const handleBlur = useCallback(() => {
    if (!autoSave) {
      handleSave();
    }
  }, [autoSave, handleSave]);

  if (disabled) {
    return (
      <span className={cn("text-muted-foreground", className)}>
        {value || placeholder}
      </span>
    );
  }

  if (!isEditing) {
    return (
      <button
        type="button"
        className={cn(
          "text-left w-full rounded-sm px-1 -mx-1 hover:bg-muted/50 transition-colors cursor-text",
          !value && "text-muted-foreground italic",
          className
        )}
        onClick={() => setIsEditing(true)}
      >
        {value || placeholder}
      </button>
    );
  }

  const InputComponent = multiline ? "textarea" : "input";

  return (
    <div className="relative">
      <InputComponent
        ref={inputRef as React.RefObject<HTMLInputElement & HTMLTextAreaElement>}
        type="text"
        value={editValue}
        onChange={(e) => handleChange(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={handleBlur}
        disabled={isSaving}
        className={cn(
          "w-full px-2 py-1 -mx-1 rounded-md border border-primary bg-background",
          "focus:outline-none focus:ring-2 focus:ring-primary/20",
          "disabled:opacity-50",
          error && "border-destructive focus:ring-destructive/20",
          multiline && "min-h-[80px] resize-y",
          inputClassName
        )}
        placeholder={placeholder}
        rows={multiline ? 3 : undefined}
      />
      <div className="absolute right-1 top-1/2 -translate-y-1/2 flex items-center gap-1">
        {isSaving && (
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        )}
        {!isSaving && isEditing && (
          <>
            <button
              type="button"
              onClick={handleSave}
              className="p-1 rounded hover:bg-muted text-green-600 dark:text-green-400"
              title="Save (Enter)"
            >
              <Check className="h-3 w-3" />
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="p-1 rounded hover:bg-muted text-muted-foreground"
              title="Cancel (Escape)"
            >
              <X className="h-3 w-3" />
            </button>
          </>
        )}
      </div>
      {error && (
        <p className="text-xs text-destructive mt-1">{error}</p>
      )}
    </div>
  );
}

export default InlineEditableText;
