"use client";

import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import type { ConnectionStatus } from "@/stores/relayConnection";

type SegmentStatus = "ok" | "connecting" | "warning" | "unknown";

interface RelayStatusOverlayProps {
  connectionStatus: ConnectionStatus;
  isRunnerDisconnected: boolean;
  className?: string;
}

/** Derive segment statuses from connection props. */
function deriveSegments(
  connectionStatus: ConnectionStatus,
  isRunnerDisconnected: boolean,
): { webRelay: SegmentStatus; relayRunner: SegmentStatus } {
  const webRelay: SegmentStatus =
    connectionStatus === "connected" ? "ok"
    : connectionStatus === "connecting" ? "connecting"
    : "warning";

  const relayRunner: SegmentStatus =
    connectionStatus !== "connected" ? "unknown"
    : isRunnerDisconnected ? "warning"
    : "ok";

  return { webRelay, relayRunner };
}

/** Worst-case status for overall badge background. */
function worstStatus(a: SegmentStatus, b: SegmentStatus): SegmentStatus {
  const priority: Record<SegmentStatus, number> = { warning: 3, connecting: 2, unknown: 1, ok: 0 };
  return priority[a] >= priority[b] ? a : b;
}

const dotColor: Record<SegmentStatus, string> = {
  ok: "bg-green-500",
  connecting: "bg-yellow-500 animate-pulse",
  warning: "bg-red-500",
  unknown: "bg-gray-500",
};

const lineColor: Record<SegmentStatus, string> = {
  ok: "bg-green-500/60",
  connecting: "bg-yellow-500/60",
  warning: "bg-red-500/60",
  unknown: "bg-gray-500/40",
};

const badgeBg: Record<SegmentStatus, string> = {
  ok: "bg-green-500/15 border-green-500/20",
  connecting: "bg-yellow-500/15 border-yellow-500/20",
  warning: "bg-red-500/15 border-red-500/20",
  unknown: "bg-gray-500/15 border-gray-500/20",
};

const labelColor: Record<SegmentStatus, string> = {
  ok: "text-green-400",
  connecting: "text-yellow-400",
  warning: "text-red-400",
  unknown: "text-gray-400",
};

/** Derive tooltip i18n key directly from raw props (avoids SegmentStatus lossy mapping). */
function webRelayTooltipKey(connectionStatus: ConnectionStatus): string {
  switch (connectionStatus) {
    case "connected": return "connected";
    case "connecting": return "connecting";
    case "disconnected": return "disconnected";
    case "error": return "error";
  }
}

function relayRunnerTooltipKey(connectionStatus: ConnectionStatus, disconnected: boolean): string {
  if (connectionStatus !== "connected") return "unknown";
  return disconnected ? "disconnected" : "connected";
}

/**
 * Floating overlay showing the Web → Relay → Runner connection chain
 * at the top of the terminal / ACP panel.
 */
export function RelayStatusOverlay({
  connectionStatus,
  isRunnerDisconnected,
  className,
}: RelayStatusOverlayProps) {
  const t = useTranslations("relayStatus");
  const { webRelay, relayRunner } = deriveSegments(connectionStatus, isRunnerDisconnected);
  const overall = worstStatus(webRelay, relayRunner);
  const webRelayTip = t(webRelayTooltipKey(connectionStatus));
  const relayRunnerTip = t(relayRunnerTooltipKey(connectionStatus, isRunnerDisconnected));

  return (
    <div
      className={cn(
        "absolute top-0 left-0 right-0 z-10 flex items-center justify-center pointer-events-none",
        className,
      )}
    >
      <div
        className={cn(
          "inline-flex items-center gap-1 px-2.5 py-0.5 rounded-b-md text-[11px] font-medium",
          "shadow-sm backdrop-blur-sm transition-colors duration-300 border-x border-b",
          badgeBg[overall],
        )}
      >
        <span className={labelColor[webRelay]}>{t("web")}</span>
        <SegmentDot status={webRelay} title={webRelayTip} />
        <span className={cn("h-px w-3 inline-block", lineColor[webRelay])} />
        <span className="text-gray-300">{t("relay")}</span>
        <span className={cn("h-px w-3 inline-block", lineColor[relayRunner])} />
        <SegmentDot status={relayRunner} title={relayRunnerTip} />
        <span className={labelColor[relayRunner]}>{t("runner")}</span>
      </div>
    </div>
  );
}

/** Small colored dot representing a segment's status. */
function SegmentDot({ status, title }: { status: SegmentStatus; title: string }) {
  return (
    <span
      className={cn("w-1.5 h-1.5 rounded-full inline-block flex-shrink-0", dotColor[status])}
      title={title}
      role="status"
      aria-label={title}
    />
  );
}

export default RelayStatusOverlay;
