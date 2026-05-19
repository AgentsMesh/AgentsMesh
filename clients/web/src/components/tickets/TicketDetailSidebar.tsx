"use client";

import Link from "next/link";
import { Ticket } from "@/stores/ticket";
import type { TicketRelation } from "@/lib/api/ticketTypes";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { cn } from "@/lib/utils";
import { CheckCircle2, Circle, GitPullRequest, Clock, Terminal } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTicketPods } from "@/hooks/useTicketPods";
import { useWorkspaceStore } from "@/stores/workspace";
import { getShortPodKey } from "@/lib/pod-display-name";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";

interface TicketDetailSidebarProps {
  ticket: Ticket;
  ticketSlug: string;
  subTickets?: Ticket[];
  relations?: TicketRelation[];
  commits?: Array<{
    sha?: string;
    message?: string;
    hash?: string;
    short_sha?: string;
    created_at?: string;
    url?: string;
  }>;
  t: (key: string, params?: Record<string, string | number>) => string;
  commentsSlot?: React.ReactNode;
}

export function TicketDetailSidebar({
  ticket,
  ticketSlug,
  subTickets = [],
  relations = [],
  commits = [],
  t,
  commentsSlot,
}: TicketDetailSidebarProps) {
  const router = useRouter();
  const currentOrg = useCurrentOrg();
  const addPane = useWorkspaceStore((s) => s.addPane);
  const { pods, loading: podsLoading } = useTicketPods(ticketSlug);

  const activePods = pods.filter((p) => p.status === "running" || p.status === "initializing");

  const handleOpenPod = (podKey: string) => {
    addPane(podKey);
    router.push(`/${currentOrg?.slug}/workspace`);
  };

  return (
    <aside className="lg:w-80 shrink-0 space-y-4">
      <RailSection title={t("tickets.rail.workingPods")} count={activePods.length}>
        {podsLoading ? (
          <RailEmpty icon={<Terminal className="h-4 w-4" />} text={t("common.loading")} />
        ) : activePods.length === 0 ? (
          <RailEmpty icon={<Terminal className="h-4 w-4" />} text={t("tickets.rail.noPods")} />
        ) : (
          <ul className="space-y-1">
            {activePods.map((pod) => (
              <li key={pod.pod_key}>
                <button
                  type="button"
                  onClick={() => handleOpenPod(pod.pod_key)}
                  className="w-full rounded-md px-2 py-1.5 text-left transition-colors hover:bg-muted"
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex min-w-0 items-center gap-2">
                      <span
                        className={cn(
                          "h-2 w-2 flex-shrink-0 rounded-full",
                          pod.status === "running" ? "bg-success" : "bg-warning",
                        )}
                      />
                      <span className="truncate font-mono text-[12px] font-medium text-foreground">
                        {getShortPodKey(pod.pod_key)}
                      </span>
                    </div>
                    <AgentStatusBadge agentStatus={pod.agent_status} podStatus={pod.status} variant="badge" />
                  </div>
                </button>
              </li>
            ))}
          </ul>
        )}
      </RailSection>

      <RailSection title={t("tickets.rail.subTickets")} count={subTickets.length}>
        {subTickets.length === 0 ? (
          <RailEmpty icon={<Circle className="h-4 w-4" />} text={t("tickets.rail.noSubTickets")} />
        ) : (
          <ul className="space-y-1">
            {subTickets.map((st) => {
              const isDone = st.status === "done";
              return (
                <li key={st.slug}>
                  <Link
                    href={`/${currentOrg?.slug}/tickets/${st.slug}`}
                    className="flex items-start gap-2 rounded-md px-2 py-1.5 hover:bg-muted"
                  >
                    {isDone ? (
                      <CheckCircle2 className="mt-0.5 h-3.5 w-3.5 flex-shrink-0 text-success" />
                    ) : (
                      <Circle className="mt-0.5 h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1.5">
                        <span className="font-mono text-[10px] text-muted-foreground">
                          {st.slug}
                        </span>
                      </div>
                      <div
                        className={cn(
                          "truncate text-[12px]",
                          isDone ? "text-muted-foreground line-through" : "text-foreground",
                        )}
                      >
                        {st.title}
                      </div>
                    </div>
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </RailSection>

      <RailSection title={t("tickets.rail.pullRequests")} count={commits.length}>
        {commits.length === 0 ? (
          <RailEmpty icon={<GitPullRequest className="h-4 w-4" />} text={t("tickets.rail.noPRs")} />
        ) : (
          <ul className="space-y-1">
            {commits.slice(0, 5).map((c, idx) => (
              <li
                key={c.sha ?? c.hash ?? idx}
                className="flex items-start gap-2 rounded-md px-2 py-1.5 hover:bg-muted"
              >
                <GitPullRequest className="mt-0.5 h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <div className="font-mono text-[10px] text-muted-foreground">
                    {(c.short_sha ?? c.sha)?.slice(0, 7)}
                  </div>
                  <div className="truncate text-[12px] text-foreground">{c.message}</div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </RailSection>

      <RailSection title={t("tickets.rail.activity")}>
        <ul className="space-y-2">
          <ActivityRow time={ticket.updated_at} text={t("tickets.rail.activityUpdated")} />
          <ActivityRow time={ticket.created_at} text={t("tickets.rail.activityCreated")} />
          {relations.slice(0, 2).map((rel, idx) => (
            <ActivityRow
              key={`rel-${idx}`}
              time={undefined}
              text={`${rel.relation_type}: ${rel.target_ticket?.slug ?? "—"}`}
            />
          ))}
        </ul>
      </RailSection>

      {commentsSlot}
    </aside>
  );
}

function RailSection({
  title,
  count,
  children,
}: {
  title: string;
  count?: number;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-md border border-border bg-card">
      <header className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
          {title}
        </span>
        {typeof count === "number" && count > 0 && (
          <span className="font-mono text-[11px] text-muted-foreground">{count}</span>
        )}
      </header>
      <div className="p-2">{children}</div>
    </section>
  );
}

function RailEmpty({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="flex items-center gap-2 px-2 py-3 text-[12px] text-muted-foreground/70">
      {icon}
      <span>{text}</span>
    </div>
  );
}

function formatRelative(time: string): string {
  const diffMs = Date.now() - new Date(time).getTime();
  const hours = Math.floor(diffMs / (60 * 60 * 1000));
  const days = Math.floor(hours / 24);
  return days > 0 ? `${days}d ago` : hours > 0 ? `${hours}h ago` : "just now";
}

function ActivityRow({ time, text }: { time?: string; text: string }) {
  if (!time) return null;
  const rel = formatRelative(time);
  return (
    <li className="flex items-start gap-2 px-2">
      <Clock className="mt-0.5 h-3 w-3 flex-shrink-0 text-muted-foreground/60" />
      <div className="flex-1 text-[12px]">
        <div className="text-foreground">{text}</div>
        <div className="text-[10px] text-muted-foreground">{rel}</div>
      </div>
    </li>
  );
}

export default TicketDetailSidebar;
