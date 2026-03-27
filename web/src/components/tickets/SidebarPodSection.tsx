"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Ticket } from "@/stores/ticket";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { ticketApi } from "@/lib/api";
import { useWorkspaceStore } from "@/stores/workspace";
import { useAuthStore } from "@/stores/auth";
import { CreatePodModal } from "@/components/ide/CreatePodModal";
import { getPodDisplayName } from "@/lib/pod-utils";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";
import { Play, Terminal, ExternalLink } from "lucide-react";
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

export function SidebarPodSection({
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
    } catch { /* silently fail */ }
    finally { setLoading(false); }
  }, [ticketSlug]);

  useEffect(() => { fetchPods(); }, [fetchPods]);

  const handlePodCreated = () => { setShowCreateModal(false); fetchPods(); };
  const handleConnect = (podKey: string) => {
    addPane(podKey);
    router.push(`/${currentOrg?.slug}/workspace`);
  };
  const handleOpenInNewTab = (podKey: string) => {
    window.open(`/${currentOrg?.slug}/workspace?pod=${podKey}`, "_blank");
  };

  const activePods = pods.filter((p) => p.status === "running" || p.status === "initializing");
  const inactivePods = pods.filter((p) => p.status !== "running" && p.status !== "initializing");

  const ticketContext = useMemo(() =>
    ticket.id ? { id: ticket.id, slug: ticketSlug, title: ticket.title, repositoryId: ticket.repository_id } : undefined,
    [ticket.id, ticketSlug, ticket.title, ticket.repository_id]
  );

  return (
    <>
      <CreatePodModal open={showCreateModal} onClose={() => setShowCreateModal(false)}
        onCreated={handlePodCreated} ticketContext={ticketContext} />
      <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
        <div className="p-3">
          <Button className="w-full gap-1.5 shadow-sm" size="sm" onClick={() => setShowCreateModal(true)}>
            <Play className="h-3.5 w-3.5" />{t("tickets.podPanel.newPod")}
          </Button>
        </div>
        <div className="border-t border-border/40">
          {loading ? (
            <div className="flex items-center justify-center py-5"><Spinner size="sm" /></div>
          ) : pods.length === 0 ? (
            <div className="py-5 px-3 text-center">
              <Terminal className="w-5 h-5 mx-auto mb-2 text-muted-foreground/20" />
              <p className="text-xs text-muted-foreground/60">{t("tickets.podPanel.noPods")}</p>
            </div>
          ) : (
            <PodList activePods={activePods} inactivePods={inactivePods}
              onConnect={handleConnect} onOpenInNewTab={handleOpenInNewTab} />
          )}
        </div>
      </div>
    </>
  );
}

function PodList({ activePods, inactivePods, onConnect, onOpenInNewTab }: {
  activePods: TicketPod[]; inactivePods: TicketPod[];
  onConnect: (key: string) => void; onOpenInNewTab: (key: string) => void;
}) {
  const t = useTranslations();
  return (
    <div className="py-1">
      {activePods.map((pod) => (
        <SidebarPodItem key={pod.pod_key} pod={pod}
          onConnect={() => onConnect(pod.pod_key)} onOpenInNewTab={() => onOpenInNewTab(pod.pod_key)} />
      ))}
      {inactivePods.length > 0 && (
        <details className="group">
          <summary className="px-3 py-1.5 text-[11px] text-muted-foreground/60 cursor-pointer hover:bg-muted/30 select-none transition-colors">
            {t("tickets.podPanel.previousPods", { count: inactivePods.length })}
          </summary>
          <div className="pb-1">
            {inactivePods.map((pod) => (
              <SidebarPodItem key={pod.pod_key} pod={pod}
                onConnect={() => onConnect(pod.pod_key)} onOpenInNewTab={() => onOpenInNewTab(pod.pod_key)} />
            ))}
          </div>
        </details>
      )}
    </div>
  );
}

function SidebarPodItem({ pod, onConnect, onOpenInNewTab }: {
  pod: TicketPod; onConnect: () => void; onOpenInNewTab: () => void;
}) {
  const t = useTranslations();
  const isActive = pod.status === "running" || pod.status === "initializing";
  return (
    <div className={cn("mx-1.5 px-2 py-1.5 flex items-center gap-2 group transition-colors rounded-md",
      isActive ? "hover:bg-green-50/60 dark:hover:bg-green-900/10" : "hover:bg-muted/40")}>
      <div className={cn("w-1.5 h-1.5 rounded-full shrink-0",
        pod.status === "running" && "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.4)] animate-pulse",
        pod.status === "initializing" && "bg-yellow-500 shadow-[0_0_6px_rgba(234,179,8,0.4)] animate-pulse",
        pod.status === "failed" && "bg-red-500",
        pod.status !== "running" && pod.status !== "initializing" && pod.status !== "failed" && "bg-muted-foreground/30")} />
      <code className="text-[11px] font-mono text-muted-foreground/80 flex-1 truncate">
        {getPodDisplayName(pod)}
      </code>
      <AgentStatusBadge agentStatus={pod.agent_status} podStatus={pod.status} variant="dot" />
      {isActive && (
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
          <button type="button" onClick={onConnect}
            className="p-1 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title={t("tickets.podPanel.connect")}><Terminal className="w-3 h-3" /></button>
          <button type="button" onClick={onOpenInNewTab}
            className="p-1 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title={t("tickets.podPanel.openInNewTab")}><ExternalLink className="w-3 h-3" /></button>
        </div>
      )}
    </div>
  );
}
