"use client";

import { useTranslations } from "next-intl";
import { AcpPlanTracker } from "@/components/workspace/acp/AcpPlanTracker";
import { AcpActivityStream } from "@/components/workspace/acp/AcpActivityStream";

// Plan tracker + scrollable activity stream. The prompt input lives at the page
// root (below the dock) so the composer stays pinned to the bottom and the data
// panels expand above it without shifting the input position.
export function LoopalActivityColumn({ podKey }: { podKey: string }) {
  const t = useTranslations("loopal");
  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
      <h2 className="border-b border-border px-4 py-2 text-xs font-medium uppercase text-muted-foreground">
        {t("activity.heading")}
      </h2>
      <AcpPlanTracker podKey={podKey} />
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <AcpActivityStream podKey={podKey} />
      </div>
    </div>
  );
}
