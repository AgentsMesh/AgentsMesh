"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { ticketApi } from "@/lib/api";
import { useWorkspaceStore } from "@/stores/workspace";
import { useAuthStore } from "@/stores/auth";
import { Terminal, ExternalLink, Plus } from "lucide-react";
import { CreatePodModal } from "@/components/ide/CreatePodModal";
import { getPodDisplayName } from "@/lib/pod-utils";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";

interface TicketPod {
  pod_key: string;
  status: string;
  agent_status: string;
  model?: string;
  started_at?: string;
  runner_id: number;
  created_by_id: number;
}

interface TicketPodPanelProps {
  ticketSlug: string;
  ticketTitle: string;
  ticketId?: number;
  repositoryId?: number;
  onPodCreated?: () => void;
}

export default function TicketPodPanel({
  ticketSlug,
  ticketTitle,
  ticketId,
  repositoryId,
  onPodCreated,
}: TicketPodPanelProps) {
  const t = useTranslations();
  const [pods, setPods] = useState<TicketPod[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);

  const fetchPods = useCallback(async () => {
    try {
      const response = await ticketApi.getPods(ticketSlug);
      setPods(response.pods || []);
    } catch (err: unknown) {
      console.error("Failed to fetch pods:", err);
    }
  }, [ticketSlug]);

  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      await fetchPods();
      setLoading(false);
    };
    loadData();
  }, [fetchPods]);


  const handlePodCreated = () => {
    setShowCreateForm(false);
    fetchPods();
    onPodCreated?.();
  };

  const handleCloseModal = () => {
    setShowCreateForm(false);
  };

  const activePods = useMemo(() => pods.filter(
    (s) => s.status === "running" || s.status === "initializing"
  ), [pods]);
  const inactivePods = useMemo(() => pods.filter(
    (s) => s.status !== "running" && s.status !== "initializing"
  ), [pods]);

  if (loading) {
    return (
      <div className="p-4 border border-border rounded-lg">
        <div className="flex items-center justify-center py-8">
          <Spinner size="sm" />
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Create Pod Modal */}
      <CreatePodModal
        open={showCreateForm}
        onClose={handleCloseModal}
        onCreated={handlePodCreated}
        ticketContext={
          ticketId
            ? {
                id: ticketId,
                slug: ticketSlug,
                title: ticketTitle,
                repositoryId: repositoryId,
              }
            : undefined
        }
      />

      <div className="space-y-2">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Terminal className="w-4 h-4 text-muted-foreground" />
            <span className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider">
              AgentPods
            </span>
            {activePods.length > 0 && (
              <span className="px-1.5 py-0.5 text-[10px] rounded-full bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
                {activePods.length} {t("tickets.podPanel.active")}
              </span>
            )}
          </div>
          <Button
            size="sm"
            variant="ghost"
            className="h-7 px-2 text-xs"
            onClick={() => setShowCreateForm(true)}
          >
            <Plus className="w-3.5 h-3.5 mr-1" />
            {t("tickets.podPanel.newPod")}
          </Button>
        </div>

        {/* Pods List */}
        <div className="space-y-1">
        {/* Active Pods */}
        {activePods.map((pod) => (
          <PodItem key={pod.pod_key} pod={pod} />
        ))}

          {/* Inactive Pods (collapsed by default if there are active ones) */}
          {inactivePods.length > 0 && (
            <details className="group">
              <summary className="px-2.5 py-1.5 text-xs text-muted-foreground cursor-pointer hover:bg-muted/50 rounded-md">
                {t("tickets.podPanel.previousPods", { count: inactivePods.length })}
              </summary>
              <div className="mt-1 space-y-1">
                {inactivePods.map((pod) => (
                  <PodItem key={pod.pod_key} pod={pod} />
                ))}
              </div>
            </details>
          )}

          {/* Empty State */}
          {pods.length === 0 && (
            <div className="py-4 text-center text-muted-foreground">
              <Terminal className="w-8 h-8 mx-auto mb-2 text-muted-foreground/30" />
              <p className="text-xs">{t("tickets.podPanel.noPods")}</p>
            </div>
          )}
        </div>
      </div>
    </>
  );
}

interface PodItemProps {
  pod: TicketPod;
}

function PodItem({ pod }: PodItemProps) {
  const t = useTranslations();
  const router = useRouter();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const addPane = useWorkspaceStore((s) => s.addPane);
  const isActive = pod.status === "running" || pod.status === "initializing";

  const handleConnect = () => {
    // Add to workspace and navigate
    addPane(pod.pod_key);
    router.push(`/${currentOrg?.slug}/workspace`);
  };

  const handleOpenInNewTab = () => {
    // Open pod detail in new tab
    window.open(`/${currentOrg?.slug}/workspace?pod=${pod.pod_key}`, "_blank");
  };

  return (
    <div
      className={`px-2.5 py-1.5 rounded-md flex items-center gap-2 group transition-colors ${
        isActive ? "hover:bg-green-50/50 dark:hover:bg-green-900/10" : "hover:bg-muted/50"
      }`}
    >
      {/* Status Indicator */}
      <div
        className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
          pod.status === "running"
            ? "bg-green-500 animate-pulse"
            : pod.status === "initializing"
            ? "bg-yellow-500 animate-pulse"
            : pod.status === "failed"
            ? "bg-red-500"
            : "bg-gray-400"
        }`}
      />

      {/* Pod Name */}
      <code className="text-xs font-mono text-muted-foreground flex-1 truncate">
        {getPodDisplayName(pod)}
      </code>
      <AgentStatusBadge
        agentStatus={pod.agent_status}
        podStatus={pod.status}
        variant="dot"
      />

      {/* Actions - show on hover */}
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        {isActive && (
          <>
            <Button
              size="sm"
              variant="ghost"
              className="h-6 px-2 text-xs"
              onClick={handleConnect}
            >
              <Terminal className="w-3 h-3 mr-1" />
              {t("tickets.podPanel.connect")}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="h-6 w-6 p-0"
              onClick={handleOpenInNewTab}
              title={t("tickets.podPanel.openInNewTab")}
            >
              <ExternalLink className="w-3 h-3" />
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
