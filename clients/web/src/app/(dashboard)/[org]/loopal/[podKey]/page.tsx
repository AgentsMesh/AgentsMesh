"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useAcpRelay } from "@/hooks/useAcpRelay";
import { LoopalActivityColumn } from "@/components/loopal/LoopalActivityColumn";
import { LoopalPromptInput } from "@/components/loopal/prompt/LoopalPromptInput";
import { LoopalTopBar } from "@/components/loopal/status/LoopalTopBar";
import { LoopalBottomDock } from "@/components/loopal/dock/LoopalBottomDock";
import { LoopalTopologySheet } from "@/components/loopal/topology/LoopalTopologySheet";

export default function LoopalConsolePage() {
  const params = useParams();
  const podKey = typeof params.podKey === "string" ? params.podKey : "";
  useAcpRelay(podKey, `loopal-${podKey}`, !!podKey);
  const [topoOpen, setTopoOpen] = useState(false);

  // Order matters: activity (flex-1) shrinks as the dock expands, but the dock
  // and the prompt input stay pinned to the bottom — panels open *above* the
  // composer, keeping the input position fixed.
  return (
    <div className="flex h-full w-full min-w-0 flex-col overflow-hidden">
      <LoopalTopBar podKey={podKey} onOpenTopology={() => setTopoOpen(true)} />
      <LoopalActivityColumn podKey={podKey} />
      <LoopalBottomDock podKey={podKey} onExpandTopology={() => setTopoOpen(true)} />
      <LoopalPromptInput podKey={podKey} />
      <LoopalTopologySheet podKey={podKey} open={topoOpen} onOpenChange={setTopoOpen} />
    </div>
  );
}
