"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import type { Channel } from "@/stores/channel";
import { cn } from "@/lib/utils";
import { FileText, FolderGit2, Settings, Ticket } from "lucide-react";
import { ChannelPodManager } from "./ChannelPodManager";
import { ChannelMemberManager } from "./ChannelMemberManager";

interface ChannelRightRailProps {
  channel: Channel | null;
  channelId: number;
  onPodsChanged?: () => void;
  onOpenSettings?: () => void;
}

/**
 * Right drawer — design/pages/channels.pastel `drawer_rail`. Four sections
 * in a fixed 280px column:
 *   PODS · N     → pod list + manager
 *   MEMBERS · N  → member manager
 *   LINKED       → ticket + repo (conditional)
 *   DOCUMENT     → channel.document preview + settings link
 *
 * Hidden unless the header "⋯" button toggles it on. No duplicated
 * controls on the header — everything lives here.
 */
export function ChannelRightRail({
  channel,
  channelId,
  onPodsChanged,
  onOpenSettings,
}: ChannelRightRailProps) {
  const t = useTranslations();
  const currentOrg = useCurrentOrg();
  const pods = channel?.pods ?? [];
  const podCount = pods.length;
  const memberCount = channel?.member_count ?? 0;
  const document = channel?.document?.trim();

  return (
    <aside
      data-testid="channel-right-rail"
      className="flex w-[280px] flex-shrink-0 flex-col gap-5 overflow-y-auto border-l border-border bg-muted/30 p-4"
    >
      <section className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <SectionTitle>{`${t("channels.rightRail.pods")} · ${podCount}`}</SectionTitle>
          <ChannelPodManager
            channelId={channelId}
            podCount={podCount}
            onPodsChanged={onPodsChanged}
          />
        </div>
        {pods.length === 0 ? (
          <EmptyRail>{t("channels.rightRail.podsEmpty")}</EmptyRail>
        ) : (
          <ul className="flex flex-col gap-1">
            {pods.map((pod) => (
              <li
                key={pod.pod_key}
                className="flex items-center gap-2.5 rounded-md px-2 py-1.5 hover:bg-muted"
              >
                <PodAvatar
                  letter={(pod.alias ?? pod.pod_key)[0]?.toUpperCase() ?? "?"}
                  status={pod.status}
                />
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate font-mono text-[12px] text-foreground">
                    {pod.alias ?? pod.pod_key}
                  </span>
                  {pod.agent?.name && (
                    <span className="truncate font-mono text-[10px] text-muted-foreground">
                      {pod.agent.name}
                    </span>
                  )}
                </span>
                <StatusDot status={pod.status} />
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <SectionTitle>{`${t("channels.rightRail.members")} · ${memberCount}`}</SectionTitle>
          <ChannelMemberManager channelId={channelId} memberCount={memberCount} />
        </div>
      </section>

      {(channel?.ticket || channel?.repository) && (
        <section className="flex flex-col gap-2">
          <SectionTitle>{t("channels.rightRail.linked")}</SectionTitle>
          {channel?.ticket && currentOrg && (
            <Link
              href={`/${currentOrg.slug}/tickets/${channel.ticket.slug}`}
              className="flex flex-col gap-1.5 rounded-md border border-border bg-background p-2.5 hover:bg-muted"
            >
              <div className="flex items-center gap-1.5">
                <Ticket className="h-3 w-3 text-muted-foreground" />
                <span className="font-mono text-[11px] text-muted-foreground">
                  {channel.ticket.slug}
                </span>
              </div>
              <span className="text-[12px] text-foreground">{channel.ticket.title}</span>
            </Link>
          )}
          {channel?.repository && currentOrg && (
            <Link
              href={`/${currentOrg.slug}/infra?tab=repositories&id=${channel.repository.id}`}
              className="flex items-center gap-2 rounded-md border border-border bg-background p-2.5 hover:bg-muted"
            >
              <FolderGit2 className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="truncate font-mono text-[12px] text-foreground">
                {channel.repository.name}
              </span>
            </Link>
          )}
        </section>
      )}

      <section className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <SectionTitle>{t("channels.rightRail.document")}</SectionTitle>
          {onOpenSettings && (
            <button
              type="button"
              onClick={onOpenSettings}
              aria-label={t("channels.header.settings")}
              title={t("channels.header.settings")}
              data-testid="channel-rail-settings"
              className="inline-flex h-5 w-5 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <Settings className="h-3 w-3" />
            </button>
          )}
        </div>
        {document ? (
          <div className="flex items-start gap-2 rounded-md border border-border bg-background p-2.5">
            <FileText className="mt-0.5 h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
            <p className="whitespace-pre-wrap break-words text-[12px] leading-5 text-foreground">
              {document}
            </p>
          </div>
        ) : (
          <EmptyRail>{t("channels.rightRail.documentEmpty")}</EmptyRail>
        )}
      </section>
    </aside>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <span className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">
      {children}
    </span>
  );
}

function EmptyRail({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-md border border-dashed border-border px-3 py-2 text-[11px] text-muted-foreground">
      {children}
    </div>
  );
}

function PodAvatar({ letter, status }: { letter: string; status: string }) {
  const bg =
    status === "running"
      ? "bg-emerald-500"
      : status === "initializing"
        ? "bg-amber-500"
        : status === "terminated"
          ? "bg-muted-foreground/50"
          : "bg-red-500";
  return (
    <span className={cn("flex h-7 w-7 items-center justify-center rounded-md font-mono text-xs font-semibold text-white", bg)}>
      {letter}
    </span>
  );
}

function StatusDot({ status }: { status: string }) {
  const color =
    status === "running"
      ? "bg-emerald-500"
      : status === "initializing"
        ? "bg-amber-500"
        : "bg-muted-foreground/50";
  return <span className={cn("h-1.5 w-1.5 rounded-full", color)} />;
}

export default ChannelRightRail;
