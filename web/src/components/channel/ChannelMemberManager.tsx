"use client";

import { useState, useEffect, useCallback } from "react";
import { Users, Plus, X, Loader2, Crown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { channelApi } from "@/lib/api/channel";
import { organizationApi } from "@/lib/api/organization";
import type { OrganizationMember } from "@/lib/api/organization";
import { useCurrentOrg } from "@/stores/auth";
import { useChannelMembers, useChannelStore } from "@/stores/channelStore";
import { useTranslations } from "next-intl";

interface ChannelMemberManagerProps {
  channelId: number;
  memberCount: number;
  onMembersChanged?: () => void;
}

export function ChannelMemberManager({
  channelId,
  memberCount,
  onMembersChanged,
}: ChannelMemberManagerProps) {
  const t = useTranslations();
  const currentOrg = useCurrentOrg();

  const [open, setOpen] = useState(false);
  const members = useChannelMembers(channelId);
  const [orgMembers, setOrgMembers] = useState<OrganizationMember[]>([]);
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState<number | null>(null);

  useEffect(() => {
    if (!open) return;
    const load = async () => {
      setLoading(true);
      try {
        // Warm Rust cache; `useChannelMembers` picks it up via tick bump.
        await channelApi.listMembers(channelId);
        useChannelStore.setState((s) => ({ _tick: s._tick + 1 }));
        if (currentOrg?.slug) {
          const orgRes = await organizationApi.listMembers(currentOrg.slug);
          setOrgMembers(orgRes.members || []);
        }
      } catch (error) {
        console.error("Failed to load member data:", error);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [open, channelId, currentOrg?.slug]);

  const memberUserIds = new Set(members.map((m) => m.user_id));
  const availableMembers = orgMembers.filter(
    (m) => m.user?.id && !memberUserIds.has(m.user.id)
  );

  const handleInvite = useCallback(
    async (userId: number) => {
      setActionLoading(userId);
      try {
        await channelApi.inviteMembers(channelId, [userId]);
        useChannelStore.setState((s) => ({ _tick: s._tick + 1 }));
        onMembersChanged?.();
      } catch (error) {
        console.error("Failed to invite member:", error);
      } finally {
        setActionLoading(null);
      }
    },
    [channelId, onMembersChanged]
  );

  const handleRemove = useCallback(
    async (userId: number) => {
      setActionLoading(userId);
      try {
        await channelApi.removeMember(channelId, userId);
        useChannelStore.setState((s) => ({ _tick: s._tick + 1 }));
        onMembersChanged?.();
      } catch (error) {
        console.error("Failed to remove member:", error);
      } finally {
        setActionLoading(null);
      }
    },
    [channelId, onMembersChanged]
  );

  const getUserDisplay = (userId: number) => {
    const orgMember = orgMembers.find((m) => m.user?.id === userId);
    return orgMember?.user?.name || orgMember?.user?.username || `User #${userId}`;
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={t("channels.header.members")}
          title={t("channels.header.members")}
          data-testid="channel-header-members"
          className="flex items-center gap-1.5 px-2 py-1 bg-muted rounded-md hover:bg-muted/80 transition-colors"
        >
          <Users className="w-3.5 h-3.5 text-muted-foreground" />
          <span className="text-xs font-medium">{memberCount}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 p-0">
        <div className="p-3 border-b border-border">
          <h4 className="text-sm font-medium">{t("channels.members.title")}</h4>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-6">
            <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="max-h-64 overflow-y-auto">
            {members.length > 0 && (
              <div className="p-2">
                <p className="text-xs text-muted-foreground px-2 py-1">
                  {t("channels.members.current")} ({members.length})
                </p>
                {members.map((m) => (
                  <div
                    key={m.user_id}
                    className="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-muted/50 group"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      {m.role === "creator" ? (
                        <Crown className="w-3.5 h-3.5 text-amber-500 flex-shrink-0" />
                      ) : (
                        <Users className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />
                      )}
                      <span className="text-xs font-medium truncate">
                        {getUserDisplay(m.user_id)}
                      </span>
                    </div>
                    {m.role !== "creator" && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                        onClick={() => handleRemove(m.user_id)}
                        disabled={actionLoading === m.user_id}
                      >
                        {actionLoading === m.user_id ? (
                          <Loader2 className="w-3 h-3 animate-spin" />
                        ) : (
                          <X className="w-3 h-3 text-muted-foreground hover:text-destructive" />
                        )}
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            )}

            {availableMembers.length > 0 && (
              <div className={cn("p-2", members.length > 0 && "border-t border-border")}>
                <p className="text-xs text-muted-foreground px-2 py-1">
                  {t("channels.members.available")} ({availableMembers.length})
                </p>
                {availableMembers.slice(0, 20).map((m) => (
                  <div
                    key={m.user_id}
                    className="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-muted/50"
                  >
                    <span className="text-xs font-medium truncate">
                      {m.user?.name || m.user?.username || `User #${m.user_id}`}
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0"
                      onClick={() => m.user?.id && handleInvite(m.user.id)}
                      disabled={actionLoading === m.user_id}
                    >
                      {actionLoading === m.user_id ? (
                        <Loader2 className="w-3 h-3 animate-spin" />
                      ) : (
                        <Plus className="w-3 h-3 text-muted-foreground hover:text-primary" />
                      )}
                    </Button>
                  </div>
                ))}
              </div>
            )}

            {members.length === 0 && availableMembers.length === 0 && (
              <div className="py-6 text-center text-xs text-muted-foreground">
                {t("channels.members.empty")}
              </div>
            )}
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}
