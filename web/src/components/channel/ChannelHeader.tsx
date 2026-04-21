"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Hash, Lock, Search, MoreHorizontal, LogIn } from "lucide-react";
import { cn } from "@/lib/utils";
import { useChannelStore, type Channel } from "@/stores/channel";

interface ChannelHeaderProps {
  name: string;
  channelId: number;
  visibility?: "public" | "private";
  isMember?: boolean;
  podCount: number;
  memberCount?: number;
  ticket?: Channel["ticket"];
  repository?: Channel["repository"];
  onOpenSearch?: () => void;
  /** Toggles the right-side drawer (Pods / Members / Linked / Document). */
  onToggleRail?: () => void;
  railOpen?: boolean;
  compact?: boolean;
}

/**
 * Channel header — design/pages/channels.pastel `chan_header`. Left: `#` +
 * name + subtitle (pods · members · linked · repo). Right: 🔍 search plus
 * ⋯ "More" which opens/closes the right drawer (members are inside the
 * drawer, no duplicate pill on the header).
 */
export function ChannelHeader({
  name,
  channelId,
  visibility = "public",
  isMember = true,
  podCount,
  memberCount = 0,
  ticket,
  repository,
  onOpenSearch,
  onToggleRail,
  railOpen = true,
  compact = false,
}: ChannelHeaderProps) {
  const t = useTranslations();
  const joinUserChannel = useChannelStore((s) => s.joinUserChannel);
  const [joining, setJoining] = useState(false);

  const isPrivate = visibility === "private";
  const Icon = isPrivate ? Lock : Hash;

  const handleJoin = async () => {
    setJoining(true);
    try {
      await joinUserChannel(channelId);
    } finally {
      setJoining(false);
    }
  };

  if (compact) {
    return (
      <div className="flex min-w-0 flex-1 items-center justify-between">
        <div className="flex min-w-0 items-center gap-2">
          <Icon className={cn("h-3.5 w-3.5 flex-shrink-0", isPrivate ? "text-amber-500" : "text-muted-foreground")} />
          <span className="truncate text-xs font-medium">{name}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-shrink-0 items-center justify-between border-b border-border px-6 py-3">
      <div className="flex min-w-0 items-center gap-2.5">
        <Icon
          className={cn(
            "h-5 w-5 flex-shrink-0",
            isPrivate ? "text-amber-500" : "text-muted-foreground",
          )}
          aria-hidden="true"
        />
        <div className="flex min-w-0 flex-col">
          <h2 className="truncate text-base font-semibold text-foreground">{name}</h2>
          <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
            <span>{t("channels.header.podsCount", { count: podCount })}</span>
            <Dot />
            <span>{t("channels.header.membersCount", { count: memberCount })}</span>
            {ticket && (
              <>
                <Dot />
                <span className="text-primary">
                  {t("channels.header.linkedTo")} {ticket.slug}
                </span>
              </>
            )}
            {repository && (
              <>
                <Dot />
                <span className="font-mono text-muted-foreground">{repository.name}</span>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="flex flex-shrink-0 items-center gap-1.5">
        {!isMember && !isPrivate ? (
          <Button size="sm" onClick={handleJoin} disabled={joining}>
            <LogIn className="mr-1.5 h-3.5 w-3.5" />
            {t("channels.actions.join")}
          </Button>
        ) : (
          <>
            <HeaderIconButton
              onClick={onOpenSearch}
              label={t("channels.header.search")}
              testId="channel-header-search"
            >
              <Search className="h-3.5 w-3.5" />
            </HeaderIconButton>
            <HeaderIconButton
              onClick={onToggleRail}
              label={t("channels.header.more")}
              testId="channel-header-more"
              active={railOpen}
            >
              <MoreHorizontal className="h-3.5 w-3.5" />
            </HeaderIconButton>
          </>
        )}
      </div>
    </div>
  );
}

function Dot() {
  return <span aria-hidden="true" className="text-border">·</span>;
}

function HeaderIconButton({
  children,
  onClick,
  label,
  testId,
  active,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  label: string;
  testId?: string;
  active?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      title={label}
      aria-pressed={active}
      data-testid={testId}
      className={cn(
        "inline-flex h-[30px] min-w-[30px] items-center justify-center rounded-md border px-2 transition-colors",
        active
          ? "border-primary/40 bg-primary/10 text-primary"
          : "border-border bg-background text-foreground hover:bg-muted",
      )}
    >
      {children}
    </button>
  );
}

export default ChannelHeader;
