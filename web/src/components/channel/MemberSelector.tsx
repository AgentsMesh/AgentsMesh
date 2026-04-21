"use client";

import { useState, useEffect, useMemo, useCallback } from "react";
import { X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { organizationApi } from "@/lib/api/organization";
import type { OrganizationMember } from "@/lib/api/organization";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";

interface MemberSelectorProps {
  selectedIds: number[];
  onChange: (ids: number[]) => void;
}

export function MemberSelector({ selectedIds, onChange }: MemberSelectorProps) {
  const t = useTranslations();
  const currentOrg = useCurrentOrg();
  const [orgMembers, setOrgMembers] = useState<OrganizationMember[]>([]);
  const [search, setSearch] = useState("");

  useEffect(() => {
    if (!currentOrg) return;
    organizationApi.listMembers(currentOrg.slug).then((res) => {
      setOrgMembers(res.members || []);
    }).catch(() => {});
  }, [currentOrg]);

  const filtered = useMemo(() => orgMembers.filter((m) => {
    if (!m.user) return false;
    if (selectedIds.includes(m.user.id)) return false;
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      m.user.username?.toLowerCase().includes(q) ||
      m.user.name?.toLowerCase().includes(q) ||
      m.user.email?.toLowerCase().includes(q)
    );
  }), [orgMembers, selectedIds, search]);

  const toggle = useCallback((userId: number) => {
    onChange(
      selectedIds.includes(userId)
        ? selectedIds.filter((id) => id !== userId)
        : [...selectedIds, userId]
    );
  }, [selectedIds, onChange]);

  const getDisplay = (userId: number) => {
    const m = orgMembers.find((om) => om.user?.id === userId);
    return m?.user?.name || m?.user?.username || `User #${userId}`;
  };

  return (
    <div>
      {selectedIds.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-2">
          {selectedIds.map((uid) => (
            <span
              key={uid}
              className="inline-flex items-center gap-1 px-2 py-0.5 bg-primary/10 text-primary rounded-full text-xs"
            >
              {getDisplay(uid)}
              <button type="button" onClick={() => toggle(uid)} className="hover:text-destructive">
                <X className="w-3 h-3" />
              </button>
            </span>
          ))}
        </div>
      )}
      <Input
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder={t("channels.members.searchPlaceholder")}
        className="mb-1"
      />
      <div className="max-h-32 overflow-y-auto border border-border rounded-md">
        {filtered.slice(0, 10).map((m) => (
          <button
            key={m.user_id}
            type="button"
            className="w-full text-left px-3 py-1.5 text-xs hover:bg-muted/50 transition-colors"
            onClick={() => m.user?.id && toggle(m.user.id)}
          >
            <span className="font-medium">{m.user?.name || m.user?.username}</span>
            {m.user?.email && (
              <span className="text-muted-foreground ml-2">{m.user.email}</span>
            )}
          </button>
        ))}
        {filtered.length === 0 && (
          <p className="px-3 py-2 text-xs text-muted-foreground">{t("channels.members.empty")}</p>
        )}
      </div>
    </div>
  );
}
