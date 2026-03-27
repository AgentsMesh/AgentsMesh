"use client";

import React, { useState, useEffect, useCallback, useMemo } from "react";
import { cn } from "@/lib/utils";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { toast } from "sonner";
import { repositoryApi } from "@/lib/api/repository";
import { useEventSubscription } from "@/hooks/useRealtimeEvents";
import type { MREventData, PipelineEventData } from "@/lib/realtime";
import type { PodData } from "@/lib/api/pod";
import { GitPullRequest, RefreshCw, Loader2, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MergeRequestCard, MergeRequestInfo } from "./MergeRequestCard";

interface DeliveryTabContentProps {
  selectedPodKey: string | null;
  pod: PodData | null;
  t: (key: string, params?: Record<string, string | number>) => string;
}

export function DeliveryTabContent({ selectedPodKey, pod, t }: DeliveryTabContentProps) {
  const [mergeRequests, setMergeRequests] = useState<MergeRequestInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canShowDelivery = useMemo(() => {
    return !!(pod?.repository?.id && pod?.branch_name);
  }, [pod?.repository?.id, pod?.branch_name]);

  const providerType = useMemo(() => pod?.repository?.provider_type, [pod?.repository?.provider_type]);

  const fetchMRs = useCallback(async () => {
    if (!pod?.repository?.id || !pod?.branch_name) return;
    setLoading(true);
    setError(null);
    try {
      const response = await repositoryApi.listMergeRequests(pod.repository.id, pod.branch_name);
      setMergeRequests(response.merge_requests as MergeRequestInfo[]);
    } catch (err) {
      const msg = getLocalizedErrorMessage(err, t, t("ide.bottomPanel.deliveryTab.loadError"));
      setError(msg);
      toast.error(msg);
    } finally { setLoading(false); }
  }, [pod?.repository?.id, pod?.branch_name, t]);

  useEffect(() => {
    if (canShowDelivery) fetchMRs();
    else setMergeRequests([]);
  }, [canShowDelivery, fetchMRs]);

  const handleMREvent = useCallback(
    (event: { data: MREventData }) => {
      if (event.data.repository_id !== pod?.repository?.id) return;
      if (event.data.source_branch && event.data.source_branch !== pod?.branch_name) return;
      fetchMRs();
    },
    [pod?.repository?.id, pod?.branch_name, fetchMRs]
  );

  const handlePipelineEvent = useCallback(
    (event: { data: PipelineEventData }) => {
      if (event.data.repository_id !== pod?.repository?.id) return;
      setMergeRequests((prev) =>
        prev.map((mr) =>
          mr.id === event.data.mr_id
            ? { ...mr, pipeline_status: event.data.pipeline_status, pipeline_url: event.data.pipeline_url }
            : mr
        )
      );
    },
    [pod?.repository?.id]
  );

  useEventSubscription("mr:created", handleMREvent);
  useEventSubscription("mr:updated", handleMREvent);
  useEventSubscription("mr:merged", handleMREvent);
  useEventSubscription("mr:closed", handleMREvent);
  useEventSubscription("pipeline:updated", handlePipelineEvent);

  if (!selectedPodKey) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
        <Terminal className="w-8 h-8 mb-2 opacity-50" />
        <span className="text-xs">{t("ide.bottomPanel.selectPodFirst")}</span>
      </div>
    );
  }

  if (!canShowDelivery) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
        <GitPullRequest className="w-8 h-8 mb-2 opacity-50" />
        <span className="text-xs">{t("ide.bottomPanel.deliveryTab.notAvailable")}</span>
        <span className="text-[10px] mt-1 opacity-70">{t("ide.bottomPanel.deliveryTab.requiresRepoBranch")}</span>
      </div>
    );
  }

  if (loading && mergeRequests.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-4 h-4 animate-spin mr-2" />
        <span className="text-muted-foreground text-xs">{t("ide.bottomPanel.deliveryTab.loading")}</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
        <span className="text-xs text-destructive">{error}</span>
        <Button variant="ghost" size="sm" onClick={fetchMRs} className="mt-2">
          <RefreshCw className="w-3 h-3 mr-1" />{t("common.refresh")}
        </Button>
      </div>
    );
  }

  if (mergeRequests.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
        <GitPullRequest className="w-8 h-8 mb-2 opacity-50" />
        <span className="text-xs">{t("ide.bottomPanel.deliveryTab.noMergeRequests")}</span>
      </div>
    );
  }

  return (
    <div className="space-y-2 h-full overflow-auto">
      <div className="flex items-center justify-between sticky top-0 bg-background pb-1">
        <span className="text-xs text-muted-foreground">
          {t("ide.bottomPanel.deliveryTab.mrCount", { count: mergeRequests.length })}
        </span>
        <Button variant="ghost" size="sm" onClick={fetchMRs} className="h-6 w-6 p-0">
          <RefreshCw className={cn("w-3 h-3", loading && "animate-spin")} />
        </Button>
      </div>
      <div className="space-y-1.5">
        {mergeRequests.map((mr) => (
          <MergeRequestCard key={mr.id} mr={mr} providerType={providerType} t={t} />
        ))}
      </div>
    </div>
  );
}

export default DeliveryTabContent;
