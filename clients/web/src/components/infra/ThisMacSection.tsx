"use client";

import { useRouter } from "next/navigation";
import { Laptop, Loader2, AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";
import { useCurrentOrg } from "@/stores/auth";
import { useRunners, getRunnerStatusInfo } from "@/stores/runner";
import { Button } from "@/components/ui/button";
import { STEP_LABELS, useLocalRunnerOnboarding } from "@/hooks/useLocalRunnerOnboarding";
import { useOrphanGrace } from "@/hooks/useOrphanGrace";

const SECTION_HEADER = "px-3 pt-3 pb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground";

export function ThisMacSection() {
  const { unsupported, isRegistered, localNodeId, phase, onRegister } = useLocalRunnerOnboarding();
  const router = useRouter();
  const currentOrg = useCurrentOrg();
  const runners = useRunners();
  // Grace-wait so fresh registrants don't flicker through OrphanedBlock
  // while the local→backend heartbeat catches up, and so an empty list
  // ("never fetched" vs. "truly empty") doesn't false-positive without
  // coupling to the store loading flag. Must run before any early
  // return — React's rules-of-hooks demand a stable call order.
  const graceExpired = useOrphanGrace(isRegistered);

  if (unsupported) return null;
  if (phase.kind === "loading") {
    return (
      <div className="border-t border-border">
        <div className={SECTION_HEADER}>This Mac</div>
        <div className="flex items-center gap-2 px-3 pb-3 text-xs text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          <span>Checking…</span>
        </div>
      </div>
    );
  }

  const matchingRunner = localNodeId
    ? runners.find((r) => r.node_id === localNodeId) ?? null
    : null;
  // Orphan = registered but the runners list (after load) doesn't
  // contain this node_id, after the orphan grace window has elapsed.
  const orphaned =
    isRegistered && !matchingRunner && runners.length > 0 && graceExpired;
  const isStale = phase.kind === "idle" && phase.status === "stale";

  return (
    <div className="border-t border-border">
      <div className={SECTION_HEADER}>This Mac</div>
      {isRegistered && matchingRunner ? (
        <RegisteredRow
          runner={matchingRunner}
          serviceRunning={phase.kind === "idle" && phase.status === "running"}
          onClick={() => {
            if (currentOrg) {
              router.push(`/${currentOrg.slug}/infra?tab=runners&id=${matchingRunner.id}`);
            }
          }}
        />
      ) : orphaned ? (
        <OrphanedBlock onReRegister={onRegister} />
      ) : isRegistered ? (
        <div
          data-testid="this-mac-syncing"
          className="px-3 pb-3 text-xs text-muted-foreground"
        >
          Active locally · syncing with server…
        </div>
      ) : (
        <OnboardingBlock
          phase={phase}
          isStale={isStale}
          onRegister={onRegister}
        />
      )}
    </div>
  );
}

function RegisteredRow({
  runner,
  serviceRunning,
  onClick,
}: {
  runner: ReturnType<typeof useRunners>[number];
  serviceRunning: boolean;
  onClick: () => void;
}) {
  const statusInfo = getRunnerStatusInfo(runner.status);
  return (
    <div
      className={cn(
        "group flex cursor-pointer items-center gap-2 px-3 py-2 hover:bg-muted/50",
      )}
      onClick={onClick}
    >
      <Laptop className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <span className={cn("h-2 w-2 flex-shrink-0 rounded-full", statusInfo.dotColor)} />
          <p className="truncate text-sm font-medium">{runner.node_id}</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>{runner.current_pods}/{runner.max_concurrent_pods} pods</span>
          {runner.host_info?.os && (
            <>
              <span>·</span>
              <span>{runner.host_info.os}</span>
            </>
          )}
          <span>·</span>
          {serviceRunning ? (
            <span className="text-green-600 dark:text-green-400">active</span>
          ) : (
            <span className="text-yellow-600 dark:text-yellow-400">service stopped</span>
          )}
        </div>
      </div>
    </div>
  );
}

function OrphanedBlock({ onReRegister }: { onReRegister: () => void }) {
  return (
    <div className="px-3 pb-3" data-testid="this-mac-orphaned">
      <div className="mb-2 flex items-start gap-1.5 text-xs text-yellow-600 dark:text-yellow-400">
        <AlertTriangle className="mt-0.5 h-3.5 w-3.5 flex-shrink-0" />
        <p>Server doesn&apos;t recognize this runner. It may have been removed remotely.</p>
      </div>
      <Button
        data-testid="this-mac-reregister-btn"
        size="sm"
        variant="default"
        className="h-8 w-full text-xs"
        onClick={onReRegister}
      >
        Re-register This Mac
      </Button>
    </div>
  );
}

function OnboardingBlock({
  phase,
  isStale,
  onRegister,
}: {
  phase: ReturnType<typeof useLocalRunnerOnboarding>["phase"];
  isStale: boolean;
  onRegister: () => void;
}) {
  const busy = phase.kind === "installing";
  const stepLabel = phase.kind === "installing" ? STEP_LABELS[phase.step] : null;
  const errorLine = phase.kind === "error"
    ? phase.step
      ? `Failed at ${STEP_LABELS[phase.step]}`
      : "Failed"
    : null;

  return (
    <div className="px-3 pb-3">
      <p className="mb-2 text-xs text-muted-foreground">
        {isStale
          ? "Old Runner service is installed but registration config is missing."
          : "Run pods locally with no cloud relay round-trip."}
      </p>
      <Button
        data-testid="this-mac-register-btn"
        size="sm"
        variant={phase.kind === "error" ? "outline" : "default"}
        className="w-full h-8 text-xs"
        onClick={onRegister}
        disabled={busy}
      >
        {busy
          ? "Working…"
          : phase.kind === "error"
            ? "Retry"
            : isStale
              ? "Re-register This Mac"
              : "Register This Mac"}
      </Button>
      {stepLabel && (
        <p className="mt-2 truncate text-[11px] text-muted-foreground" title={stepLabel}>
          {stepLabel}…
        </p>
      )}
      {errorLine && phase.kind === "error" && (
        <p
          className="mt-2 truncate text-[11px] text-destructive"
          title={`${errorLine}: ${phase.message}`}
        >
          {errorLine}
        </p>
      )}
    </div>
  );
}
