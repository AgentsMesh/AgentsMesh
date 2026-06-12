"use client";

import { useTranslations } from "next-intl";
import { Maximize2 } from "lucide-react";
import { LoopalTopologyFlow } from "../topology/LoopalTopologyFlow";

// In-dock mini topology. The expand button hands off to the full-screen Sheet
// owned by the page.
export function AgentsSection({ podKey, onExpand }: { podKey: string; onExpand?: () => void }) {
  const t = useTranslations("loopal");
  return (
    <div className="relative h-60">
      {onExpand && (
        <button
          type="button"
          onClick={onExpand}
          title={t("dock.agents.expand")}
          data-testid="loopal-topology-expand"
          className="absolute right-2 top-2 z-10 rounded bg-background/80 p-1 text-muted-foreground shadow-sm hover:text-foreground"
        >
          <Maximize2 className="h-3.5 w-3.5" />
        </button>
      )}
      <LoopalTopologyFlow podKey={podKey} />
    </div>
  );
}
