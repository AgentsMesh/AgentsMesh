"use client";

import { useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";

interface Channel {
  id: number;
  name: string;
  description?: string;
  isArchived: boolean;
  createdAt: string;
  updatedAt: string;
  repository?: {
    id: number;
    name: string;
  };
  ticket?: {
    id: number;
    identifier: string;
    title: string;
  };
  pods?: Array<{
    podKey: string;
    status: string;
    agentType?: {
      name: string;
    };
  }>;
}

interface ChannelListProps {
  channels: Channel[];
  selectedId?: number;
  onSelect?: (channel: Channel) => void;
  onArchive?: (id: number) => void;
  onUnarchive?: (id: number) => void;
}

export function ChannelList({
  channels,
  selectedId,
  onSelect,
  onArchive,
  onUnarchive,
}: ChannelListProps) {
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
  const [showArchived, setShowArchived] = useState(false);

  const filteredChannels = showArchived
    ? channels
    : channels.filter((c) => !c.isArchived);

  const activePodCount = (channel: Channel) =>
    channel.pods?.filter((p) => p.status === "running").length || 0;

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b">
        <h2 className="font-semibold">{t("mesh.channelList.title")}</h2>
        <div className="flex items-center gap-2">
          <button
            className={`text-xs px-2 py-1 rounded ${
              showArchived ? "bg-muted" : "text-muted-foreground"
            }`}
            onClick={() => setShowArchived(!showArchived)}
          >
            {showArchived ? t("mesh.channelList.hideArchived") : t("mesh.channelList.showArchived")}
          </button>
        </div>
      </div>

      {/* Channel List */}
      <div className="flex-1 overflow-y-auto">
        {filteredChannels.length === 0 ? (
          <div className="p-4 text-center text-muted-foreground text-sm">
            {t("mesh.channelList.noChannels")}
          </div>
        ) : (
          <div className="space-y-1 p-2">
            {filteredChannels.map((channel) => (
              <div
                key={channel.id}
                className={`group p-3 rounded-lg cursor-pointer transition-colors ${
                  selectedId === channel.id
                    ? "bg-primary/10 border border-primary/20"
                    : "hover:bg-muted"
                } ${channel.isArchived ? "opacity-60" : ""}`}
                onClick={() => onSelect?.(channel)}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-lg">#</span>
                      <span className="font-medium truncate">{channel.name}</span>
                      {channel.isArchived && (
                        <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                          {t("mesh.channelList.archived")}
                        </span>
                      )}
                    </div>
                    {channel.description && (
                      <p className="text-xs text-muted-foreground mt-1 line-clamp-1">
                        {channel.description}
                      </p>
                    )}
                  </div>

                  {/* Active Pods Badge */}
                  {activePodCount(channel) > 0 && (
                    <div className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                      <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
                      {activePodCount(channel)}
                    </div>
                  )}
                </div>

                {/* Related Ticket */}
                {channel.ticket && (
                  <div className="mt-2 text-xs text-muted-foreground">
                    <Link
                      href={`/${currentOrg?.slug}/tickets/${channel.ticket.identifier}`}
                      className="hover:text-primary hover:underline"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {channel.ticket.identifier}
                    </Link>
                  </div>
                )}

                {/* Actions (visible on hover) */}
                <div className="hidden group-hover:flex items-center gap-1 mt-2">
                  {channel.isArchived ? (
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-6 text-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        onUnarchive?.(channel.id);
                      }}
                    >
                      {t("mesh.channelList.unarchive")}
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-6 text-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        onArchive?.(channel.id);
                      }}
                    >
                      {t("mesh.channelList.archive")}
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default ChannelList;
