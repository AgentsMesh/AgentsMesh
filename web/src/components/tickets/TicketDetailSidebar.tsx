"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Ticket, TicketStatus, TicketPriority } from "@/stores/ticket";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { StatusSelect } from "./StatusSelect";
import { PrioritySelect } from "./PrioritySelect";
import { ticketApi } from "@/lib/api";
import { useWorkspaceStore } from "@/stores/workspace";
import { useAuthStore } from "@/stores/auth";
import { CreatePodModal } from "@/components/ide/CreatePodModal";
import { getPodDisplayName } from "@/lib/pod-utils";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";
import {
  Trash2, Clock, Play, Terminal,
  ExternalLink,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface TicketPod {
  pod_key: string;
  status: string;
  agent_status: string;
  model?: string;
  started_at?: string;
  runner_id: number;
  created_by_id: number;
}

interface TicketDetailSidebarProps {
  ticket: Ticket;
  onDelete: () => void;
  onStatusChange: (status: TicketStatus) => void;
  onPriorityChange?: (priority: TicketPriority) => void;
  ticketSlug: string;
  t: (key: string, params?: Record<string, string | number>) => string;
  commentsSlot?: React.ReactNode;
}

function formatRelativeDate(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffDay > 30) return date.toLocaleDateString();
  if (diffDay > 0) return `${diffDay}d ago`;
  if (diffHr > 0) return `${diffHr}h ago`;
  if (diffMin > 0) return `${diffMin}m ago`;
  return "just now";
}

export function TicketDetailSidebar({
  ticket,
  onDelete,
  onStatusChange,
  onPriorityChange,
  ticketSlug,
  t,
  commentsSlot,
}: TicketDetailSidebarProps) {
  const handleStatusChange = async (status: TicketStatus) => {
    onStatusChange(status);
  };

  const handlePriorityChange = async (priority: TicketPriority) => {
    onPriorityChange?.(priority);
  };

  return (
    <div className="lg:w-72 shrink-0 space-y-3">
      {/* Execute / AgentPods */}
      <SidebarPodSection
        ticket={ticket}
        ticketSlug={ticketSlug}
      />

      {/* Properties panel */}
      <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
        {/* Status */}
        <div className="px-4 py-3 hover:bg-muted/30 transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground">{t("tickets.filters.status")}</span>
            <StatusSelect
              value={ticket.status}
              onChange={handleStatusChange}
              showLabel
              size="sm"
            />
          </div>
        </div>

        <div className="mx-4 border-t border-border/40" />

        {/* Priority */}
        <div className="px-4 py-3 hover:bg-muted/30 transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground">{t("tickets.filters.priority")}</span>
            {onPriorityChange ? (
              <PrioritySelect
                value={ticket.priority}
                onChange={handlePriorityChange}
                showLabel
                size="sm"
              />
            ) : (
              <span className="text-sm">{t(`tickets.priority.${ticket.priority}`)}</span>
            )}
          </div>
        </div>

        <div className="mx-4 border-t border-border/40" />

        {/* Due Date */}
        {ticket.due_date && (
          <>
            <div className="px-4 py-3 hover:bg-muted/30 transition-colors">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground">{t("tickets.detail.dueDate")}</span>
                <span className={cn(
                  "text-sm tabular-nums",
                  new Date(ticket.due_date) < new Date() && ticket.status !== "done"
                    ? "text-destructive font-medium"
                    : "text-foreground"
                )}>
                  {new Date(ticket.due_date).toLocaleDateString()}
                </span>
              </div>
            </div>
            <div className="mx-4 border-t border-border/40" />
          </>
        )}

        {/* Assignees */}
        <div className="px-4 py-3">
          <span className="text-xs font-medium text-muted-foreground block mb-2.5">{t("tickets.detail.assignees")}</span>
          {ticket.assignees && ticket.assignees.length > 0 ? (
            <div className="flex flex-col gap-2">
              {ticket.assignees.map((assignee) => (
                <div key={assignee.user_id} className="flex items-center gap-2 group">
                  {assignee.user?.avatar_url ? (
                    /* eslint-disable-next-line @next/next/no-img-element */
                    <img src={assignee.user.avatar_url} alt="" className="w-6 h-6 rounded-full ring-1 ring-border/50" />
                  ) : (
                    <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-[10px] font-semibold text-primary ring-1 ring-primary/20">
                      {(assignee.user?.name || assignee.user?.username || "?")[0].toUpperCase()}
                    </div>
                  )}
                  <span className="text-sm text-foreground/90">{assignee.user?.name || assignee.user?.username}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground/50 italic">{t("tickets.detail.noAssignees")}</p>
          )}
        </div>

        <div className="mx-4 border-t border-border/40" />

        {/* Timestamps */}
        <div className="px-4 py-3">
          <div className="flex flex-col gap-1 text-xs text-muted-foreground/70">
            <div className="flex items-center gap-1.5">
              <Clock className="w-3 h-3 shrink-0" />
              <span title={new Date(ticket.created_at).toLocaleString()}>
                {t("tickets.detail.created")} {formatRelativeDate(ticket.created_at)}
              </span>
            </div>
            <div className="flex items-center gap-1.5 ml-[18px]">
              <span title={new Date(ticket.updated_at).toLocaleString()}>
                {t("tickets.detail.updated")} {formatRelativeDate(ticket.updated_at)}
              </span>
            </div>
          </div>
        </div>
      </div>

      {commentsSlot}

      {/* Delete */}
      <Button
        className="w-full"
        variant="outline"
        size="sm"
        onClick={onDelete}
      >
        <Trash2 className="h-3.5 w-3.5 mr-1.5 text-destructive" />
        <span className="text-destructive">{t("common.delete")}</span>
      </Button>
    </div>
  );
}

/**
 * AgentPods section for the sidebar — shows active pods and an Execute button.
 */
function SidebarPodSection({
  ticket,
  ticketSlug,
}: {
  ticket: Ticket;
  ticketSlug: string;
}) {
  const t = useTranslations();
  const router = useRouter();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const addPane = useWorkspaceStore((s) => s.addPane);

  const [pods, setPods] = useState<TicketPod[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const fetchPods = useCallback(async () => {
    setLoading(true);
    try {
      const response = await ticketApi.getPods(ticketSlug);
      setPods(response.pods || []);
    } catch {
      // silently fail
    } finally {
      setLoading(false);
    }
  }, [ticketSlug]);

  useEffect(() => {
    fetchPods();
  }, [fetchPods]);

  const handlePodCreated = () => {
    setShowCreateModal(false);
    fetchPods();
  };

  const handleConnect = (podKey: string) => {
    addPane(podKey);
    router.push(`/${currentOrg?.slug}/workspace`);
  };

  const handleOpenInNewTab = (podKey: string) => {
    window.open(`/${currentOrg?.slug}/workspace?pod=${podKey}`, "_blank");
  };

  const activePods = pods.filter(
    (p) => p.status === "running" || p.status === "initializing"
  );
  const inactivePods = pods.filter(
    (p) => p.status !== "running" && p.status !== "initializing"
  );

  // Memoize ticketContext to prevent CreatePodModal re-renders on parent re-render
  const ticketContext = useMemo(() =>
    ticket.id
      ? {
          id: ticket.id,
          slug: ticketSlug,
          title: ticket.title,
          repositoryId: ticket.repository_id,
        }
      : undefined,
    [ticket.id, ticketSlug, ticket.title, ticket.repository_id]
  );

  return (
    <>
      <CreatePodModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onCreated={handlePodCreated}
        ticketContext={ticketContext}
      />

      <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
        {/* Execute button */}
        <div className="p-3">
          <Button
            className="w-full gap-1.5 shadow-sm"
            size="sm"
            onClick={() => setShowCreateModal(true)}
          >
            <Play className="h-3.5 w-3.5" />
            {t("tickets.podPanel.newPod")}
          </Button>
        </div>

        {/* Pod list */}
        <div className="border-t border-border/40">
          {loading ? (
            <div className="flex items-center justify-center py-5">
              <Spinner size="sm" />
            </div>
          ) : pods.length === 0 ? (
            <div className="py-5 px-3 text-center">
              <Terminal className="w-5 h-5 mx-auto mb-2 text-muted-foreground/20" />
              <p className="text-xs text-muted-foreground/60">{t("tickets.podPanel.noPods")}</p>
            </div>
          ) : (
            <div className="py-1">
              {activePods.map((pod) => (
                <SidebarPodItem
                  key={pod.pod_key}
                  pod={pod}
                  onConnect={() => handleConnect(pod.pod_key)}
                  onOpenInNewTab={() => handleOpenInNewTab(pod.pod_key)}
                />
              ))}

              {inactivePods.length > 0 && (
                <details className="group">
                  <summary className="px-3 py-1.5 text-[11px] text-muted-foreground/60 cursor-pointer hover:bg-muted/30 select-none transition-colors">
                    {t("tickets.podPanel.previousPods", { count: inactivePods.length })}
                  </summary>
                  <div className="pb-1">
                    {inactivePods.map((pod) => (
                      <SidebarPodItem
                        key={pod.pod_key}
                        pod={pod}
                        onConnect={() => handleConnect(pod.pod_key)}
                        onOpenInNewTab={() => handleOpenInNewTab(pod.pod_key)}
                      />
                    ))}
                  </div>
                </details>
              )}
            </div>
          )}
        </div>
      </div>
    </>
  );
}

function SidebarPodItem({
  pod,
  onConnect,
  onOpenInNewTab,
}: {
  pod: TicketPod;
  onConnect: () => void;
  onOpenInNewTab: () => void;
}) {
  const t = useTranslations();
  const isActive = pod.status === "running" || pod.status === "initializing";

  return (
    <div
      className={cn(
        "mx-1.5 px-2 py-1.5 flex items-center gap-2 group transition-colors rounded-md",
        isActive ? "hover:bg-green-50/60 dark:hover:bg-green-900/10" : "hover:bg-muted/40"
      )}
    >
      <div
        className={cn(
          "w-1.5 h-1.5 rounded-full shrink-0",
          pod.status === "running" && "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.4)] animate-pulse",
          pod.status === "initializing" && "bg-yellow-500 shadow-[0_0_6px_rgba(234,179,8,0.4)] animate-pulse",
          pod.status === "failed" && "bg-red-500",
          pod.status !== "running" && pod.status !== "initializing" && pod.status !== "failed" && "bg-muted-foreground/30"
        )}
      />
      <code className="text-[11px] font-mono text-muted-foreground/80 flex-1 truncate">
        {getPodDisplayName(pod)}
      </code>
      <AgentStatusBadge
        agentStatus={pod.agent_status}
        podStatus={pod.status}
        variant="dot"
      />
      {isActive && (
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            type="button"
            onClick={onConnect}
            className="p-1 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title={t("tickets.podPanel.connect")}
          >
            <Terminal className="w-3 h-3" />
          </button>
          <button
            type="button"
            onClick={onOpenInNewTab}
            className="p-1 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title={t("tickets.podPanel.openInNewTab")}
          >
            <ExternalLink className="w-3 h-3" />
          </button>
        </div>
      )}
    </div>
  );
}

export default TicketDetailSidebar;
