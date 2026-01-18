"use client";

import { Button } from "@/components/ui/button";
import { X, Radio, Users, RefreshCw } from "lucide-react";
import { cn } from "@/lib/utils";

interface ChannelHeaderProps {
  name: string;
  description?: string;
  podCount: number;
  onClose: () => void;
  onRefresh?: () => void;
  loading?: boolean;
}

export function ChannelHeader({
  name,
  description,
  podCount,
  onClose,
  onRefresh,
  loading,
}: ChannelHeaderProps) {
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
          {/* Pod count badge */}
          <div className="flex items-center gap-1.5 px-2 py-1 bg-muted rounded-md">
            <Users className="w-3.5 h-3.5 text-muted-foreground" />
            <span className="text-xs font-medium">{podCount}</span>
          </div>

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
