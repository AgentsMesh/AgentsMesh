"use client";

import { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import type { OrganizationMember } from "@/lib/api";
import { getOrgApiService } from "@/lib/wasm-getters";

interface MentionPopoverProps {
  visible: boolean;
  query: string;
  position: { top: number; left: number };
  onSelect: (username: string) => void;
  onClose: () => void;
}

export function MentionPopover({
  visible,
  query,
  position,
  onSelect,
  onClose,
}: MentionPopoverProps) {
  const currentOrg = useCurrentOrg();
  const [members, setMembers] = useState<OrganizationMember[]>([]);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [prevQuery, setPrevQuery] = useState(query);
  const popoverRef = useRef<HTMLDivElement>(null);

  if (prevQuery !== query) {
    setPrevQuery(query);
    setSelectedIndex(0);
  }

  useEffect(() => {
    if (!currentOrg?.slug) return;
    getOrgApiService()
      .list_members(currentOrg.slug)
      .then((raw: string) => setMembers(JSON.parse(raw).members || []))
      .catch(() => setMembers([]));
  }, [currentOrg?.slug]);

  const filtered = useMemo(() => {
    return members.filter((m) => {
      if (!query) return true;
      const q = query.toLowerCase();
      const username = m.user?.username?.toLowerCase() || "";
      const name = m.user?.name?.toLowerCase() || "";
      return username.includes(q) || name.includes(q);
    });
  }, [members, query]);

  useEffect(() => {
    if (!visible) return;
    const handleClick = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [visible, onClose]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!visible) return;
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIndex((i) => Math.min(i + 1, filtered.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        if (filtered[selectedIndex]?.user?.username) {
          onSelect(filtered[selectedIndex].user!.username);
        }
      } else if (e.key === "Escape") {
        onClose();
      }
    },
    [visible, filtered, selectedIndex, onSelect, onClose]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  if (!visible || filtered.length === 0) return null;

  return (
    <div
      ref={popoverRef}
      className="absolute z-50 w-64 max-h-48 overflow-y-auto bg-popover border border-border rounded-lg shadow-lg"
      style={{ top: position.top, left: position.left }}
    >
      {filtered.slice(0, 10).map((member, index) => (
        <button
          key={member.user_id ?? member.user?.id ?? index}
          type="button"
          className={`w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-muted/50 transition-colors ${
            index === selectedIndex ? "bg-muted/50" : ""
          }`}
          onMouseDown={(e) => {
            e.preventDefault();
            if (member.user?.username) {
              onSelect(member.user.username);
            }
          }}
          onMouseEnter={() => setSelectedIndex(index)}
        >
          {member.user?.avatar_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={member.user.avatar_url}
              alt=""
              className="w-6 h-6 rounded-full"
            />
          ) : (
            <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-xs font-medium text-primary">
              {(member.user?.username || "?")[0].toUpperCase()}
            </div>
          )}
          <div className="flex-1 min-w-0">
            <div className="truncate font-medium">
              {member.user?.name || member.user?.username}
            </div>
            {member.user?.name && (
              <div className="truncate text-xs text-muted-foreground">
                @{member.user.username}
              </div>
            )}
          </div>
        </button>
      ))}
    </div>
  );
}

export default MentionPopover;
