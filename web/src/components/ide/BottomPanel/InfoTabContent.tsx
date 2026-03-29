"use client";

import React from "react";
import Link from "next/link";
import { cn } from "@/lib/utils";
import type { PodData } from "@/lib/api/pod";
import { getPodDisplayName } from "@/lib/pod-utils";
import { getPodStatusInfo } from "@/stores/mesh";
import { usePodStore } from "@/stores/pod";
import { AgentStatusBadge } from "@/components/shared/AgentStatusBadge";
import { InfoRow } from "./InfoRow";
import { RelatedPodsList } from "./RelatedPodsList";
import {
  Terminal,
  Server,
  GitBranch,
  FolderGit2,
  Bot,
  Ticket,
  User,
  Clock,
  AlertCircle,
} from "lucide-react";

function getRelatedPods(pods: PodData[], pod: PodData | null): PodData[] {
  if (!pod?.ticket?.id) return [];
  return pods.filter(
    (p) => p.ticket?.id === pod.ticket?.id && p.pod_key !== pod.pod_key
  );
}

interface InfoTabContentProps {
  selectedPodKey: string | null;
  pod: PodData | null;
  orgSlug: string;
  t: (key: string, params?: Record<string, string | number>) => string;
}

export function InfoTabContent({
  selectedPodKey,
  pod,
  orgSlug,
  t,
}: InfoTabContentProps) {
  const pods = usePodStore((s) => s.pods);
  const relatedPods = getRelatedPods(pods, pod);

  if (!selectedPodKey) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
        <Terminal className="w-8 h-8 mb-2 opacity-50" />
        <span className="text-xs">{t("ide.bottomPanel.selectPodFirst")}</span>
      </div>
    );
  }

  if (!pod) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
        <Terminal className="w-8 h-8 mb-2 opacity-50" />
        <span className="text-xs">{t("ide.bottomPanel.infoTab.notFound")}</span>
      </div>
    );
  }

  const statusInfo = getPodStatusInfo(pod.status);

  return (
    <div className="h-full overflow-auto space-y-3">
      {/* Pod Name & Status */}
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium truncate">
          {getPodDisplayName(pod, 40)}
        </span>
        <span
          className={cn(
            "px-1.5 py-0.5 rounded text-[10px] font-medium whitespace-nowrap",
            statusInfo.color,
            statusInfo.bgColor
          )}
        >
          {statusInfo.label}
        </span>
      </div>

      {/* Info Grid */}
      <div className="grid grid-cols-2 gap-x-6 gap-y-1.5">
        {/* Pod Key */}
        <InfoRow
          icon={<Terminal className="w-3 h-3" />}
          label={t("ide.bottomPanel.infoTab.podKey")}
          value={pod.pod_key}
          mono
        />

        {/* Agent */}
        {pod.agent && (
          <InfoRow
            icon={<Bot className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.agent")}
            value={pod.agent.name}
          />
        )}

        {/* Agent Status */}
        {pod.agent_status && pod.status === "running" && (
          <InfoRow
            icon={<Bot className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.agentStatus")}
            value={
              <AgentStatusBadge
                agentStatus={pod.agent_status}
                podStatus={pod.status}
                variant="inline"
              />
            }
          />
        )}

        {/* Runner */}
        {pod.runner && (
          <InfoRow
            icon={<Server className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.runner")}
            value={pod.runner.node_id}
            mono
          />
        )}

        {/* Repository */}
        {pod.repository && (
          <InfoRow
            icon={<FolderGit2 className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.repository")}
            value={pod.repository.slug}
            href={orgSlug ? `/${orgSlug}/repositories/${pod.repository.id}` : undefined}
          />
        )}

        {/* Branch */}
        {pod.branch_name && (
          <InfoRow
            icon={<GitBranch className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.branch")}
            value={pod.branch_name}
            mono
          />
        )}

        {/* Sandbox Path (Worktree) */}
        {pod.sandbox_path && (
          <InfoRow
            icon={<FolderGit2 className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.worktree")}
            value={pod.sandbox_path}
            mono
          />
        )}

        {/* Ticket */}
        {pod.ticket && (
          <InfoRow
            icon={<Ticket className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.ticket")}
            value={`${pod.ticket.slug} - ${pod.ticket.title}`}
            href={orgSlug ? `/${orgSlug}/tickets/${pod.ticket.slug}` : undefined}
          />
        )}

        {/* Created By */}
        {pod.created_by && (
          <InfoRow
            icon={<User className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.createdBy")}
            value={pod.created_by.name || pod.created_by.username}
          />
        )}

        {/* Started At */}
        {pod.started_at && (
          <InfoRow
            icon={<Clock className="w-3 h-3" />}
            label={t("ide.bottomPanel.infoTab.startedAt")}
            value={new Date(pod.started_at).toLocaleString()}
          />
        )}

        {/* Created At */}
        <InfoRow
          icon={<Clock className="w-3 h-3" />}
          label={t("ide.bottomPanel.infoTab.createdAt")}
          value={new Date(pod.created_at).toLocaleString()}
        />

        {/* Error */}
        {pod.error_message && (
          <InfoRow
            icon={<AlertCircle className="w-3 h-3 text-red-500" />}
            label={t("ide.bottomPanel.infoTab.error")}
            value={`${pod.error_code ? `[${pod.error_code}] ` : ""}${pod.error_message}`}
            className="col-span-2"
            valueClassName="text-red-500"
          />
        )}
      </div>

      {/* Related Pods */}
      {relatedPods.length > 0 && (
        <RelatedPodsList relatedPods={relatedPods} t={t} />
      )}
    </div>
  );
}

export default InfoTabContent;
