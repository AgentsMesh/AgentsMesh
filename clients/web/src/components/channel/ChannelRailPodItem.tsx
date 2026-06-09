import { cn } from "@/lib/utils";
import type { ChannelPodSummary } from "@/hooks/useChannelPods";

interface ChannelRailPodItemProps {
  pod: ChannelPodSummary;
  dimmed?: boolean;
}

export function ChannelRailPodItem({ pod, dimmed }: ChannelRailPodItemProps) {
  const label = pod.alias ?? pod.pod_key;
  return (
    <li
      data-testid="channel-rail-pod"
      data-status={pod.status}
      className={cn(
        "flex items-center gap-2.5 rounded-md px-2 py-1.5 hover:bg-muted",
        dimmed && "opacity-60",
      )}
    >
      <PodAvatar letter={label[0]?.toUpperCase() ?? "?"} status={pod.status} />
      <span className="flex min-w-0 flex-1 flex-col">
        <span
          className={cn(
            "truncate font-mono text-[12px]",
            dimmed ? "text-muted-foreground line-through" : "text-foreground",
          )}
        >
          {label}
        </span>
      </span>
      <StatusDot status={pod.status} />
    </li>
  );
}

function statusColorClass(status: string): string {
  if (status === "running") return "bg-emerald-500";
  if (status === "initializing") return "bg-amber-500";
  if (status === "error" || status === "failed") return "bg-red-500";
  return "bg-muted-foreground/50";
}

function PodAvatar({ letter, status }: { letter: string; status: string }) {
  return (
    <span className={cn("flex h-7 w-7 items-center justify-center rounded-md font-mono text-xs font-semibold text-white", statusColorClass(status))}>
      {letter}
    </span>
  );
}

function StatusDot({ status }: { status: string }) {
  return <span className={cn("h-1.5 w-1.5 rounded-full", statusColorClass(status))} />;
}
