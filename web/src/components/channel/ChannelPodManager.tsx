"use client";

import { useState, useEffect, useCallback } from "react";
import { Bot, Plus, X, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { channelApi } from "@/lib/api/channel";
import { usePodStore } from "@/stores/pod";
import { useTranslations } from "next-intl";

interface ChannelPod {
  id: number;
  pod_key: string;
  status: string;
  agent_status: string;
}

interface ChannelPodManagerProps {
  channelId: number;
  podCount: number;
  /** Compact mode for embedded use (e.g., bottom panel) */
  compact?: boolean;
  /** Callback when pod list changes */
  onPodsChanged?: () => void;
}

/**
 * Popover component for managing pods in a channel.
 * Shows joined pods and allows adding/removing active pods.
 */
export function ChannelPodManager({
  channelId,
  podCount,
  compact = false,
  onPodsChanged,
}: ChannelPodManagerProps) {
  const t = useTranslations();
  const allPods = usePodStore((s) => s.pods);
  const fetchPods = usePodStore((s) => s.fetchPods);

  const [open, setOpen] = useState(false);
  const [channelPods, setChannelPods] = useState<ChannelPod[]>([]);
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Fetch channel pods and active pods when popover opens
  useEffect(() => {
    if (!open) return;

    const loadData = async () => {
      setLoading(true);
      try {
        const [channelPodsRes] = await Promise.all([
          channelApi.getPods(channelId),
          fetchPods({ status: "running" }),
        ]);
        setChannelPods(channelPodsRes.pods || []);
      } catch (error) {
        console.error("Failed to load pod data:", error);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [open, channelId, fetchPods]);

  // Filter active pods not yet in the channel
  const joinedKeys = new Set(channelPods.map((p) => p.pod_key));
  const availablePods = allPods.filter(
    (p) =>
      (p.status === "running" || p.status === "initializing") &&
      !joinedKeys.has(p.pod_key)
  );

  // Add pod to channel
  const handleJoin = useCallback(
    async (podKey: string) => {
      setActionLoading(podKey);
      try {
        await channelApi.joinPod(channelId, podKey);
        // Refresh channel pods
        const res = await channelApi.getPods(channelId);
        setChannelPods(res.pods || []);
        onPodsChanged?.();
      } catch (error) {
        console.error("Failed to add pod to channel:", error);
      } finally {
        setActionLoading(null);
      }
    },
    [channelId, onPodsChanged]
  );

  // Remove pod from channel
  const handleLeave = useCallback(
    async (podKey: string) => {
      setActionLoading(podKey);
      try {
        await channelApi.leavePod(channelId, podKey);
        // Refresh channel pods
        const res = await channelApi.getPods(channelId);
        setChannelPods(res.pods || []);
        onPodsChanged?.();
      } catch (error) {
        console.error("Failed to remove pod from channel:", error);
      } finally {
        setActionLoading(null);
      }
    },
    [channelId, onPodsChanged]
  );

  // Short display key for pod
  const shortKey = (podKey: string) => podKey.slice(0, 12);

  // Find pod detail from store
  const getPodDetail = (podKey: string) =>
    allPods.find((p) => p.pod_key === podKey);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        {compact ? (
          <button
            type="button"
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <Bot className="w-3 h-3" />
            <span>{podCount}</span>
          </button>
        ) : (
          <button
            type="button"
            className="flex items-center gap-1.5 px-2 py-1 bg-muted rounded-md hover:bg-muted/80 transition-colors"
          >
            <Bot className="w-3.5 h-3.5 text-muted-foreground" />
            <span className="text-xs font-medium">{podCount}</span>
          </button>
        )}
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 p-0">
        <div className="p-3 border-b border-border">
          <h4 className="text-sm font-medium">
            {t("mesh.channelPodManager.title")}
          </h4>
          <p className="text-xs text-muted-foreground mt-0.5">
            {t("mesh.channelPodManager.description")}
          </p>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-6">
            <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="max-h-64 overflow-y-auto">
            {/* Joined pods */}
            {channelPods.length > 0 && (
              <div className="p-2">
                <p className="text-xs text-muted-foreground px-2 py-1">
                  {t("mesh.channelPodManager.joined")} ({channelPods.length})
                </p>
                {channelPods.map((pod) => {
                  const detail = getPodDetail(pod.pod_key);
                  return (
                    <div
                      key={pod.pod_key}
                      className="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-muted/50 group"
                    >
                      <div className="flex items-center gap-2 min-w-0">
                        <Bot className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                        <div className="min-w-0">
                          <p className="text-xs font-medium truncate">
                            {detail?.title || detail?.agent_type?.name || shortKey(pod.pod_key)}
                          </p>
                          <p className="text-[10px] text-muted-foreground truncate">
                            {shortKey(pod.pod_key)}
                          </p>
                        </div>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                        onClick={() => handleLeave(pod.pod_key)}
                        disabled={actionLoading === pod.pod_key}
                      >
                        {actionLoading === pod.pod_key ? (
                          <Loader2 className="w-3 h-3 animate-spin" />
                        ) : (
                          <X className="w-3 h-3 text-muted-foreground hover:text-destructive" />
                        )}
                      </Button>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Available pods to add */}
            {availablePods.length > 0 && (
              <div className={cn("p-2", channelPods.length > 0 && "border-t border-border")}>
                <p className="text-xs text-muted-foreground px-2 py-1">
                  {t("mesh.channelPodManager.available")} ({availablePods.length})
                </p>
                {availablePods.map((pod) => (
                  <div
                    key={pod.pod_key}
                    className="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-muted/50"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <Bot className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />
                      <div className="min-w-0">
                        <p className="text-xs font-medium truncate">
                          {pod.title || pod.agent_type?.name || shortKey(pod.pod_key)}
                        </p>
                        <p className="text-[10px] text-muted-foreground truncate">
                          {shortKey(pod.pod_key)}
                        </p>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0"
                      onClick={() => handleJoin(pod.pod_key)}
                      disabled={actionLoading === pod.pod_key}
                    >
                      {actionLoading === pod.pod_key ? (
                        <Loader2 className="w-3 h-3 animate-spin" />
                      ) : (
                        <Plus className="w-3 h-3 text-muted-foreground hover:text-primary" />
                      )}
                    </Button>
                  </div>
                ))}
              </div>
            )}

            {/* Empty state */}
            {channelPods.length === 0 && availablePods.length === 0 && (
              <div className="py-6 text-center text-xs text-muted-foreground">
                {t("mesh.channelPodManager.empty")}
              </div>
            )}
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

export default ChannelPodManager;
