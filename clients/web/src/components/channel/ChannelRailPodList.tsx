"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ChannelPodSummary } from "@/hooks/useChannelPods";
import { isDestroyedPodStatus } from "./podLifecycle";
import { ChannelRailPodItem } from "./ChannelRailPodItem";

export function ChannelRailPodList({ pods }: { pods: ChannelPodSummary[] }) {
  const t = useTranslations();
  const alive = pods.filter((p) => !isDestroyedPodStatus(p.status));
  const destroyed = pods.filter((p) => isDestroyedPodStatus(p.status));
  const [userToggled, setUserToggled] = useState<boolean | null>(null);
  const showDestroyed = userToggled ?? alive.length === 0;

  return (
    <div className="flex flex-col gap-1">
      {alive.length > 0 && (
        <ul className="flex flex-col gap-1">
          {alive.map((pod) => (
            <ChannelRailPodItem key={pod.pod_key} pod={pod} />
          ))}
        </ul>
      )}
      {destroyed.length > 0 && (
        <div className="flex flex-col gap-1">
          <button
            type="button"
            onClick={() => setUserToggled(!showDestroyed)}
            aria-expanded={showDestroyed}
            data-testid="channel-rail-destroyed-toggle"
            className="flex items-center gap-1 rounded px-2 py-1 text-[11px] font-medium text-muted-foreground hover:bg-muted"
          >
            <ChevronRight
              className={cn("h-3 w-3 transition-transform", showDestroyed && "rotate-90")}
            />
            <span>{`${t("channels.rightRail.destroyed")} · ${destroyed.length}`}</span>
          </button>
          {showDestroyed && (
            <ul className="flex flex-col gap-1">
              {destroyed.map((pod) => (
                <ChannelRailPodItem key={pod.pod_key} pod={pod} dimmed />
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
