"use client";

import { useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import type { MentionItem } from "@/hooks/useMentionCandidates";
import { useTranslations } from "next-intl";

interface MentionDropdownProps {
  items: MentionItem[];
  activeIndex: number;
  onSelect: (item: MentionItem) => void;
  position: { top: number; left: number } | null;
  visible: boolean;
}

/**
 * Floating dropdown for @ mention candidates.
 * Rendered as an absolutely positioned popup near the textarea cursor.
 */
export function MentionDropdown({
  items,
  activeIndex,
  onSelect,
  position,
  visible,
}: MentionDropdownProps) {
  const t = useTranslations();
  const listRef = useRef<HTMLDivElement>(null);
  const activeRef = useRef<HTMLButtonElement>(null);

  // Scroll active item into view
  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: "nearest" });
  }, [activeIndex]);

  if (!visible || items.length === 0 || !position) return null;

  // Group items by type
  const userItems = items.filter((i) => i.type === "user");
  const podItems = items.filter((i) => i.type === "pod");

  // Calculate flat index offset for pods
  const podIndexOffset = userItems.length;

  const renderItem = (item: MentionItem, index: number) => {
    const isActive = index === activeIndex;

    return (
      <button
        key={item.id}
        ref={isActive ? activeRef : undefined}
        className={cn(
          "w-full flex items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors",
          isActive
            ? "bg-primary/10 text-primary"
            : "hover:bg-muted text-foreground"
        )}
        onMouseDown={(e) => {
          // Prevent blur on textarea
          e.preventDefault();
          onSelect(item);
        }}
      >
        {/* Avatar / Icon */}
        {item.type === "user" ? (
          item.avatarUrl ? (
            /* eslint-disable-next-line @next/next/no-img-element */
            <img
              src={item.avatarUrl}
              alt={item.displayName}
              className="w-6 h-6 rounded-full flex-shrink-0"
            />
          ) : (
            <div className="w-6 h-6 rounded-full bg-muted flex items-center justify-center flex-shrink-0">
              <span className="text-xs font-medium">
                {item.displayName[0]?.toUpperCase()}
              </span>
            </div>
          )
        ) : (
          <div className="w-6 h-6 rounded-full bg-primary flex items-center justify-center flex-shrink-0">
            <svg
              className="w-3.5 h-3.5 text-primary-foreground"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
              />
            </svg>
          </div>
        )}

        {/* Name & description */}
        <div className="min-w-0 flex-1">
          <div className="font-medium text-sm truncate">
            {item.displayName}
          </div>
          {item.description && (
            <div className="text-xs text-muted-foreground truncate">
              {item.description}
            </div>
          )}
        </div>
      </button>
    );
  };

  return (
    <div
      ref={listRef}
      className="absolute z-50 w-64 max-h-48 overflow-y-auto rounded-lg border bg-popover shadow-lg"
      style={{
        bottom: position.top,
        left: position.left,
      }}
    >
      {/* Users section */}
      {userItems.length > 0 && (
        <>
          <div className="px-3 py-1 text-xs font-medium text-muted-foreground bg-muted/30 sticky top-0">
            {t("mesh.mention.members")}
          </div>
          {userItems.map((item, i) => renderItem(item, i))}
        </>
      )}

      {/* Pods section */}
      {podItems.length > 0 && (
        <>
          <div className="px-3 py-1 text-xs font-medium text-muted-foreground bg-muted/30 sticky top-0">
            {t("mesh.mention.pods")}
          </div>
          {podItems.map((item, i) =>
            renderItem(item, podIndexOffset + i)
          )}
        </>
      )}
    </div>
  );
}

export default MentionDropdown;
