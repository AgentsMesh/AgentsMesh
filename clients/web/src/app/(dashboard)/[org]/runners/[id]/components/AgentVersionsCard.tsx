"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { Terminal, ArrowUpCircle } from "lucide-react";
import type { RunnerData } from "@/lib/viewModels/runner";
import { upgradeAgent } from "@/lib/api/facade/runnerConnect";
import { listAgents } from "@/lib/api/facade/agentConnect";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";

interface Props {
  runner: RunnerData;
  onUpgraded?: () => void;
}

// Installed agents with version + per-agent upgrade. Merges agent_versions
// (slug + detected version) with available_agents (slug only). The upgrade
// manager (npm/pip) comes from the agent definition (listAgents), not the
// runner probe. `managers === null` means "not loaded yet / fetch failed" — we
// then fall back to showing the button and letting the backend be the authority
// (it rejects unsupported agents), rather than mislabeling every agent
// "unsupported" during the load flicker or on a transient listAgents failure.
export function AgentVersionsCard({ runner, onUpgraded }: Props) {
  const t = useTranslations();
  const params = useParams();
  const orgSlug = String(params.org ?? "");
  const { dialogProps, confirm } = useConfirmDialog();
  const [upgrading, setUpgrading] = useState<Record<string, boolean>>({});
  const [managers, setManagers] = useState<Record<string, string> | null>(null);
  const refreshTimers = useRef<Set<ReturnType<typeof setTimeout>>>(new Set());

  useEffect(() => {
    const timers = refreshTimers.current;
    return () => timers.forEach(clearTimeout);
  }, []);

  useEffect(() => {
    if (!orgSlug) return;
    let cancelled = false;
    listAgents(orgSlug)
      .then((res) => {
        if (cancelled) return;
        const m: Record<string, string> = {};
        for (const a of res.builtin_agents) if (a.upgrade_manager) m[a.slug] = a.upgrade_manager;
        setManagers(m);
      })
      .catch(() => undefined); // keep managers=null → fall back to button + backend authority
    return () => {
      cancelled = true;
    };
  }, [orgSlug]);

  const agents = useMemo(() => {
    const versionMap = new Map((runner.agent_versions ?? []).map((v) => [v.slug, v.version]));
    const slugs = new Set<string>([
      ...(runner.agent_versions ?? []).map((v) => v.slug),
      ...(runner.available_agents ?? []),
    ]);
    return [...slugs].sort().map((slug) => ({ slug, version: versionMap.get(slug) }));
  }, [runner.agent_versions, runner.available_agents]);

  if (agents.length === 0) return null;

  const handleUpgrade = async (slug: string) => {
    const confirmed = await confirm({
      title: t("runners.detail.agentUpgradeDialogTitle"),
      description: t("runners.detail.agentUpgradeConfirm", { agent: slug }),
      confirmText: t("runners.detail.agentUpgrade"),
      cancelText: t("common.cancel"),
    });
    if (!confirmed) return;

    setUpgrading((s) => ({ ...s, [slug]: true }));
    try {
      await upgradeAgent(orgSlug, runner.id, slug);
      toast.success(t("runners.detail.agentUpgradeSent", { agent: slug }));
      // Deferred refresh: version lands via the next heartbeat. Tracked so an
      // unmount (tab switch / navigation) clears it instead of firing on a
      // torn-down tree.
      const tid = setTimeout(() => {
        refreshTimers.current.delete(tid);
        onUpgraded?.();
      }, 3000);
      refreshTimers.current.add(tid);
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("runners.detail.agentUpgradeFailed", { agent: slug })));
    } finally {
      setUpgrading((s) => ({ ...s, [slug]: false }));
    }
  };

  return (
    <div className="bg-card rounded-lg border border-border p-6 md:col-span-2">
      <h3 className="text-lg font-medium text-foreground mb-4">
        {t("runners.detail.agentVersionsTitle")}
      </h3>
      <div className="flex flex-col gap-2">
        {agents.map(({ slug, version }) => {
          const manager = managers?.[slug];
          // null = unknown (loading/failed) → show button, backend is authority.
          const upgradable = managers === null || manager !== undefined;
          return (
            <div
              key={slug}
              className="flex items-center justify-between rounded-md border border-border/60 px-3 py-2"
            >
              <span className="flex items-center gap-2 text-sm text-foreground">
                <Terminal className="w-4 h-4 text-blue-500" />
                <span className="font-medium">{slug}</span>
                <span className="text-muted-foreground">
                  {version ? `v${version}` : t("runners.detail.agentVersionUnknown")}
                </span>
                {manager && (
                  <span className="inline-flex items-center rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                    {manager}
                  </span>
                )}
              </span>
              {upgradable ? (
                <button
                  onClick={() => handleUpgrade(slug)}
                  disabled={runner.status !== "online" || upgrading[slug]}
                  title={
                    runner.status !== "online"
                      ? t("runners.detail.upgradeOffline")
                      : t("runners.detail.agentUpgradeTooltip", { agent: slug })
                  }
                  className="inline-flex items-center gap-1 rounded-md bg-primary px-2.5 py-1 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <ArrowUpCircle className="h-3.5 w-3.5" />
                  {upgrading[slug] ? t("runners.detail.agentUpgrading") : t("runners.detail.agentUpgrade")}
                </button>
              ) : (
                <span className="text-xs text-muted-foreground">
                  {t("runners.detail.agentUpgradeUnsupported")}
                </span>
              )}
            </div>
          );
        })}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
