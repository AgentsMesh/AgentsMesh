"use client";

import { Button } from "@/components/ui/button";
import { X, Radio, RefreshCw } from "lucide-react";
import { cn } from "@/lib/utils";
import { ChannelPodManager } from "./ChannelPodManager";

interface ChannelHeaderProps {
  name: string;
  description?: string;
  podCount: number;
  /** Channel ID for pod management */
  channelId: number;
  onClose: () => void;
  onRefresh?: () => void;
  loading?: boolean;
  /** Compact mode for embedded use (e.g., bottom panel) */
  compact?: boolean;
  /** Callback when pod membership changes */
  onPodsChanged?: () => void;
}

export function ChannelHeader({
  name,
  description,
  podCount,
  channelId,
  onClose,
  onRefresh,
  loading,
  compact = false,
  onPodsChanged,
}: ChannelHeaderProps) {
  // Compact mode for bottom panel
  if (compact) {
    return (
      <div className="flex items-center justify-between flex-1 min-w-0">
        <div className="flex items-center gap-2 min-w-0">
          <Radio className="w-3.5 h-3.5 text-blue-500 dark:text-blue-400 flex-shrink-0" />
          <span className="font-medium text-xs truncate">#{name}</span>
          <ChannelPodManager
            channelId={channelId}
            podCount={podCount}
            compact
            onPodsChanged={onPodsChanged}
          />
        </div>
        <div className="flex items-center gap-1 flex-shrink-0">
          {onRefresh && (
            <Button
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0"
              onClick={onRefresh}
              disabled={loading}
            >
              <RefreshCw className={cn("w-3.5 h-3.5", loading && "animate-spin")} />
            </Button>
          )}
        </div>
      </div>
    );
  }

  // Default full mode
  return (
    <div className="flex-shrink-0 border-b border-border">
      {/* Main header */}
      <div className="flex items-center justify-between px-4 py-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="w-8 h-8 rounded-lg bg-blue-500/10 flex items-center justify-center flex-shrink-0">
            <Radio className="w-4 h-4 text-blue-500 dark:text-blue-400" />
          </div>
          <div className="min-w-0">
            <h3 className="font-semibold text-sm truncate">#{name}</h3>
            {description && (
              <p className="text-xs text-muted-foreground truncate">
                {description}
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {/* Pod manager popover */}
          <ChannelPodManager
            channelId={channelId}
            podCount={podCount}
            onPodsChanged={onPodsChanged}
          />

          {/* Refresh button */}
          {onRefresh && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={onRefresh}
              disabled={loading}
            >
              <RefreshCw className={cn("w-4 h-4", loading && "animate-spin")} />
            </Button>
          )}

          {/* Close button */}
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onClose}
          >
            <X className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}

export default ChannelHeader;
