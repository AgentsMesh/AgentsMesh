"use client";

import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { Wifi, WifiOff, Loader2, AlertTriangle } from "lucide-react";
import type { ConnectionStatus } from "@/stores/terminalConnection";

interface RelayStatusOverlayProps {
  connectionStatus: ConnectionStatus;
  isRunnerDisconnected: boolean;
  className?: string;
}

/**
 * Floating overlay that shows the Relay connection status
 * at the top of the terminal area. Always visible, real-time updates.
 */
export function RelayStatusOverlay({
  connectionStatus,
  isRunnerDisconnected,
  className,
}: RelayStatusOverlayProps) {
  const t = useTranslations();

  // Determine display state: runner disconnect takes priority over relay connected
  const isWarning = isRunnerDisconnected || connectionStatus === "disconnected" || connectionStatus === "error";
  const isConnecting = connectionStatus === "connecting";
  const isConnected = connectionStatus === "connected" && !isRunnerDisconnected;

  const getLabel = () => {
    if (isRunnerDisconnected) return t("relayStatus.runnerDisconnected");
    switch (connectionStatus) {
      case "connected":
        return t("relayStatus.connected");
      case "connecting":
        return t("relayStatus.connecting");
      case "disconnected":
        return t("relayStatus.disconnected");
      case "error":
        return t("relayStatus.error");
      default:
        return t("relayStatus.disconnected");
    }
  };

  const getIcon = () => {
    if (isRunnerDisconnected) {
      return <AlertTriangle className="w-3 h-3" />;
    }
    switch (connectionStatus) {
      case "connected":
        return <Wifi className="w-3 h-3" />;
      case "connecting":
        return <Loader2 className="w-3 h-3 animate-spin" />;
      case "disconnected":
      case "error":
        return <WifiOff className="w-3 h-3" />;
      default:
        return <WifiOff className="w-3 h-3" />;
    }
  };

  return (
    <div
      className={cn(
        "absolute top-0 left-0 right-0 z-10 flex items-center justify-center pointer-events-none",
        className
      )}
    >
      <div
        className={cn(
          "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-b-md text-[11px] font-medium",
          "shadow-sm backdrop-blur-sm transition-colors duration-300",
          isConnected && "bg-green-500/15 text-green-400 border-x border-b border-green-500/20",
          isConnecting && "bg-yellow-500/15 text-yellow-400 border-x border-b border-yellow-500/20",
          isWarning && !isConnecting && "bg-red-500/15 text-red-400 border-x border-b border-red-500/20",
        )}
      >
        {getIcon()}
        <span>{getLabel()}</span>
      </div>
    </div>
  );
}

export default RelayStatusOverlay;
