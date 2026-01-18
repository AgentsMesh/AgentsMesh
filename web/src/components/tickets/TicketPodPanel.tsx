"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "@/lib/i18n/client";
import { Button } from "@/components/ui/button";
import { ticketApi } from "@/lib/api/client";
import { getPodStatusInfo, getAgentStatusInfo } from "@/stores/mesh";
import { useWorkspaceStore } from "@/stores/workspace";
import { useAuthStore } from "@/stores/auth";
import { Terminal, ExternalLink } from "lucide-react";
import { CreatePodForm } from "@/components/pod/CreatePodForm";

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
  ticketIdentifier: string;
  ticketTitle: string;
  ticketDescription?: string;
  ticketId?: number;
  onPodCreated?: () => void;
}

export default function TicketPodPanel({
  ticketIdentifier,
  ticketTitle,
  ticketDescription,
  ticketId,
  onPodCreated,
}: TicketPodPanelProps) {
  const t = useTranslations();
  const [pods, setPods] = useState<TicketPod[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);

  const fetchPods = useCallback(async () => {
    try {
      const response = await ticketApi.getPods(ticketIdentifier);
      setPods(response.pods || []);
    } catch (err: unknown) {
      console.error("Failed to fetch pods:", err);
    }
  }, [ticketIdentifier]);

  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      await fetchPods();
      setLoading(false);
    };
    loadData();
  }, [fetchPods]);

  // Poll for pod updates
  useEffect(() => {
    const interval = setInterval(fetchPods, 5000);
    return () => clearInterval(interval);
  }, [fetchPods]);

  const handlePodCreated = () => {
    setShowCreateForm(false);
    fetchPods();
    onPodCreated?.();
  };

  const activePods = pods.filter(
    (s) => s.status === "running" || s.status === "initializing"
  );
  const inactivePods = pods.filter(
    (s) => s.status !== "running" && s.status !== "initializing"
  );

  if (loading) {
    return (
      <div className="p-4 border border-border rounded-lg">
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg">
      {/* Header */}
      <div className="px-4 py-3 border-b border-border flex items-center justify-between">
        <div className="flex items-center gap-2">
          <svg className="w-5 h-5 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <h3 className="font-medium">AgentPods</h3>
          {activePods.length > 0 && (
            <span className="px-2 py-0.5 text-xs rounded-full bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
              {t("tickets.podPanel.activeCount", { count: activePods.length })}
            </span>
          )}
        </div>
        <Button
          size="sm"
          onClick={() => setShowCreateForm(!showCreateForm)}
        >
          <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          {t("tickets.podPanel.newPod")}
        </Button>
      </div>

      {/* Create Form - 使用共享的 CreatePodForm */}
      {showCreateForm && (
        <div className="p-4 border-b border-border bg-muted/30">
          <h4 className="text-sm font-medium mb-3">{t("tickets.podPanel.createNewPod")}</h4>
          <CreatePodForm
            enabled={showCreateForm}
            config={{
              scenario: "ticket",
              context: {
                ticket: ticketId ? {
                  id: ticketId,
                  identifier: ticketIdentifier,
                  title: ticketTitle,
                  description: ticketDescription,
                } : undefined,
              },
              promptPlaceholder: t("tickets.podPanel.initialPromptPlaceholder", { title: ticketTitle }),
              onSuccess: handlePodCreated,
              onCancel: () => setShowCreateForm(false),
            }}
          />
        </div>
      )}

      {/* Pods List */}
      <div className="divide-y divide-border">
        {/* Active Pods */}
        {activePods.map((pod) => (
          <PodItem key={pod.pod_key} pod={pod} ticketIdentifier={ticketIdentifier} />
        ))}

        {/* Inactive Pods (collapsed by default if there are active ones) */}
        {inactivePods.length > 0 && (
          <details className="group">
            <summary className="px-4 py-2 text-sm text-muted-foreground cursor-pointer hover:bg-muted/50">
              {t("tickets.podPanel.previousPods", { count: inactivePods.length })}
            </summary>
            <div className="divide-y divide-border border-t border-border">
              {inactivePods.map((pod) => (
                <PodItem key={pod.pod_key} pod={pod} ticketIdentifier={ticketIdentifier} />
              ))}
            </div>
          </details>
        )}

        {/* Empty State */}
        {pods.length === 0 && !showCreateForm && (
          <div className="px-4 py-8 text-center text-muted-foreground">
            <svg className="w-10 h-10 mx-auto mb-2 text-muted-foreground/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p className="text-sm">{t("tickets.podPanel.noPods")}</p>
          </div>
        )}
      </div>
    </div>
  );
}

interface PodItemProps {
  pod: TicketPod;
  ticketIdentifier: string;
}

function PodItem({ pod, ticketIdentifier }: PodItemProps) {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();
  const { addPane } = useWorkspaceStore();
  const statusInfo = getPodStatusInfo(pod.status);
  const agentInfo = getAgentStatusInfo(pod.agent_status);
  const isActive = pod.status === "running" || pod.status === "initializing";

  const handleConnect = () => {
    // Add to workspace and navigate
    addPane(pod.pod_key, `${ticketIdentifier} Pod`);
    router.push(`/${currentOrg?.slug}/workspace`);
  };

  const handleOpenInNewTab = () => {
    // Open pod detail in new tab
    window.open(`/${currentOrg?.slug}/workspace?pod=${pod.pod_key}`, "_blank");
  };

  return (
    <div className={`px-4 py-3 ${isActive ? "bg-green-50/50 dark:bg-green-900/10" : ""}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {/* Status Indicator */}
          <div
            className={`w-2 h-2 rounded-full ${
              pod.status === "running"
                ? "bg-green-500 animate-pulse"
                : pod.status === "initializing"
                ? "bg-yellow-500 animate-pulse"
                : pod.status === "failed"
                ? "bg-red-500"
                : "bg-gray-400"
            }`}
          />

          {/* Pod Info */}
          <div>
            <code className="text-xs font-mono text-muted-foreground">
              {pod.pod_key.substring(0, 12)}...
            </code>
            <div className="flex items-center gap-2 mt-0.5">
              <span
                className={`px-1.5 py-0.5 text-xs rounded ${statusInfo.bgColor} ${statusInfo.color}`}
              >
                {statusInfo.label}
              </span>
              {isActive && (
                <span className={`text-xs flex items-center gap-1 ${agentInfo.color}`}>
                  <span>{agentInfo.icon}</span>
                  {agentInfo.label}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {pod.model && (
            <span className="text-xs text-muted-foreground hidden sm:inline">{pod.model}</span>
          )}
          {isActive && (
            <>
              <Button size="sm" variant="outline" onClick={handleConnect}>
                <Terminal className="w-3.5 h-3.5 mr-1" />
                {t("tickets.podPanel.connect")}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="hidden sm:flex"
                onClick={handleOpenInNewTab}
                title={t("tickets.podPanel.openInNewTab")}
              >
                <ExternalLink className="w-3.5 h-3.5" />
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Started At */}
      {pod.started_at && (
        <div className="mt-1 text-xs text-muted-foreground ml-5">
          {t("tickets.podPanel.started")}: {new Date(pod.started_at).toLocaleString()}
        </div>
      )}
    </div>
  );
}
